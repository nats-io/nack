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
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/nats-io/nats.go/jetstream"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// KeyValueReconciler reconciles a KeyValue object
type KeyValueReconciler struct {
	JetStreamController
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// It performs three main operations:
// - Initialize finalizer and ready condition if not present
// - Delete KeyValue if it is marked for deletion.
// - Create or Update the KeyValue
//
// A call to reconcile may perform only one action, expecting the reconciliation to be triggered again by an update.
// For example: Setting the finalizer triggers a second reconciliation. Reconcile returns after setting the finalizer,
// to prevent parallel reconciliations performing the same steps.
func (r *KeyValueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := klog.FromContext(ctx)

	if ok := r.ValidNamespace(req.Namespace); !ok {
		log.Info("Controller restricted to namespace, skipping reconciliation.")
		return ctrl.Result{}, nil
	}

	// Fetch KeyValue resource
	keyValue := &api.KeyValue{}
	if err := r.Get(ctx, req.NamespacedName, keyValue); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("KeyValue resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get keyvalue resource '%s': %w", req.NamespacedName.String(), err)
	}

	log = log.WithValues("keyValueName", keyValue.Spec.Name)

	// Update ready status to unknown when no status is set
	if len(keyValue.Status.Conditions) == 0 {
		log.Info("Setting initial ready condition to unknown.")
		keyValue.Status.Conditions = updateReadyCondition(keyValue.Status.Conditions, v1.ConditionUnknown, "Reconciling", "Starting reconciliation")
		err := r.Status().Update(ctx, keyValue)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("set condition unknown: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(keyValue, keyValueFinalizer) {
		log.Info("Adding KeyValue finalizer.")
		if ok := controllerutil.AddFinalizer(keyValue, keyValueFinalizer); !ok {
			return ctrl.Result{}, errors.New("failed to add finalizer to keyvalue resource")
		}

		if err := r.Update(ctx, keyValue); err != nil {
			return ctrl.Result{}, fmt.Errorf("update keyvalue resource to add finalizer: %w", err)
		}
		return ctrl.Result{}, nil
	}

	// Check Deletion
	markedForDeletion := keyValue.GetDeletionTimestamp() != nil
	if markedForDeletion {
		if controllerutil.ContainsFinalizer(keyValue, keyValueFinalizer) {
			err := r.deleteKeyValue(ctx, log, keyValue)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("delete keyvalue: %w", err)
			}
		} else {
			log.Info("KeyValue marked for deletion and already finalized. Ignoring.")
		}

		return ctrl.Result{}, nil
	}

	// Create or update KeyValue
	if err := r.createOrUpdate(ctx, log, keyValue); err != nil {
		return ctrl.Result{}, fmt.Errorf("create or update: %s", err)
	}
	return ctrl.Result{}, nil
}

func (r *KeyValueReconciler) deleteKeyValue(ctx context.Context, log logr.Logger, keyValue *api.KeyValue) error {
	// Set status to not false
	keyValue.Status.Conditions = updateReadyCondition(keyValue.Status.Conditions, v1.ConditionFalse, "Finalizing", "Performing finalizer operations.")
	if err := r.Status().Update(ctx, keyValue); err != nil {
		return fmt.Errorf("update ready condition: %w", err)
	}

	if !keyValue.Spec.PreventDelete && !r.ReadOnly() {
		log.Info("Deleting KeyValue.")
		err := r.WithJetStreamClient(keyValueConnOpts(keyValue.Spec), func(js jetstream.JetStream) error {
			return js.DeleteKeyValue(ctx, keyValue.Spec.Name)
		})
		if errors.Is(err, jetstream.ErrBucketNotFound) {
			log.Info("KeyValue does not exist, unable to delete.", "keyValueName", keyValue.Spec.Name)
		} else if err != nil {
			return fmt.Errorf("delete keyvalue during finalization: %w", err)
		}
	} else {
		log.Info("Skipping KeyValue deletion.",
			"preventDelete", keyValue.Spec.PreventDelete,
			"read-only", r.ReadOnly(),
		)
	}

	log.Info("Removing KeyValue finalizer.")
	if ok := controllerutil.RemoveFinalizer(keyValue, keyValueFinalizer); !ok {
		return errors.New("failed to remove keyvalue finalizer")
	}
	if err := r.Update(ctx, keyValue); err != nil {
		return fmt.Errorf("remove finalizer: %w", err)
	}

	return nil
}

func (r *KeyValueReconciler) createOrUpdate(ctx context.Context, log logr.Logger, keyValue *api.KeyValue) error {
	// Create or Update the KeyValue based on the spec
	if r.ReadOnly() {
		log.Info("Skipping KeyValue creation or update.",
			"read-only", r.ReadOnly(),
		)
		return nil
	}

	// Map spec to KeyValue targetConfig
	targetConfig, err := keyValueSpecToConfig(&keyValue.Spec)
	if err != nil {
		return fmt.Errorf("map spec to keyvalue targetConfig: %w", err)
	}

	// UpdateKeyValue is called on every reconciliation when the stream is not to be deleted.
	// TODO(future-feature): Do we need to check if config differs?
	err = r.WithJetStreamClient(keyValueConnOpts(keyValue.Spec), func(js jetstream.JetStream) error {
		exists := false
		_, err := js.KeyValue(ctx, targetConfig.Bucket)
		if err == nil {
			exists = true
		}

		if !exists {
			log.Info("Creating KeyValue.")
			_, err = js.CreateKeyValue(ctx, targetConfig)
			return err
		}

		if !keyValue.Spec.PreventUpdate {
			log.Info("Updating KeyValue.")
			_, err = js.UpdateKeyValue(ctx, targetConfig)
			return err
		} else {
			log.Info("Skipping KeyValue update.",
				"preventUpdate", keyValue.Spec.PreventUpdate,
			)
		}

		return nil
	})
	if err != nil {
		err = fmt.Errorf("create or update keyvalue: %w", err)
		keyValue.Status.Conditions = updateReadyCondition(keyValue.Status.Conditions, v1.ConditionFalse, "Errored", err.Error())
		if err := r.Status().Update(ctx, keyValue); err != nil {
			log.Error(err, "Failed to update ready condition to Errored.")
		}
		return err
	}

	// update the observed generation and ready status
	keyValue.Status.ObservedGeneration = keyValue.Generation
	keyValue.Status.Conditions = updateReadyCondition(
		keyValue.Status.Conditions,
		v1.ConditionTrue,
		"Reconciling",
		"KeyValue successfully created or updated.",
	)
	err = r.Status().Update(ctx, keyValue)
	if err != nil {
		return fmt.Errorf("update ready condition: %w", err)
	}

	return nil
}

// keyValueConnOpts extracts nats connection relevant fields from the given KeyValue spec as connectionOptions.
func keyValueConnOpts(spec api.KeyValueSpec) *connectionOptions {
	return &connectionOptions{
		Account: spec.Account,
		Creds:   spec.Creds,
		Nkey:    spec.Nkey,
		Servers: spec.Servers,
		TLS:     spec.TLS,
	}
}

// keyValueSpecToConfig creates a jetstream.KeyValueConfig matching the given KeyValue resource spec
func keyValueSpecToConfig(spec *api.KeyValueSpec) (jetstream.KeyValueConfig, error) {
	// Set directly mapped fields
	config := jetstream.KeyValueConfig{
		Bucket:       spec.Name,
		Compression:  spec.Compression,
		Description:  spec.Description,
		History:      uint8(spec.History),
		MaxBytes:     int64(spec.MaxBytes),
		MaxValueSize: int32(spec.MaxValueSize),
		Replicas:     spec.Replicas,
	}

	// TTL
	if spec.TTL != "" {
		t, err := time.ParseDuration(spec.TTL)
		if err != nil {
			return jetstream.KeyValueConfig{}, fmt.Errorf("invalid ttl: %w", err)
		}
		config.TTL = t
	}

	// storage
	if spec.Storage != "" {
		err := config.Storage.UnmarshalJSON(asJsonString(spec.Storage))
		if err != nil {
			return jetstream.KeyValueConfig{}, fmt.Errorf("invalid storage: %w", err)
		}
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
			return jetstream.KeyValueConfig{}, fmt.Errorf("map mirror keyvalue source: %w", err)
		}
		config.Mirror = ss
	}

	// sources
	if spec.Sources != nil {
		config.Sources = []*jetstream.StreamSource{}
		for _, source := range spec.Sources {
			s, err := mapStreamSource(source)
			if err != nil {
				return jetstream.KeyValueConfig{}, fmt.Errorf("map keyvalue source: %w", err)
			}
			config.Sources = append(config.Sources, s)
		}
	}

	// RePublish
	if spec.Republish != nil {
		config.RePublish = &jetstream.RePublish{
			Source:      spec.Republish.Source,
			Destination: spec.Republish.Destination,
			HeadersOnly: spec.Republish.HeadersOnly,
		}
	}

	return config, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeyValueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.KeyValue{}).
		Owns(&api.KeyValue{}).
		// Only trigger on generation changes
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
