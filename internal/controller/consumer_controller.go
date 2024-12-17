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
	"errors"
	"fmt"
	"github.com/nats-io/nats.go/jetstream"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ConsumerReconciler reconciles a Consumer object
type ConsumerReconciler struct {
	JetStreamController
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *ConsumerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := klog.FromContext(ctx)

	// Fetch consumer resource
	consumer := &api.Consumer{}
	if err := r.Get(ctx, req.NamespacedName, consumer); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Consumer resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get consumer resource '%s': %w", req.NamespacedName.String(), err)
	}

	// Update ready status to unknown when no status is set
	if consumer.Status.Conditions == nil || len(consumer.Status.Conditions) == 0 {
		log.Info("Setting initial ready condition to unknown.")
		consumer.Status.Conditions = updateReadyCondition(consumer.Status.Conditions, v1.ConditionUnknown, "Reconciling", "Starting reconciliation")
		err := r.Status().Update(ctx, consumer)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("set condition unknown: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
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
		return ctrl.Result{}, fmt.Errorf("create or update: %s", err)
	}
	return ctrl.Result{}, nil
}

func (r *ConsumerReconciler) createOrUpdate(ctx context.Context, log klog.Logger, consumer *api.Consumer) error {

	targetConfig, err := consumerSpecToConfig(&consumer.Spec)
	if err != nil {
		return fmt.Errorf("map consumer spec to target config: %w", err)
	}

	err = r.WithJetStreamClient(consumerConnOpts(consumer.Spec), func(js jetstream.JetStream) error {
		log.Info("create or update consumer", "streamName", consumer.Spec.StreamName, "name", consumer.Spec.DurableName)
		_, err := js.CreateOrUpdateConsumer(ctx, consumer.Spec.StreamName, *targetConfig)
		return err
	})
	if err != nil {
		err = fmt.Errorf("create or update consumer: %w", err)
		consumer.Status.Conditions = updateReadyCondition(consumer.Status.Conditions, v1.ConditionFalse, "Errored", err.Error())
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
		"Reconciling",
		"Stream successfully created or updated.",
	)
	err = r.Status().Update(ctx, consumer)
	if err != nil {
		return fmt.Errorf("update ready condition: %w", err)
	}

	return nil
}

func consumerConnOpts(spec api.ConsumerSpec) *connectionOptions {
	return &connectionOptions{
		Account: spec.Account,
		Creds:   spec.Creds,
		Nkey:    spec.Nkey,
		Servers: spec.Servers,
		TLS:     spec.TLS,
	}
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

		// Explicitly set not (yet) mapped fields
		Name:              "",
		InactiveThreshold: 0,
	}

	// DeliverPolicy
	if spec.DeliverPolicy != "" {
		err := config.DeliverPolicy.UnmarshalJSON(asJsonString(spec.DeliverPolicy))
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
		err := config.AckPolicy.UnmarshalJSON(asJsonString(spec.AckPolicy))
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

	//BackOff
	for _, bo := range spec.BackOff {
		d, err := time.ParseDuration(bo)
		if err != nil {
			return nil, fmt.Errorf("invalid backoff: %w", err)
		}

		config.BackOff = append(config.BackOff, d)
	}

	//	ReplayPolicy
	if spec.ReplayPolicy != "" {
		err := config.ReplayPolicy.UnmarshalJSON(asJsonString(spec.ReplayPolicy))
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

	return config, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConsumerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Consumer{}).
		Complete(r)
}
