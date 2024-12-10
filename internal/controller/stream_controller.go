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
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/nats-io/nats.go/jetstream"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

// StreamReconciler reconciles a Stream object
type StreamReconciler struct {
	JetStreamController
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// It performs three main operations:
// - Initialize finalizer and ready condition if not present
// - Delete stream if it is marked for deletion.
// - Create or Update the stream
func (r *StreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := klog.FromContext(ctx).
		WithName("StreamReconciler").
		WithValues("namespace", req.Namespace, "name", req.Name)

	log.Info("reconciling")

	if ok := r.ValidNamespace(req.Namespace); !ok {
		log.Info("Controller restricted to namespace, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Fetch stream resource
	stream := &api.Stream{}
	if err := r.Get(ctx, req.NamespacedName, stream); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("stream resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get stream resource '%s': %w", req.NamespacedName.String(), err)
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(stream, streamFinalizer) {
		log.Info("Adding stream finalizer")
		if ok := controllerutil.AddFinalizer(stream, streamFinalizer); !ok {
			log.Error(nil, "Failed to add finalizer to stream resource")
			return ctrl.Result{Requeue: true}, nil
		}

		if err := r.Update(ctx, stream); err != nil {
			return ctrl.Result{}, fmt.Errorf("update stream resource to add finalizer: %w", err)
		}

		// re-fetch stream
		if err := r.Get(ctx, req.NamespacedName, stream); err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("stream resource not found. Ignoring since object must be deleted")
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, fmt.Errorf("re-fetch stream resource: %w", err)
		}
	}

	// Update ready status to unknown when no status is set
	if stream.Status.Conditions == nil || len(stream.Status.Conditions) == 0 {
		log.Info("Setting initial ready condition to unknown")
		stream.Status.Conditions = updateReadyCondition(stream.Status.Conditions, v1.ConditionUnknown, "Reconciling", "Starting reconciliation")
		err := r.Status().Update(ctx, stream)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("set condition unknown: %w", err)
		}

		// re-fetch stream
		if err := r.Get(ctx, req.NamespacedName, stream); err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("stream resource not found. Ignoring since object must be deleted")
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, fmt.Errorf("re-fetch stream resource: %w", err)
		}
	}

	if r.ReadOnly() {
		log.Info("read-only enabled, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Connection options specific to this spec
	specConnectionOptions := &connectionOptions{
		Account: stream.Spec.Account, // TODO(review): Where does Spec.Account have to be considered?
		Creds:   stream.Spec.Creds,
		Nkey:    stream.Spec.Nkey,
		Servers: stream.Spec.Servers,
		TLS:     stream.Spec.TLS,
	}

	// Delete if marked for deletion and has stream finalizer
	markedForDeletion := stream.GetDeletionTimestamp() != nil
	// TODO deletion is triggered multiple times?. Set additional status condition?
	if markedForDeletion && controllerutil.ContainsFinalizer(stream, streamFinalizer) {
		// Set status to not unknown
		stream.Status.Conditions = updateReadyCondition(stream.Status.Conditions, v1.ConditionUnknown, "Finalizing", "Performing finalizer operations.")
		if err := r.Status().Update(ctx, stream); err != nil {
			return ctrl.Result{}, fmt.Errorf("update ready condition: %w", err)
		}

		log.Info("performing finalizing operations")
		if stream.Spec.PreventDelete {
			log.Info("skip delete stream during resource deletion.", "streamName", stream.Spec.Name)
		} else {
			// Remove stream
			err := r.WithJetStreamClient(specConnectionOptions, func(js jetstream.JetStream) error {
				return js.DeleteStream(ctx, stream.Spec.Name)
			})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("delete stream during finalization: %w", err)
			}
		}

		// Re-Fetch resource
		if err := r.Get(ctx, req.NamespacedName, stream); err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("stream resource not found. Ignoring since object must be deleted")
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, fmt.Errorf("re-fetch stream resource: %w", err)
		}

		// Update status to not ready
		stream.Status.Conditions = updateReadyCondition(stream.Status.Conditions, v1.ConditionFalse, "Finalizing", "Performed finalizer operations.")
		if err := r.Status().Update(ctx, stream); err != nil {
			return ctrl.Result{}, fmt.Errorf("update ready condition: %w", err)
		}

		if err := r.Status().Update(ctx, stream); err != nil {
			return ctrl.Result{}, fmt.Errorf("update ready condition: %w", err)
		}

		log.Info("Removing stream finalizer after performing finalizing operations")
		if ok := controllerutil.RemoveFinalizer(stream, streamFinalizer); !ok {
			return ctrl.Result{Requeue: true}, errors.New("failed to remove stream finalizer")
		}

		if err := r.Update(ctx, stream); err != nil {
			return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
		}

		return ctrl.Result{}, nil
	}

	// Create or Update the stream based on the spec
	if stream.Spec.PreventUpdate {
		log.Info("Spec.PreventUpdate: skip create/updating the stream during reconciliation.", "streamName", stream.Spec.Name)
		return ctrl.Result{}, nil
	}

	// Map spec to stream targetConfig
	targetConfig, err := mapSpecToConfig(&stream.Spec)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("map spec to stream targetConfig: %w", err)
	}

	// CreateOrUpdateStream is called on every reconciliation when the stream is not to be deleted.
	// TODO(future-feature): Do we need to check if generation has changed or the config differs?
	err = r.WithJetStreamClient(specConnectionOptions, func(js jetstream.JetStream) error {
		_, err = js.CreateOrUpdateStream(ctx, targetConfig)
		return err
	})
	if err != nil {
		msg := fmt.Sprintf("create or update stream: %s", err.Error())
		stream.Status.Conditions = updateReadyCondition(stream.Status.Conditions, v1.ConditionFalse, "Errored", msg)
		if err := r.Status().Update(ctx, stream); err != nil {
			log.Error(err, "failed to update ready condition to Errored")
		}
		return ctrl.Result{}, fmt.Errorf("create or update stream: %w", err)
	}

	// update the observed generation and ready status
	stream.Status.ObservedGeneration = stream.Generation
	stream.Status.Conditions = updateReadyCondition(
		stream.Status.Conditions,
		v1.ConditionTrue,
		"Reconciling",
		"Stream successfully created or updated",
	)
	err = r.Status().Update(ctx, stream)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("update ready condition: %w", err)
	}

	return ctrl.Result{}, nil
}

// mapSpecToConfig creates a jetstream.StreamConfig matching the given stream resource spec
func mapSpecToConfig(spec *api.StreamSpec) (jetstream.StreamConfig, error) {

	// Set directly mapped fields
	config := jetstream.StreamConfig{
		Name:                 spec.Name,
		Description:          spec.Description,
		Subjects:             spec.Subjects,
		MaxConsumers:         spec.MaxConsumers,
		MaxMsgs:              int64(spec.MaxMsgs),
		MaxBytes:             int64(spec.MaxBytes),
		DiscardNewPerSubject: spec.DiscardPerSubject,
		MaxMsgsPerSubject:    int64(spec.MaxMsgsPerSubject),
		MaxMsgSize:           int32(spec.MaxMsgSize),
		Replicas:             spec.Replicas,
		NoAck:                spec.NoAck,
		DenyDelete:           spec.DenyDelete,
		DenyPurge:            spec.DenyPurge,
		AllowRollup:          spec.AllowRollup,
		FirstSeq:             spec.FirstSequence,
		AllowDirect:          spec.AllowDirect,
		// Explicitly set not (yet) mapped fields
		Sealed:         false,
		MirrorDirect:   false,
		ConsumerLimits: jetstream.StreamConsumerLimits{},
	}

	// Set not directly mapped fields

	// retention
	if spec.Retention != "" {
		// Wrap string in " to be properly unmarshalled as json string
		err := config.Retention.UnmarshalJSON(asJsonString(spec.Retention))
		if err != nil {
			return jetstream.StreamConfig{}, fmt.Errorf("invalid retention policy: %w", err)
		}
	}

	// discard
	if spec.Discard != "" {
		err := config.Discard.UnmarshalJSON(asJsonString(spec.Discard))
		if err != nil {
			return jetstream.StreamConfig{}, fmt.Errorf("invalid retention policy: %w", err)
		}
	}

	// maxAge
	if spec.MaxAge != "" {
		d, err := time.ParseDuration(spec.MaxAge)
		if err != nil {
			return jetstream.StreamConfig{}, fmt.Errorf("parse max age: %w", err)
		}
		config.MaxAge = d
	}
	// storage
	if spec.Storage != "" {
		err := config.Storage.UnmarshalJSON(asJsonString(spec.Storage))
		if err != nil {
			return jetstream.StreamConfig{}, fmt.Errorf("invalid storage: %w", err)
		}
	}

	// duplicates
	if spec.DuplicateWindow != "" {
		d, err := time.ParseDuration(spec.DuplicateWindow)
		if err != nil {
			return jetstream.StreamConfig{}, fmt.Errorf("parse duplicate window: %w", err)
		}
		config.Duplicates = d
	}

	// placement
	if spec.Placement != nil {
		config.Placement = &jetstream.Placement{
			Cluster: spec.Placement.Cluster,
			Tags:    spec.Placement.Tags,
		}
	}

	// mirror
	if spec.Mirror != nil {
		ss, err := mapStreamSource(spec.Mirror)
		if err != nil {
			return jetstream.StreamConfig{}, fmt.Errorf("map mirror stream soruce: %w", err)
		}
		config.Mirror = ss
	}

	// sources
	if spec.Sources != nil {
		config.Sources = []*jetstream.StreamSource{}
		for _, source := range spec.Sources {
			s, err := mapStreamSource(source)
			if err != nil {
				return jetstream.StreamConfig{}, fmt.Errorf("map stream soruce: %w", err)
			}
			config.Sources = append(config.Sources, s)
		}
	}

	// compression
	if spec.Compression != "" {
		err := config.Compression.UnmarshalJSON(asJsonString(spec.Compression))
		if err != nil {
			return jetstream.StreamConfig{}, fmt.Errorf("invalid compression: %w", err)
		}
	}

	// subjectTransform
	if spec.SubjectTransform != nil {
		config.SubjectTransform = &jetstream.SubjectTransformConfig{
			Source:      spec.SubjectTransform.Source,
			Destination: spec.SubjectTransform.Dest,
		}
	}

	// rePublish
	if spec.Republish != nil {
		config.RePublish = &jetstream.RePublish{
			Source:      spec.Republish.Source,
			Destination: spec.Republish.Destination,
			HeadersOnly: spec.Republish.HeadersOnly,
		}
	}

	// metadata
	if spec.Metadata != nil {
		config.Metadata = spec.Metadata
	}

	return config, nil
}

func mapStreamSource(ss *api.StreamSource) (*jetstream.StreamSource, error) {
	jss := &jetstream.StreamSource{
		Name:          ss.Name,
		FilterSubject: ss.FilterSubject,
	}

	if ss.OptStartSeq > 0 {
		jss.OptStartSeq = uint64(ss.OptStartSeq)
	}
	if ss.OptStartTime != "" {
		t, err := time.Parse(time.RFC3339, ss.OptStartTime)
		if err != nil {
			return nil, fmt.Errorf("parse opt start time: %w", err)
		}
		jss.OptStartTime = &t
	}

	if ss.ExternalAPIPrefix != "" || ss.ExternalDeliverPrefix != "" {
		jss.External = &jetstream.ExternalStream{
			APIPrefix:     ss.ExternalAPIPrefix,
			DeliverPrefix: ss.ExternalDeliverPrefix,
		}
	}

	for _, transform := range ss.SubjectTransforms {
		jss.SubjectTransforms = append(jss.SubjectTransforms, jetstream.SubjectTransformConfig{
			Source:      transform.Source,
			Destination: transform.Dest,
		})
	}

	return jss, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Stream{}).
		Complete(r)
}
