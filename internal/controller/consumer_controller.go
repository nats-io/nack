/*
Copyright 2024.

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
	"time"

	"github.com/go-logr/logr"
	"github.com/nats-io/nats.go/jetstream"
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
			log.Info("Consumer deleted.", "consumerName", req.NamespacedName.String())
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

	if consumer.Spec.FlowControl || consumer.Spec.DeliverSubject != "" || consumer.Spec.DeliverGroup != "" || consumer.Spec.HeartbeatInterval != "" {
		log.Info("FlowControl, DeliverSubject, DeliverGroup, and HeartbeatInterval are Push Consumer options, which are not supported. Skipping consumer creation or update.")
		consumer.Status.Conditions = updateReadyCondition(consumer.Status.Conditions, v1.ConditionFalse, stateErrored, "Push Consumer options are not supported.")
		if err := r.Status().Update(ctx, consumer); err != nil {
			log.Error(err, "Failed to update ready condition to Errored.")
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
		err := r.WithJetStreamClient(consumer.Spec.ConnectionOpts, consumer.Namespace, func(js jetstream.JetStream) error {
			_, err := getServerConsumerState(ctx, js, consumer)
			// If we have no known state for this consumer it has never been reconciled.
			// If we are also receiving an error fetching state, either the consumer does not exist
			// or this resource config is invalid.
			if err != nil && storedState == nil {
				return nil
			}

			return js.DeleteConsumer(ctx, consumer.Spec.StreamName, consumer.Spec.DurableName)
		})
		switch {
		case errors.Is(err, jetstream.ErrConsumerNotFound):
			log.Info("Consumer does not exist. Unable to delete.")
		case errors.Is(err, jetstream.ErrStreamNotFound):
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

	err = r.WithJetStreamClient(consumer.Spec.ConnectionOpts, consumer.Namespace, func(js jetstream.JetStream) error {
		storedState, err := getStoredConsumerState(consumer)
		if err != nil {
			log.Error(err, "Failed to fetch stored consumer state.")
		}

		serverState, err := getServerConsumerState(ctx, js, consumer)
		if err != nil {
			return err
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

		var updatedConsumer jetstream.Consumer
		err = nil

		if serverState == nil {
			log.Info("Creating Consumer.")
			updatedConsumer, err = js.CreateConsumer(ctx, consumer.Spec.StreamName, *targetConfig)
			if err != nil {
				return err
			}
		} else if !consumer.Spec.PreventUpdate {
			log.Info("Updating Consumer.")
			updatedConsumer, err = js.UpdateConsumer(ctx, consumer.Spec.StreamName, *targetConfig)
			if err != nil {
				return err
			}
		} else {
			log.Info("Skipping Consumer update.",
				"preventUpdate", consumer.Spec.PreventUpdate,
			)
		}

		if updatedConsumer != nil {
			// Store known state in annotation
			updatedState, err := json.Marshal(updatedConsumer.CachedInfo().Config)
			if err != nil {
				return err
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

func getStoredConsumerState(consumer *api.Consumer) (*jetstream.ConsumerConfig, error) {
	var storedState *jetstream.ConsumerConfig
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
func getServerConsumerState(ctx context.Context, js jetstream.JetStream, consumer *api.Consumer) (*jetstream.ConsumerConfig, error) {
	c, err := js.Consumer(ctx, consumer.Spec.StreamName, consumer.Spec.DurableName)
	if errors.Is(err, jetstream.ErrConsumerNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &c.CachedInfo().Config, nil
}

func consumerSpecToConfig(spec *api.ConsumerSpec) (*jetstream.ConsumerConfig, error) {
	config := &jetstream.ConsumerConfig{
		Durable:            spec.DurableName,
		Description:        spec.Description,
		OptStartSeq:        uint64(spec.OptStartSeq),
		MaxDeliver:         spec.MaxDeliver,
		FilterSubject:      spec.FilterSubject,
		RateLimit:          uint64(spec.RateLimitBps),
		SampleFrequency:    spec.SampleFreq,
		MaxWaiting:         spec.MaxWaiting,
		MaxAckPending:      spec.MaxAckPending,
		HeadersOnly:        spec.HeadersOnly,
		MaxRequestBatch:    spec.MaxRequestBatch,
		MaxRequestMaxBytes: spec.MaxRequestMaxBytes,
		Replicas:           spec.Replicas,
		MemoryStorage:      spec.MemStorage,
		FilterSubjects:     spec.FilterSubjects,
		Metadata:           spec.Metadata,
	}

	// DeliverPolicy
	if spec.DeliverPolicy != "" {
		err := config.DeliverPolicy.UnmarshalJSON(jsonString(spec.DeliverPolicy))
		if err != nil {
			return nil, fmt.Errorf("invalid delivery policy: %w", err)
		}
	}

	//	OptStartTime RFC3339
	if spec.OptStartTime != "" {
		t, err := time.Parse(time.RFC3339, spec.OptStartTime)
		if err != nil {
			return nil, fmt.Errorf("invalid opt start time: %w", err)
		}
		config.OptStartTime = &t
	}

	//	AckPolicy
	if spec.AckPolicy != "" {
		err := config.AckPolicy.UnmarshalJSON(jsonString(spec.AckPolicy))
		if err != nil {
			return nil, fmt.Errorf("invalid ack policy: %w", err)
		}
	}

	//	AckWait
	if spec.AckWait != "" {
		d, err := time.ParseDuration(spec.AckWait)
		if err != nil {
			return nil, fmt.Errorf("invalid ack wait duration: %w", err)
		}
		config.AckWait = d
	}

	// BackOff
	for _, bo := range spec.BackOff {
		d, err := time.ParseDuration(bo)
		if err != nil {
			return nil, fmt.Errorf("invalid backoff: %w", err)
		}

		config.BackOff = append(config.BackOff, d)
	}

	//	ReplayPolicy
	if spec.ReplayPolicy != "" {
		err := config.ReplayPolicy.UnmarshalJSON(jsonString(spec.ReplayPolicy))
		if err != nil {
			return nil, fmt.Errorf("invalid replay policy: %w", err)
		}
	}

	//	MaxRequestExpires
	if spec.MaxRequestExpires != "" {
		d, err := time.ParseDuration(spec.MaxRequestExpires)
		if err != nil {
			return nil, fmt.Errorf("invalid opt start time: %w", err)
		}
		config.MaxRequestExpires = d
	}

	if spec.InactiveThreshold != "" {
		d, err := time.ParseDuration(spec.InactiveThreshold)
		if err != nil {
			return nil, fmt.Errorf("invalid inactive threshold: %w", err)
		}
		config.InactiveThreshold = d
	}

	return config, nil
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
