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
	"time"

	"github.com/go-logr/logr"
	"github.com/nats-io/jsm.go"
	jsmapi "github.com/nats-io/jsm.go/api"
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
		err := r.WithJSMClient(stream.Spec.ConnectionOpts, stream.Namespace, func(js *jsm.Manager) error {
			_, err := getServerStreamState(js, stream)
			// If we have no known state for this stream it has never been reconciled.
			// If we are also receiving an error fetching state, either the stream does not exist
			// or this resource config is invalid.
			if err != nil && storedState == nil {
				return nil
			}

			return js.DeleteStream(stream.Spec.Name)
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
	err = r.WithJSMClient(stream.Spec.ConnectionOpts, stream.Namespace, func(js *jsm.Manager) error {
		storedState, err := getStoredStreamState(stream)
		if err != nil {
			log.Error(err, "Failed to fetch stored stream state")
		}

		serverState, err := getServerStreamState(js, stream)
		if err != nil {
			log.Info("get err", "err", err, "code", getErrCode(err))
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

		var updatedStream *jsm.Stream
		err = nil

		if serverState == nil {
			log.Info("Creating Stream.")
			updatedStream, err = js.NewStream(stream.Spec.Name, targetConfig...)
			if err != nil {
				return err
			}
		} else if !stream.Spec.PreventUpdate {
			log.Info("Updating Stream.")
			s, err := js.LoadStream(stream.Spec.Name)
			if err != nil {
				return err
			}

			err = s.UpdateConfiguration(*serverState, targetConfig...)
			if err != nil {
				return err
			}

			updatedStream, err = js.LoadStream(stream.Spec.Name)
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
			updatedState, err := json.Marshal(updatedStream.Configuration())
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

func getStoredStreamState(stream *api.Stream) (*jsmapi.StreamConfig, error) {
	var storedState *jsmapi.StreamConfig
	if state, ok := stream.Annotations[stateAnnotationStream]; ok {
		err := json.Unmarshal([]byte(state), &storedState)
		if err != nil {
			return nil, err
		}
	}

	return storedState, nil
}

// Fetch the current state of the stream from the server.
// JSStreamNotFoundErr is considered a valid response and does not return error
func getServerStreamState(jsm *jsm.Manager, stream *api.Stream) (*jsmapi.StreamConfig, error) {
	s, err := jsm.LoadStream(stream.Spec.Name)
	// 10059 -> JSStreamNotFoundErr
	if jsmapi.IsNatsErr(err, 10059) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	streamCfg := s.Configuration()
	return &streamCfg, nil
}

// streamSpecToConfig creates a jetstream.StreamConfig matching the given stream resource spec
func streamSpecToConfig(spec *api.StreamSpec) ([]jsm.StreamOption, error) {
	opts := []jsm.StreamOption{
		jsm.StreamDescription(spec.Description),
		jsm.Subjects(spec.Subjects...),
		jsm.MaxConsumers(spec.MaxConsumers),
		jsm.MaxMessages(int64(spec.MaxMsgs)),
		jsm.MaxBytes(int64(spec.MaxBytes)),
		jsm.MaxMessageSize(int32(spec.MaxMsgSize)),
		jsm.Replicas(spec.Replicas),
	}

	// Set not directly mapped fields

	// retention
	switch spec.Retention {
	case "limits":
		opts = append(opts, jsm.LimitsRetention())
	case "interest":
		opts = append(opts, jsm.InterestRetention())
	case "workqueue":
		opts = append(opts, jsm.WorkQueueRetention())
	}

	// maxMsgsPerSubject
	if spec.MaxMsgsPerSubject > 0 {
		opts = append(opts, func(o *jsmapi.StreamConfig) error {
			o.MaxMsgsPer = int64(spec.MaxMsgsPerSubject)
			return nil
		})
	}

	// maxAge
	if spec.MaxAge != "" {
		d, err := time.ParseDuration(spec.MaxAge)
		if err != nil {
			return nil, fmt.Errorf("parse max age: %w", err)
		}
		opts = append(opts, jsm.MaxAge(d))
	}

	// storage
	switch spec.Storage {
	case "file":
		opts = append(opts, jsm.FileStorage())
	case "memory":
		opts = append(opts, jsm.MemoryStorage())
	}

	// discard
	switch spec.Discard {
	case "old":
		opts = append(opts, jsm.DiscardOld())
	case "new":
		opts = append(opts, jsm.DiscardNew())
	}

	// noAck
	if spec.NoAck {
		opts = append(opts, jsm.NoAck())
	}

	// duplicateWindow
	if spec.DuplicateWindow != "" {
		d, err := time.ParseDuration(spec.DuplicateWindow)
		if err != nil {
			return nil, fmt.Errorf("parse duplicate window: %w", err)
		}
		opts = append(opts, jsm.DuplicateWindow(d))
	}

	// placement
	if spec.Placement != nil {
		if spec.Placement.Cluster != "" {
			opts = append(opts, jsm.PlacementCluster(spec.Placement.Cluster))
		}
		if spec.Placement.Tags != nil {
			opts = append(opts, jsm.PlacementTags(spec.Placement.Tags...))
		}
	}

	// mirror
	if spec.Mirror != nil {
		ss, err := mapJSMStreamSource(spec.Mirror)
		if err != nil {
			return nil, fmt.Errorf("map mirror stream source: %w", err)
		}
		opts = append(opts, jsm.Mirror(ss))
	}

	// sources
	if spec.Sources != nil {
		streamSources := make([]*jsmapi.StreamSource, 0)
		for _, source := range spec.Sources {
			ss, err := mapJSMStreamSource(source)
			if err != nil {
				return nil, fmt.Errorf("map stream source: %w", err)
			}
			streamSources = append(streamSources, ss)
		}

		opts = append(opts, jsm.Sources(streamSources...))
	}

	// compression
	switch spec.Compression {
	case "s2":
		opts = append(opts, jsm.Compression(jsmapi.S2Compression))
	case "none":
		opts = append(opts, jsm.Compression(jsmapi.NoCompression))
	}

	// subjectTransform
	if spec.SubjectTransform != nil {
		st := &jsmapi.SubjectTransformConfig{
			Source:      spec.SubjectTransform.Source,
			Destination: spec.SubjectTransform.Dest,
		}

		opts = append(opts, jsm.SubjectTransform(st))
	}

	// rePublish
	if spec.RePublish != nil {
		r := &jsmapi.RePublish{
			Source:      spec.RePublish.Source,
			Destination: spec.RePublish.Destination,
			HeadersOnly: spec.RePublish.HeadersOnly,
		}

		opts = append(opts, jsm.Republish(r))
	}

	if spec.Sealed {
		opts = append(opts, func(o *jsmapi.StreamConfig) error {
			o.Sealed = spec.Sealed
			return nil
		})
	}

	// denyDelete
	if spec.DenyDelete {
		opts = append(opts, jsm.DenyDelete())
	}

	// denyPurge
	if spec.DenyPurge {
		opts = append(opts, jsm.DenyPurge())
	}

	// allowDirect
	if spec.AllowDirect {
		opts = append(opts, jsm.AllowDirect())
	}

	// allowRollup
	if spec.AllowRollup {
		opts = append(opts, jsm.AllowRollup())
	}

	// mirrorDirect
	if spec.MirrorDirect {
		opts = append(opts, jsm.MirrorDirect())
	}

	// discardPerSubject
	if spec.DiscardPerSubject {
		opts = append(opts, jsm.DiscardNewPerSubject())
	}

	// firstSequence
	if spec.FirstSequence > 0 {
		opts = append(opts, jsm.FirstSequence(spec.FirstSequence))
	}

	// metadata
	if spec.Metadata != nil {
		opts = append(opts, jsm.StreamMetadata(spec.Metadata))
	}

	// consumerLimits
	if spec.ConsumerLimits != nil {
		cl := jsmapi.StreamConsumerLimits{
			MaxAckPending: spec.ConsumerLimits.MaxAckPending,
		}
		if spec.ConsumerLimits.InactiveThreshold != "" {
			inactiveThreshold, err := time.ParseDuration(spec.ConsumerLimits.InactiveThreshold)
			if err != nil {
				return nil, fmt.Errorf("parse inactive threshold: %w", err)
			}
			cl.InactiveThreshold = inactiveThreshold
		}

		opts = append(opts, jsm.ConsumerLimits(cl))
	}

	return opts, nil
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

func mapJSMStreamSource(ss *api.StreamSource) (*jsmapi.StreamSource, error) {
	jss := &jsmapi.StreamSource{
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
		jss.External = &jsmapi.ExternalStream{
			ApiPrefix:     ss.ExternalAPIPrefix,
			DeliverPrefix: ss.ExternalDeliverPrefix,
		}
	}

	for _, transform := range ss.SubjectTransforms {
		jss.SubjectTransforms = append(jss.SubjectTransforms, jsmapi.SubjectTransformConfig{
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
