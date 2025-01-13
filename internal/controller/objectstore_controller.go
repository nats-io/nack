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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ObjectStoreReconciler reconciles a ObjectStore object
type ObjectStoreReconciler struct {
	Scheme *runtime.Scheme

	JetStreamController
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// It performs three main operations:
// - Initialize finalizer and ready condition if not present
// - Delete ObjectStore if it is marked for deletion.
// - Create or Update the ObjectStore
//
// A call to reconcile may perform only one action, expecting the reconciliation to be triggered again by an update.
// For example: Setting the finalizer triggers a second reconciliation. Reconcile returns after setting the finalizer,
// to prevent parallel reconciliations performing the same steps.
func (r *ObjectStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := klog.FromContext(ctx)

	if ok := r.ValidNamespace(req.Namespace); !ok {
		log.Info("Controller restricted to namespace, skipping reconciliation.")
		return ctrl.Result{}, nil
	}

	// Fetch ObjectStore resource
	objectStore := &api.ObjectStore{}
	if err := r.Get(ctx, req.NamespacedName, objectStore); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ObjectStore resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get objectstore resource '%s': %w", req.NamespacedName.String(), err)
	}

	log = log.WithValues("objectStoreName", objectStore.Spec.Bucket)

	// Update ready status to unknown when no status is set
	if len(objectStore.Status.Conditions) == 0 {
		log.Info("Setting initial ready condition to unknown.")
		objectStore.Status.Conditions = updateReadyCondition(objectStore.Status.Conditions, v1.ConditionUnknown, "Reconciling", "Starting reconciliation")
		err := r.Status().Update(ctx, objectStore)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("set condition unknown: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(objectStore, objectStoreFinalizer) {
		log.Info("Adding ObjectStore finalizer.")
		if ok := controllerutil.AddFinalizer(objectStore, objectStoreFinalizer); !ok {
			return ctrl.Result{}, errors.New("failed to add finalizer to objectstore resource")
		}

		if err := r.Update(ctx, objectStore); err != nil {
			return ctrl.Result{}, fmt.Errorf("update objectstore resource to add finalizer: %w", err)
		}
		return ctrl.Result{}, nil
	}

	// Check Deletion
	markedForDeletion := objectStore.GetDeletionTimestamp() != nil
	if markedForDeletion {
		if controllerutil.ContainsFinalizer(objectStore, objectStoreFinalizer) {
			err := r.deleteObjectStore(ctx, log, objectStore)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("delete objectstore: %w", err)
			}
		} else {
			log.Info("ObjectStore marked for deletion and already finalized. Ignoring.")
		}

		return ctrl.Result{}, nil
	}

	// Create or update ObjectStore
	if err := r.createOrUpdate(ctx, log, objectStore); err != nil {
		return ctrl.Result{}, fmt.Errorf("create or update: %s", err)
	}
	return ctrl.Result{}, nil
}

func (r *ObjectStoreReconciler) deleteObjectStore(ctx context.Context, log logr.Logger, objectStore *api.ObjectStore) error {
	// Set status to false
	objectStore.Status.Conditions = updateReadyCondition(objectStore.Status.Conditions, v1.ConditionFalse, "Finalizing", "Performing finalizer operations.")
	if err := r.Status().Update(ctx, objectStore); err != nil {
		return fmt.Errorf("update ready condition: %w", err)
	}

	if !objectStore.Spec.PreventDelete && !r.ReadOnly() {
		log.Info("Deleting ObjectStore.")
		err := r.WithJetStreamClient(objectStore.Spec.ConnectionOpts, func(js jetstream.JetStream) error {
			return js.DeleteObjectStore(ctx, objectStore.Spec.Bucket)
		})
		// FIX: ErrStreamNotFound -> ErrBucketNotFound once nats.go is corrected
		if errors.Is(err, jetstream.ErrStreamNotFound) {
			log.Info("ObjectStore does not exist, unable to delete.", "objectStoreName", objectStore.Spec.Bucket)
		} else if err != nil {
			return fmt.Errorf("delete objectstore during finalization: %w", err)
		}
	} else {
		log.Info("Skipping ObjectStore deletion.",
			"preventDelete", objectStore.Spec.PreventDelete,
			"read-only", r.ReadOnly(),
		)
	}

	log.Info("Removing ObjectStore finalizer.")
	if ok := controllerutil.RemoveFinalizer(objectStore, objectStoreFinalizer); !ok {
		return errors.New("failed to remove objectstore finalizer")
	}
	if err := r.Update(ctx, objectStore); err != nil {
		return fmt.Errorf("remove finalizer: %w", err)
	}

	return nil
}

func (r *ObjectStoreReconciler) createOrUpdate(ctx context.Context, log logr.Logger, objectStore *api.ObjectStore) error {
	// Create or Update the ObjectStore based on the spec
	if r.ReadOnly() {
		log.Info("Skipping ObjectStore creation or update.",
			"read-only", r.ReadOnly(),
		)
		return nil
	}

	// Map spec to ObjectStore targetConfig
	targetConfig, err := objectStoreSpecToConfig(&objectStore.Spec)
	if err != nil {
		return fmt.Errorf("map spec to objectstore targetConfig: %w", err)
	}

	// UpdateObjectStore is called on every reconciliation when the stream is not to be deleted.
	// TODO(future-feature): Do we need to check if config differs?
	err = r.WithJetStreamClient(objectStore.Spec.ConnectionOpts, func(js jetstream.JetStream) error {
		exists := false
		_, err := js.ObjectStore(ctx, targetConfig.Bucket)
		if err == nil {
			exists = true
		} else if !errors.Is(err, jetstream.ErrBucketNotFound) {
			return err
		}

		if !exists {
			log.Info("Creating ObjectStore.")
			_, err = js.CreateObjectStore(ctx, targetConfig)
			return err
		}

		if !objectStore.Spec.PreventUpdate {
			log.Info("Updating ObjectStore.")
			_, err = js.UpdateObjectStore(ctx, targetConfig)
			return err
		} else {
			log.Info("Skipping ObjectStore update.",
				"preventUpdate", objectStore.Spec.PreventUpdate,
			)
		}

		return nil
	})
	if err != nil {
		err = fmt.Errorf("create or update objectstore: %w", err)
		objectStore.Status.Conditions = updateReadyCondition(objectStore.Status.Conditions, v1.ConditionFalse, "Errored", err.Error())
		if err := r.Status().Update(ctx, objectStore); err != nil {
			log.Error(err, "Failed to update ready condition to Errored.")
		}
		return err
	}

	// update the observed generation and ready status
	objectStore.Status.ObservedGeneration = objectStore.Generation
	objectStore.Status.Conditions = updateReadyCondition(
		objectStore.Status.Conditions,
		v1.ConditionTrue,
		"Reconciling",
		"ObjectStore successfully created or updated.",
	)
	err = r.Status().Update(ctx, objectStore)
	if err != nil {
		return fmt.Errorf("update ready condition: %w", err)
	}

	return nil
}

// objectStoreSpecToConfig creates a jetstream.ObjectStoreConfig matching the given ObjectStore resource spec
func objectStoreSpecToConfig(spec *api.ObjectStoreSpec) (jetstream.ObjectStoreConfig, error) {
	// Set directly mapped fields
	config := jetstream.ObjectStoreConfig{
		Bucket:      spec.Bucket,
		Description: spec.Description,
		MaxBytes:    int64(spec.MaxBytes),
		Replicas:    spec.Replicas,
		Compression: spec.Compression,
		Metadata:    spec.Metadata,
	}

	// TTL
	if spec.TTL != "" {
		t, err := time.ParseDuration(spec.TTL)
		if err != nil {
			return jetstream.ObjectStoreConfig{}, fmt.Errorf("invalid ttl: %w", err)
		}
		config.TTL = t
	}

	// storage
	if spec.Storage != "" {
		err := config.Storage.UnmarshalJSON(jsonString(spec.Storage))
		if err != nil {
			return jetstream.ObjectStoreConfig{}, fmt.Errorf("invalid storage: %w", err)
		}
	}

	// placement
	if spec.Placement != nil {
		config.Placement = &jetstream.Placement{
			Cluster: spec.Placement.Cluster,
			Tags:    spec.Placement.Tags,
		}
	}

	return config, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ObjectStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.ObjectStore{}).
		Owns(&api.ObjectStore{}).
		// Only trigger on generation changes
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
