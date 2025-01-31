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

const (
	objStreamPrefix = "OBJ_"
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
			log.Info("ObjectStore resource deleted.", "objectStoreName", req.NamespacedName.String())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get objectstore resource '%s': %w", req.NamespacedName.String(), err)
	}

	log = log.WithValues("objectStoreName", objectStore.Spec.Bucket)

	// Update ready status to unknown when no status is set
	if len(objectStore.Status.Conditions) == 0 {
		log.Info("Setting initial ready condition to unknown.")
		objectStore.Status.Conditions = updateReadyCondition(objectStore.Status.Conditions, v1.ConditionUnknown, stateReconciling, "Starting reconciliation")
		err := r.Status().Update(ctx, objectStore)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("set condition unknown: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
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

	// Create or update ObjectStore
	if err := r.createOrUpdate(ctx, log, objectStore); err != nil {
		return ctrl.Result{}, fmt.Errorf("create or update: %s", err)
	}

	return ctrl.Result{RequeueAfter: r.RequeueInterval()}, nil
}

func (r *ObjectStoreReconciler) deleteObjectStore(ctx context.Context, log logr.Logger, objectStore *api.ObjectStore) error {
	// Set status to false
	objectStore.Status.Conditions = updateReadyCondition(objectStore.Status.Conditions, v1.ConditionFalse, stateFinalizing, "Performing finalizer operations.")
	if err := r.Status().Update(ctx, objectStore); err != nil {
		return fmt.Errorf("update ready condition: %w", err)
	}

	storedState, err := getStoredObjectStoreState(objectStore)
	if err != nil {
		log.Error(err, "Failed to fetch stored state.")
	}

	if !objectStore.Spec.PreventDelete && !r.ReadOnly() {
		log.Info("Deleting ObjectStore.")
		err := r.WithJetStreamClient(objectStore.Spec.ConnectionOpts, objectStore.Namespace, func(js jetstream.JetStream) error {
			_, err := getServerObjectStoreState(ctx, js, objectStore)
			// If we have no known state for this object store it has never been reconciled.
			// If we are also receiving an error fetching state, either the object store does not exist
			// or this resource config is invalid.
			if err != nil && storedState == nil {
				return nil
			}

			return js.DeleteObjectStore(ctx, objectStore.Spec.Bucket)
		})
		if errors.Is(err, jetstream.ErrStreamNotFound) || errors.Is(err, jetstream.ErrBucketNotFound) {
			log.Info("ObjectStore does not exist, unable to delete.", "objectStoreName", objectStore.Spec.Bucket)
		} else if err != nil && storedState == nil {
			log.Info("ObjectStore not reconciled and no state received from server. Removing finalizer.")
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
	// Map spec to ObjectStore targetConfig
	targetConfig, err := objectStoreSpecToConfig(&objectStore.Spec)
	if err != nil {
		return fmt.Errorf("map spec to objectstore targetConfig: %w", err)
	}

	// UpdateObjectStore is called on every reconciliation when the stream is not to be deleted.
	err = r.WithJetStreamClient(objectStore.Spec.ConnectionOpts, objectStore.Namespace, func(js jetstream.JetStream) error {
		storedState, err := getStoredObjectStoreState(objectStore)
		if err != nil {
			log.Error(err, "Failed to fetch stored objectstore state")
		}

		serverState, err := getServerObjectStoreState(ctx, js, objectStore)
		if err != nil {
			return err
		}

		// Check against known state. Skip Update if converged.
		// Storing returned state from the server avoids have to
		// check default values or call Update on already converged resources
		if storedState != nil && serverState != nil && objectStore.Status.ObservedGeneration == objectStore.Generation {
			diff := compareConfigState(storedState, serverState)

			if diff == "" {
				return nil
			}

			log.Info("Object Store config drifted from desired state.", "diff", diff)
		}

		if r.ReadOnly() {
			log.Info("Skipping ObjectStore creation or update.",
				"read-only", r.ReadOnly(),
			)
			return nil
		}

		var updatedObjectStore jetstream.ObjectStore
		err = nil

		if serverState == nil {
			log.Info("Creating ObjectStore.")
			updatedObjectStore, err = js.CreateObjectStore(ctx, targetConfig)
			if err != nil {
				return err
			}
		} else if !objectStore.Spec.PreventUpdate {
			log.Info("Updating ObjectStore.")
			updatedObjectStore, err = js.UpdateObjectStore(ctx, targetConfig)
			if err != nil {
				return err
			}

			updatedObjectStore, err := getServerObjectStoreState(ctx, js, objectStore)
			if err != nil {
				log.Error(err, "Failed to fetch updated objectstore state")
			} else {
				diff := compareConfigState(updatedObjectStore, serverState)
				log.Info("Updated ObjectStore.", "diff", diff)
			}
		} else {
			log.Info("Skipping ObjectStore update.",
				"preventUpdate", objectStore.Spec.PreventUpdate,
			)
		}

		if updatedObjectStore != nil {
			// Store known state in annotation
			serverState, err = getServerObjectStoreState(ctx, js, objectStore)
			if err != nil {
				return err
			}

			updatedState, err := json.Marshal(serverState)
			if err != nil {
				return err
			}

			if objectStore.Annotations == nil {
				objectStore.Annotations = map[string]string{}
			}
			objectStore.Annotations[stateAnnotationObj] = string(updatedState)

			return r.Update(ctx, objectStore)
		}

		return nil
	})
	if err != nil {
		err = fmt.Errorf("create or update objectstore: %w", err)
		objectStore.Status.Conditions = updateReadyCondition(objectStore.Status.Conditions, v1.ConditionFalse, stateErrored, err.Error())
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
		stateReady,
		"ObjectStore successfully created or updated.",
	)
	err = r.Status().Update(ctx, objectStore)
	if err != nil {
		return fmt.Errorf("update ready condition: %w", err)
	}

	return nil
}

func getStoredObjectStoreState(objectStore *api.ObjectStore) (*jetstream.StreamConfig, error) {
	var storedState *jetstream.StreamConfig
	if state, ok := objectStore.Annotations[stateAnnotationObj]; ok {
		err := json.Unmarshal([]byte(state), &storedState)
		if err != nil {
			return nil, err
		}
	}

	return storedState, nil
}

// Fetch the current state of the ObjectStore stream from the server.
// ErrStreamNotFound is considered a valid response and does not return error
func getServerObjectStoreState(ctx context.Context, js jetstream.JetStream, objectStore *api.ObjectStore) (*jetstream.StreamConfig, error) {
	s, err := js.Stream(ctx, fmt.Sprintf("%s%s", objStreamPrefix, objectStore.Spec.Bucket))
	if errors.Is(err, jetstream.ErrStreamNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &s.CachedInfo().Config, nil
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
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}
