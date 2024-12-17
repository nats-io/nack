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
	"fmt"
	"github.com/nats-io/nats.go/jetstream"
	"k8s.io/klog/v2"
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
	log.Info("reconcile", "namespace", req.Namespace, "name", req.Name)

	return ctrl.Result{}, nil
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
