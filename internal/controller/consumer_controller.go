/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/jsm.go"
	jsmapi "github.com/nats-io/jsm.go/api"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ConsumerReconciler reconciles a Consumer object
type ConsumerReconciler struct {
	Scheme *runtime.Scheme

	JetStreamController
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *ConsumerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := klog.FromContext(ctx)

	if ok := r.ValidNamespace(req.Namespace); !ok {
		log.Info("Controller restricted to namespace, skipping reconciliation.")
		return ctrl.Result{}, nil
	}

	// Fetch consumer resource
	consumer := &api.Consumer{}
	if err := r.Get(ctx, req.NamespacedName, consumer); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Consumer resource deleted.", "consumerName", req.NamespacedName.String())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get consumer resource '%s': %w", req.NamespacedName.String(), err)
	}

	log = log.WithValues(
		"streamName", consumer.Spec.StreamName,
		"consumerName", consumer.Spec.DurableName,
	)

	// Update ready status to unknown when no status is set
	if len(consumer.Status.Conditions) == 0 {
		log.Info("Setting initial ready condition to unknown.")
		consumer.Status.Conditions = updateReadyCondition(consumer.Status.Conditions, v1.ConditionUnknown, stateReconciling, "Starting reconciliation")
		err := r.Status().Update(ctx, consumer)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("set condition unknown: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Check Deletion
	markedForDeletion := consumer.GetDeletionTimestamp() != nil
	if markedForDeletion {
		if controllerutil.ContainsFinalizer(consumer, consumerFinalizer) {
			err := r.deleteConsumer(ctx, log, consumer)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("delete consumer: %w", err)
			}
		} else {
			log.Info("Consumer marked for deletion and already finalized. Ignoring.")
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(consumer, consumerFinalizer) {
		log.Info("Adding consumer finalizer.")
		if ok := controllerutil.AddFinalizer(consumer, consumerFinalizer); !ok {
			return ctrl.Result{}, errors.New("failed to add finalizer to consumer resource")
		}

		if err := r.Update(ctx, consumer); err != nil {
			return ctrl.Result{}, fmt.Errorf("update consumer resource to add finalizer: %w", err)
		}
		return ctrl.Result{}, nil
	}

	// Create or update stream
	if err := r.createOrUpdate(ctx, log, consumer); err != nil {
		if err := r.Get(ctx, client.ObjectKeyFromObject(consumer), consumer); err != nil {
			return ctrl.Result{}, fmt.Errorf("get consumer resource: %w", err)
		}
		consumer.Status.Conditions = updateReadyCondition(consumer.Status.Conditions, v1.ConditionFalse, stateErrored, err.Error())
		if err := r.Status().Update(ctx, consumer); err != nil {
			log.Error(err, "Failed to update ready condition to Errored.")
		}
		return ctrl.Result{}, fmt.Errorf("create or update: %s", err)
	}

	return ctrl.Result{RequeueAfter: r.RequeueInterval()}, nil
}

func (r *ConsumerReconciler) deleteConsumer(ctx context.Context, log logr.Logger, consumer *api.Consumer) error {
	// Set status to false
	consumer.Status.Conditions = updateReadyCondition(consumer.Status.Conditions, v1.ConditionFalse, stateFinalizing, "Performing finalizer operations.")
	if err := r.Status().Update(ctx, consumer); err != nil {
		return fmt.Errorf("update ready condition: %w", err)
	}

	storedState, err := getStoredConsumerState(consumer)
	if err != nil {
		log.Error(err, "Failed to fetch stored state.")
	}

	if !consumer.Spec.PreventDelete && !r.ReadOnly() {
		err := r.WithJSMClient(consumer.Spec.ConnectionOpts, consumer.Namespace, func(js *jsm.Manager) error {
			_, err := getServerConsumerState(js, consumer)
			// If we have no known state for this consumer it has never been reconciled.
			// If we are also receiving an error fetching state, either the consumer does not exist
			// or this resource config is invalid.
			if err != nil && storedState == nil {
				return nil
			}

			return js.DeleteConsumer(consumer.Spec.StreamName, consumer.Spec.DurableName)
		})
		switch {
		case jsm.IsNatsError(err, JSConsumerNotFoundErr):
			log.Info("Consumer does not exist. Unable to delete.")
		case jsm.IsNatsError(err, JSStreamNotFoundErr):
			log.Info("Stream of consumer does not exist. Unable to delete.")
		case err != nil:
			if storedState == nil {
				log.Info("Consumer not reconciled and no state received from server. Removing finalizer.")
			} else {
				return fmt.Errorf("delete jetstream consumer: %w", err)
			}
		default:
			log.Info("Consumer deleted.")
		}
	} else {
		log.Info("Skipping consumer deletion.",
			"consumerName", consumer.Spec.DurableName,
			"preventDelete", consumer.Spec.PreventDelete,
			"read-only", r.ReadOnly(),
		)
	}

	log.Info("Removing consumer finalizer.")
	if ok := controllerutil.RemoveFinalizer(consumer, consumerFinalizer); !ok {
		return errors.New("failed to remove consumer finalizer")
	}
	if err := r.Update(ctx, consumer); err != nil {
		return fmt.Errorf("remove finalizer: %w", err)
	}

	return nil
}

func (r *ConsumerReconciler) createOrUpdate(ctx context.Context, log klog.Logger, consumer *api.Consumer) error {
	// Create or Update the stream based on the spec
	// Map spec to consumer target config
	targetConfig, err := consumerSpecToConfig(&consumer.Spec)
	if err != nil {
		return fmt.Errorf("map consumer spec to target config: %w", err)
	}

	err = r.WithJSMClient(consumer.Spec.ConnectionOpts, consumer.Namespace, func(js *jsm.Manager) error {
		storedState, err := getStoredConsumerState(consumer)
		if err != nil {
			log.Error(err, "Failed to fetch stored consumer state.")
		}

		serverState, err := getServerConsumerState(js, consumer)
		if err != nil {
			return fmt.Errorf("fetching consumer current state: %w", err)
		}

		// Check against known state. Skip Update if converged.
		// Storing returned state from the server avoids have to
		// check default values or call Update on already converged resources
		if storedState != nil && serverState != nil && consumer.Status.ObservedGeneration == consumer.Generation {
			diff := compareConfigState(storedState, serverState)

			if diff == "" {
				return nil
			}

			log.Info("Consumer config drifted from desired state.", "diff", diff)
		}

		if r.ReadOnly() {
			log.Info("Skipping consumer creation or update.",
				"read-only", r.ReadOnly(),
			)
			return nil
		}

		var updatedConsumer *jsm.Consumer
		err = nil

		if serverState == nil {
			log.Info("Creating Consumer.")
			updatedConsumer, err = js.NewConsumer(consumer.Spec.StreamName, targetConfig...)
			if err != nil {
				return fmt.Errorf("creating consumer: %w", err)
			}
		} else if !consumer.Spec.PreventUpdate {
			log.Info("Updating Consumer.")
			c, err := js.LoadConsumer(consumer.Spec.StreamName, consumer.Spec.DurableName)
			if err != nil {
				return fmt.Errorf("loading consumer: %w", err)
			}

			err = c.UpdateConfiguration(targetConfig...)
			if err != nil {
				return fmt.Errorf("updating the consumer configuration: %w", err)
			}

			updatedConsumer, err = js.LoadConsumer(consumer.Spec.StreamName, consumer.Spec.DurableName)
			if err != nil {
				return fmt.Errorf("loading updated consumer: %w", err)
			}

			diff := compareConfigState(updatedConsumer.Configuration(), *serverState)
			log.Info("Updated Consumer.", "diff", diff)
		} else {
			log.Info("Skipping Consumer update.",
				"preventUpdate", consumer.Spec.PreventUpdate,
			)
		}

		if updatedConsumer != nil {
			// Store known state in annotation
			updatedState, err := json.Marshal(updatedConsumer.Configuration())
			if err != nil {
				return fmt.Errorf("marshaling JSON: %w", err)
			}

			if consumer.Annotations == nil {
				consumer.Annotations = map[string]string{}
			}
			consumer.Annotations[stateAnnotationConsumer] = string(updatedState)

			return r.Update(ctx, consumer)
		}

		return nil
	})
	if err != nil {
		err = fmt.Errorf("create or update consumer: %w", err)
		consumer.Status.Conditions = updateReadyCondition(consumer.Status.Conditions, v1.ConditionFalse, stateErrored, err.Error())
		if err := r.Status().Update(ctx, consumer); err != nil {
			log.Error(err, "Failed to update ready condition to Errored.")
		}
		return err
	}

	// update the observed generation and ready status
	consumer.Status.ObservedGeneration = consumer.Generation
	consumer.Status.Conditions = updateReadyCondition(
		consumer.Status.Conditions,
		v1.ConditionTrue,
		stateReady,
		"Consumer successfully created or updated.",
	)
	err = r.Status().Update(ctx, consumer)
	if err != nil {
		return fmt.Errorf("update ready condition: %w", err)
	}

	return nil
}

func getStoredConsumerState(consumer *api.Consumer) (*jsmapi.ConsumerConfig, error) {
	var storedState *jsmapi.ConsumerConfig
	if state, ok := consumer.Annotations[stateAnnotationConsumer]; ok {
		err := json.Unmarshal([]byte(state), &storedState)
		if err != nil {
			return nil, err
		}
	}

	return storedState, nil
}

// Fetch the current state of the consumer from the server.
// ErrConsumerNotFound is considered a valid response and does not return error
func getServerConsumerState(js *jsm.Manager, consumer *api.Consumer) (*jsmapi.ConsumerConfig, error) {
	c, err := js.LoadConsumer(consumer.Spec.StreamName, consumer.Spec.DurableName)
	if jsm.IsNatsError(err, JSConsumerNotFoundErr) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	consumerCfg := c.Configuration()
	return &consumerCfg, nil
}

func consumerSpecToConfig(spec *api.ConsumerSpec) ([]jsm.ConsumerOption, error) {
	opts := []jsm.ConsumerOption{
		jsm.ConsumerDescription(spec.Description),
		jsm.DeliverySubject(spec.DeliverSubject),
		jsm.DeliverGroup(spec.DeliverGroup),
		jsm.DurableName(spec.DurableName),
		jsm.MaxAckPending(uint(spec.MaxAckPending)),
		jsm.MaxWaiting(uint(spec.MaxWaiting)),
		jsm.RateLimitBitsPerSecond(uint64(spec.RateLimitBps)),
		jsm.MaxRequestBatch(uint(spec.MaxRequestBatch)),
		jsm.MaxRequestMaxBytes(spec.MaxRequestMaxBytes),
		jsm.ConsumerOverrideReplicas(spec.Replicas),
		jsm.ConsumerMetadata(spec.Metadata),
	}

	// ackPolicy
	switch spec.AckPolicy {
	case "none":
		opts = append(opts, jsm.AcknowledgeNone())
	case "all":
		opts = append(opts, jsm.AcknowledgeAll())
	case "explicit":
		opts = append(opts, jsm.AcknowledgeExplicit())
	case "":
	default:
		return nil, fmt.Errorf("invalid value for 'ackPolicy': '%s'. Must be one of 'none', 'all', 'explicit'", spec.AckPolicy)
	}

	//	ackWait
	if spec.AckWait != "" {
		d, err := time.ParseDuration(spec.AckWait)
		if err != nil {
			return nil, fmt.Errorf("invalid ack wait duration: %w", err)
		}
		opts = append(opts, jsm.AckWait(d))
	}

	// deliverPolicy
	switch spec.DeliverPolicy {
	case "all":
		opts = append(opts, jsm.DeliverAllAvailable())
	case "last":
		opts = append(opts, jsm.StartWithLastReceived())
	case "new":
		opts = append(opts, jsm.StartWithNextReceived())
	case "byStartSequence":
		opts = append(opts, jsm.StartAtSequence(uint64(spec.OptStartSeq)))
	case "byStartTime":
		if spec.OptStartTime == "" {
			return nil, fmt.Errorf("'optStartTime' is required for deliver policy 'byStartTime'")
		}
		t, err := time.Parse(time.RFC3339, spec.OptStartTime)
		if err != nil {
			return nil, err
		}
		opts = append(opts, jsm.StartAtTime(t))
	case "lastPerSubject":
		opts = append(opts, jsm.DeliverLastPerSubject())
	case "":
	default:
		return nil, fmt.Errorf("invalid value for 'deliverPolicy': '%s'. Must be one of 'all', 'last', 'new', 'lastPerSubject', 'byStartSequence', 'byStartTime'", spec.DeliverPolicy)
	}

	// filterSubject
	if spec.FilterSubject != "" && len(spec.FilterSubjects) > 0 {
		return nil, errors.New("cannot set both 'filterSubject' and 'filterSubjects'")
	}

	if spec.FilterSubject != "" {
		opts = append(opts, jsm.FilterStreamBySubject(spec.FilterSubject))
	} else if len(spec.FilterSubjects) > 0 {
		opts = append(opts, jsm.FilterStreamBySubject(spec.FilterSubjects...))
	}

	// flowControl
	if spec.FlowControl {
		opts = append(opts, jsm.PushFlowControl())
	}

	// heartbeatInterval
	if spec.HeartbeatInterval != "" {
		d, err := time.ParseDuration(spec.HeartbeatInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid heartbeat interval: %w", err)
		}

		opts = append(opts, jsm.IdleHeartbeat(d))
	}

	// maxDeliver
	if spec.MaxDeliver != 0 {
		opts = append(opts, jsm.MaxDeliveryAttempts(spec.MaxDeliver))
	}

	// backoff
	if len(spec.BackOff) > 0 {
		backoffs := make([]time.Duration, 0)
		for _, bo := range spec.BackOff {
			d, err := time.ParseDuration(bo)
			if err != nil {
				return nil, fmt.Errorf("invalid backoff: %w", err)
			}
			backoffs = append(backoffs, d)
		}

		opts = append(opts, jsm.BackoffIntervals(backoffs...))
	}

	//	replayPolicy
	switch spec.ReplayPolicy {
	case "instant":
		opts = append(opts, jsm.ReplayInstantly())
	case "original":
		opts = append(opts, jsm.ReplayAsReceived())
	case "":
	default:
		return nil, fmt.Errorf("invalid value for 'replayPolicy': '%s'. Must be one of 'instant', 'original'", spec.ReplayPolicy)
	}

	if spec.SampleFreq != "" {
		n, err := strconv.Atoi(
			strings.TrimSuffix(spec.SampleFreq, "%"),
		)
		if err != nil {
			return nil, err
		}
		opts = append(opts, jsm.SamplePercent(n))
	}

	if spec.HeadersOnly {
		opts = append(opts, jsm.DeliverHeadersOnly())
	}

	//	MaxRequestExpires
	if spec.MaxRequestExpires != "" {
		d, err := time.ParseDuration(spec.MaxRequestExpires)
		if err != nil {
			return nil, fmt.Errorf("invalid opt start time: %w", err)
		}
		opts = append(opts, jsm.MaxRequestExpires(d))
	}

	// inactiveThreshold
	if spec.InactiveThreshold != "" {
		d, err := time.ParseDuration(spec.InactiveThreshold)
		if err != nil {
			return nil, fmt.Errorf("invalid inactive threshold: %w", err)
		}
		opts = append(opts, jsm.InactiveThreshold(d))
	}

	// memStorage
	if spec.MemStorage {
		opts = append(opts, jsm.ConsumerOverrideMemoryStorage())
	}

	// Handle PauseUntil for pausing consumer
	if spec.PauseUntil != "" {
		t, err := time.Parse(time.RFC3339, spec.PauseUntil)
		if err != nil {
			return nil, fmt.Errorf("invalid pauseUntil time: %w", err)
		}
		opts = append(opts, jsm.PauseUntil(t))
	}

	// Handle PriorityPolicy with PriorityGroups and PinnedTTL
	switch spec.PriorityPolicy {
	case "", "none":
		// Default is none, no need to set
	case "pinned_client", "pinned":
		if spec.PinnedTTL != "" {
			dur, err := time.ParseDuration(spec.PinnedTTL)
			if err != nil {
				return nil, fmt.Errorf("invalid pinnedTTL duration: %w", err)
			}
			opts = append(opts, jsm.PinnedClientPriorityGroups(dur, spec.PriorityGroups...))
		}
	case "overflow":
		opts = append(opts, jsm.OverflowPriorityGroups(spec.PriorityGroups...))
	case "prioritized":
		opts = append(opts, jsm.PrioritizedPriorityGroups(spec.PriorityGroups...))
	default:
		return nil, fmt.Errorf("invalid priority policy: %s", spec.PriorityPolicy)
	}

	return opts, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConsumerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Consumer{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}
