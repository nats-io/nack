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
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/nats-io/nats.go/jetstream"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// StreamReconciler reconciles a Stream object
type StreamReconciler struct {
	Scheme *runtime.Scheme

	JetStreamController
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// It performs three main operations:
// - Initialize finalizer and ready condition if not present
// - Delete stream if it is marked for deletion.
// - Create or Update the stream
//
// A call to reconcile may perform only one action, expecting the reconciliation to be triggered again by an update.
// For example: Setting the finalizer triggers a second reconciliation. Reconcile returns after setting the finalizer,
// to prevent parallel reconciliations performing the same steps.
func (r *StreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := klog.FromContext(ctx)

	if ok := r.ValidNamespace(req.Namespace); !ok {
		log.Info("Controller restricted to namespace, skipping reconciliation.")
		return ctrl.Result{}, nil
	}

	// Fetch stream resource
	stream := &api.Stream{}
	if err := r.Get(ctx, req.NamespacedName, stream); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Stream deleted.", "streamName", req.NamespacedName.String())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get stream resource '%s': %w", req.NamespacedName.String(), err)
	}

	log = log.WithValues("streamName", stream.Spec.Name)

	// Update ready status to unknown when no status is set
	if len(stream.Status.Conditions) == 0 {
		log.Info("Setting initial ready condition to unknown.")
		stream.Status.Conditions = updateReadyCondition(stream.Status.Conditions, v1.ConditionUnknown, stateReconciling, "Starting reconciliation")
		err := r.Status().Update(ctx, stream)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("set condition unknown: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Check Deletion
	markedForDeletion := stream.GetDeletionTimestamp() != nil
	if markedForDeletion {
		if controllerutil.ContainsFinalizer(stream, streamFinalizer) {
			err := r.deleteStream(ctx, log, stream)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("delete stream: %w", err)
			}
		} else {
			log.Info("Stream marked for deletion and already finalized. Ignoring.")
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(stream, streamFinalizer) {
		log.Info("Adding stream finalizer.")
		if ok := controllerutil.AddFinalizer(stream, streamFinalizer); !ok {
			return ctrl.Result{}, errors.New("failed to add finalizer to stream resource")
		}

		if err := r.Update(ctx, stream); err != nil {
			return ctrl.Result{}, fmt.Errorf("update stream resource to add finalizer: %w", err)
		}
		return ctrl.Result{}, nil
	}

	// Create or update stream
	if err := r.createOrUpdate(ctx, log, stream); err != nil {
		return ctrl.Result{}, fmt.Errorf("create or update: %s", err)
	}

	return ctrl.Result{RequeueAfter: r.RequeueInterval()}, nil
}

func (r *StreamReconciler) deleteStream(ctx context.Context, log logr.Logger, stream *api.Stream) error {
	// Set status to false
	stream.Status.Conditions = updateReadyCondition(stream.Status.Conditions, v1.ConditionFalse, stateFinalizing, "Performing finalizer operations.")
	if err := r.Status().Update(ctx, stream); err != nil {
		return fmt.Errorf("update ready condition: %w", err)
	}

	storedState, err := getStoredStreamState(stream)
	if err != nil {
		log.Error(err, "Failed to fetch stored state.")
	}

	if !stream.Spec.PreventDelete && !r.ReadOnly() {
		log.Info("Deleting stream.")
		err := r.WithJetStreamClient(stream.Spec.ConnectionOpts, stream.Namespace, func(js jetstream.JetStream) error {
			_, err := getServerStreamState(ctx, js, stream)
			// If we have no known state for this stream it has never been reconciled.
			// If we are also receiving an error fetching state, either the stream does not exist
			// or this resource config is invalid.
			if err != nil && storedState == nil {
				return nil
			}

			return js.DeleteStream(ctx, stream.Spec.Name)
		})
		if errors.Is(err, jetstream.ErrStreamNotFound) {
			log.Info("Stream does not exist, unable to delete.", "streamName", stream.Spec.Name)
		} else if err != nil && storedState == nil {
			log.Info("Stream not reconciled and no state received from server. Removing finalizer.")
		} else if err != nil {
			return fmt.Errorf("delete stream during finalization: %w", err)
		}
	} else {
		log.Info("Skipping stream deletion.",
			"preventDelete", stream.Spec.PreventDelete,
			"read-only", r.ReadOnly(),
		)
	}

	log.Info("Removing stream finalizer.")
	if ok := controllerutil.RemoveFinalizer(stream, streamFinalizer); !ok {
		return errors.New("failed to remove stream finalizer")
	}
	if err := r.Update(ctx, stream); err != nil {
		return fmt.Errorf("remove finalizer: %w", err)
	}

	return nil
}

func (r *StreamReconciler) createOrUpdate(ctx context.Context, log logr.Logger, stream *api.Stream) error {
	// Create or Update the stream based on the spec
	// Map spec to stream targetConfig
	targetConfig, err := streamSpecToConfig(&stream.Spec)
	if err != nil {
		return fmt.Errorf("map spec to stream targetConfig: %w", err)
	}

	// CreateOrUpdateStream is called on every reconciliation when the stream is not to be deleted.
	err = r.WithJetStreamClient(stream.Spec.ConnectionOpts, stream.Namespace, func(js jetstream.JetStream) error {
		storedState, err := getStoredStreamState(stream)
		if err != nil {
			log.Error(err, "Failed to fetch stored stream state")
		}

		serverState, err := getServerStreamState(ctx, js, stream)
		if err != nil {
			return err
		}

		// Check against known state. Skip Update if converged.
		// Storing returned state from the server avoids have to
		// check default values or call Update on already converged resources
		if storedState != nil && serverState != nil && stream.Status.ObservedGeneration == stream.Generation {
			diff := compareConfigState(storedState, serverState)

			if diff == "" {
				return nil
			}

			log.Info("Stream config drifted from desired state.", "diff", diff)
		}

		if r.ReadOnly() {
			log.Info("Skipping stream creation or update.",
				"read-only", r.ReadOnly(),
			)
			return nil
		}

		var updatedStream jetstream.Stream
		err = nil

		if serverState == nil {
			log.Info("Creating Stream.")
			updatedStream, err = js.CreateStream(ctx, targetConfig)
			if err != nil {
				return err
			}
		} else if !stream.Spec.PreventUpdate {
			log.Info("Updating Stream.")
			updatedStream, err = js.UpdateStream(ctx, targetConfig)
			if err != nil {
				return err
			}
		} else {
			log.Info("Skipping Stream update.",
				"preventUpdate", stream.Spec.PreventUpdate,
			)
		}

		if updatedStream != nil {
			// Store known state in annotation
			updatedState, err := json.Marshal(updatedStream.CachedInfo().Config)
			if err != nil {
				return err
			}

			if stream.Annotations == nil {
				stream.Annotations = map[string]string{}
			}
			stream.Annotations[stateAnnotationStream] = string(updatedState)

			return r.Update(ctx, stream)
		}

		return nil
	})
	if err != nil {
		err = fmt.Errorf("create or update stream: %w", err)
		stream.Status.Conditions = updateReadyCondition(stream.Status.Conditions, v1.ConditionFalse, stateErrored, err.Error())
		if err := r.Status().Update(ctx, stream); err != nil {
			log.Error(err, "Failed to update ready condition to Errored.")
		}
		return err
	}

	// update the observed generation and ready status
	stream.Status.ObservedGeneration = stream.Generation
	stream.Status.Conditions = updateReadyCondition(
		stream.Status.Conditions,
		v1.ConditionTrue,
		stateReady,
		"Stream successfully created or updated.",
	)
	err = r.Status().Update(ctx, stream)
	if err != nil {
		return fmt.Errorf("update ready condition: %w", err)
	}

	return nil
}

func getStoredStreamState(stream *api.Stream) (*jetstream.StreamConfig, error) {
	var storedState *jetstream.StreamConfig
	if state, ok := stream.Annotations[stateAnnotationStream]; ok {
		err := json.Unmarshal([]byte(state), &storedState)
		if err != nil {
			return nil, err
		}
	}

	return storedState, nil
}

// Fetch the current state of the stream from the server.
// ErrStreamNotFound is considered a valid response and does not return error
func getServerStreamState(ctx context.Context, js jetstream.JetStream, stream *api.Stream) (*jetstream.StreamConfig, error) {
	s, err := js.Stream(ctx, stream.Spec.Name)
	if errors.Is(err, jetstream.ErrStreamNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &s.CachedInfo().Config, nil
}

// streamSpecToConfig creates a jetstream.StreamConfig matching the given stream resource spec
func streamSpecToConfig(spec *api.StreamSpec) (jetstream.StreamConfig, error) {
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
		Sealed:               spec.Sealed,
		DenyDelete:           spec.DenyDelete,
		DenyPurge:            spec.DenyPurge,
		AllowRollup:          spec.AllowRollup,
		FirstSeq:             spec.FirstSequence,
		AllowDirect:          spec.AllowDirect,
		MirrorDirect:         spec.MirrorDirect,
		Metadata:             spec.Metadata,
	}

	// Set not directly mapped fields

	// retention
	if spec.Retention != "" {
		// Wrap string in " to be properly unmarshalled as json string
		err := config.Retention.UnmarshalJSON(jsonString(spec.Retention))
		if err != nil {
			return jetstream.StreamConfig{}, fmt.Errorf("invalid retention policy: %w", err)
		}
	}

	// discard
	if spec.Discard != "" {
		err := config.Discard.UnmarshalJSON(jsonString(spec.Discard))
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
		err := config.Storage.UnmarshalJSON(jsonString(spec.Storage))
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
			return jetstream.StreamConfig{}, fmt.Errorf("map mirror stream source: %w", err)
		}
		config.Mirror = ss
	}

	// sources
	if spec.Sources != nil {
		config.Sources = []*jetstream.StreamSource{}
		for _, source := range spec.Sources {
			s, err := mapStreamSource(source)
			if err != nil {
				return jetstream.StreamConfig{}, fmt.Errorf("map stream source: %w", err)
			}
			config.Sources = append(config.Sources, s)
		}
	}

	// compression
	if spec.Compression != "" {
		err := config.Compression.UnmarshalJSON(jsonString(spec.Compression))
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
	if spec.RePublish != nil {
		config.RePublish = &jetstream.RePublish{
			Source:      spec.RePublish.Source,
			Destination: spec.RePublish.Destination,
			HeadersOnly: spec.RePublish.HeadersOnly,
		}
	}

	// consumerLimits
	if spec.ConsumerLimits != nil {
		inactiveThreshold, err := time.ParseDuration(spec.ConsumerLimits.InactiveThreshold)
		if err != nil {
			return jetstream.StreamConfig{}, fmt.Errorf("parse inactive threshold: %w", err)
		}
		config.ConsumerLimits = jetstream.StreamConsumerLimits{
			InactiveThreshold: inactiveThreshold,
			MaxAckPending:     spec.ConsumerLimits.MaxAckPending,
		}
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
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}
