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
	js "github.com/nats-io/nack/controllers/jetstream"
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/nats-io/nats.go/jetstream"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

// StreamReconciler reconciles a Stream object
type StreamReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Config    *Config
	JetStream jetstream.JetStream
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *StreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := klog.FromContext(ctx).
		WithName("StreamReconciler").
		WithValues("namespace", req.Namespace, "name", req.Name)

	log.Info("reconciling")

	// TODO honour r.Config.ReadOnly and r.Config.Namespace

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

	// Update Status to unknown when no status is set
	if stream.Status.Conditions == nil || len(stream.Status.Conditions) == 0 {
		updated, err := r.updateReadyCondition(ctx, stream, v1.ConditionUnknown, "Reconciling", "Starting reconciliation")
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("set condition unknown: %w", err)
		}
		if updated {
			// re-fetch stream
			if err := r.Get(ctx, req.NamespacedName, stream); err != nil {
				if apierrors.IsNotFound(err) {
					log.Info("stream resource not found. Ignoring since object must be deleted")
					return ctrl.Result{}, nil
				}
				return ctrl.Result{}, fmt.Errorf("re-fetch stream resource: %w", err)
			}
		}

	}

	// TODO Check if marked for deletion and delete
	// TODO honour stream.Spec.PreventDelete and

	// Create or Update the stream based on the spec

	// Map spec to stream targetConfig
	targetConfig, err := mapSpecToConfig(&stream.Spec)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("map spec to stream targetConfig: %w", err)
	}

	// TODO Handle Account, Nkey, Servers and TLS from spec.
	// TODO Get specific client when there is connection config in the spec.
	var sm jetstream.StreamManager
	sm = r.JetStream

	//  TODO honour stream.Spec.PreventUpdate
	_, err = sm.CreateOrUpdateStream(ctx, targetConfig)
	if err != nil {
		_, conditionErr := r.updateReadyCondition(ctx, stream, v1.ConditionFalse, "Errored", fmt.Sprintf("create or update stream: %s", err.Error()))
		if conditionErr != nil {
			log.Error(conditionErr, "failed to update ready condition to CreationFailed")
		}
		return ctrl.Result{}, fmt.Errorf("create or update stream: %w", err)
	}

	// TODO update the generation in the status

	_, err = r.updateReadyCondition(ctx, stream, v1.ConditionTrue, "Reconciling", "Stream successfully created or updated")
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("update ready condition: %w", err)
	}

	return ctrl.Result{}, nil
}

// updateReadyCondition sets the status, reason and message.
// Then updates the resource.
// Returns true if the resource was updated by this call.
func (r *StreamReconciler) updateReadyCondition(ctx context.Context, stream *api.Stream, status v1.ConditionStatus, reason string, message string) (bool, error) {

	newCondition := api.Condition{
		Type:               readyCondType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
	}

	// Mapping to and from the metav1.Condition allows using meta.SetStatusCondition,
	// which properly handles updating conditions.
	// The mapping happens transparently to the caller.

	cond := mapConditionToMeta(newCondition)
	var conditions []metav1.Condition
	for _, condition := range stream.Status.Conditions {
		conditions = append(conditions, mapConditionToMeta(condition))
	}

	changed := meta.SetStatusCondition(&conditions, cond)
	if !changed {
		return false, nil
	}

	stream.Status.Conditions = []api.Condition{}
	for _, condition := range conditions {
		stream.Status.Conditions = append(stream.Status.Conditions, mapConditionFromMeta(condition))
	}

	stream.Status.Conditions = js.UpsertCondition(stream.Status.Conditions, newCondition)

	if err := r.Status().Update(ctx, stream); err != nil {
		return false, fmt.Errorf("update stream status: %w", err)
	}
	return true, nil
}

// mapConditionToMeta transforms a condition according the jetstream v1beta2 spec to a metav1 condition.
func mapConditionToMeta(condition api.Condition) metav1.Condition {

	status := metav1.ConditionUnknown
	switch condition.Status {
	case v1.ConditionTrue:
		status = metav1.ConditionTrue
	case v1.ConditionFalse:
		status = metav1.ConditionFalse
	}

	lastTransitionTime, err := time.Parse(time.RFC3339Nano, condition.LastTransitionTime)
	if err != nil {
		lastTransitionTime = time.Time{}
	}

	return metav1.Condition{
		Type:               condition.Type,
		Status:             status,
		LastTransitionTime: metav1.NewTime(lastTransitionTime),
		Reason:             condition.Reason,
		Message:            condition.Message,
	}
}

// mapConditionFromMeta transforms a condition according the metav1 spec to a jetstream v1beta2 condition
func mapConditionFromMeta(condition metav1.Condition) api.Condition {

	status := v1.ConditionUnknown
	switch condition.Status {
	case metav1.ConditionTrue:
		status = v1.ConditionTrue
	case metav1.ConditionFalse:
		status = v1.ConditionFalse
	}

	lastTransitionTime := condition.LastTransitionTime.Format(time.RFC3339Nano)

	return api.Condition{
		Type:               condition.Type,
		Status:             status,
		LastTransitionTime: lastTransitionTime,
		Reason:             condition.Reason,
		Message:            condition.Message,
	}
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

// asJsonString returns the given string wrapped in " and converted to []byte.
func asJsonString(v string) []byte {
	return []byte("\"" + v + "\"")
}

// SetupWithManager sets up the controller with the Manager.
func (r *StreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Stream{}).
		Complete(r)
}
