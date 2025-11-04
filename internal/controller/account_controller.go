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

	"github.com/go-logr/logr"
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// AccountReconciler reconciles a Account object
type AccountReconciler struct {
	Scheme *runtime.Scheme
	JetStreamController
}

type JetStreamResource interface {
	GetName() string
	GetNamespace() string
}

type JetStreamResourceList []JetStreamResource

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// It performs two main operations:
// - Initialize finalizer if not present
// - Remove the finalizer on deletion once no other resources are referencing the account
//
// A call to reconcile may perform only one action, expecting the reconciliation to be triggered again by an update.
// For example: Setting the finalizer triggers a second reconciliation. Reconcile returns after setting the finalizer,
// to prevent parallel reconciliations performing the same steps.
func (r *AccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := klog.FromContext(ctx)

	if ok := r.ValidNamespace(req.Namespace); !ok {
		log.Info("Controller restricted to namespace, skipping reconciliation.")
		return ctrl.Result{}, nil
	}

	// Fetch Account resource
	account := &api.Account{}
	if err := r.Get(ctx, req.NamespacedName, account); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Account deleted.", "accountName", req.NamespacedName.String())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get account resource '%s': %w", req.NamespacedName.String(), err)
	}

	// Update ready status to unknown when no status is set
	if len(account.Status.Conditions) == 0 {
		log.Info("Setting initial ready condition to unknown.")
		account.Status.Conditions = updateReadyCondition(account.Status.Conditions, v1.ConditionUnknown, stateReconciling, "Starting reconciliation")
		err := r.Status().Update(ctx, account)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("set condition unknown: %w", err)
		}
		r.Get(ctx, req.NamespacedName, account)
		log.Info("Status", "Conditions", account.Status.Conditions)
		return ctrl.Result{Requeue: true}, nil
	}

	// Check Deletion
	markedForDeletion := account.GetDeletionTimestamp() != nil
	if markedForDeletion {
		if controllerutil.ContainsFinalizer(account, accountFinalizer) {
			// Get list of resources referencing this account
			requests := r.findDependentResources(ctx, log, account)
			if len(requests) > 0 {
				log.Info("Account still has dependent resources, cannot delete", "dependentCount", len(requests))
				account.Status.Conditions = updateReadyCondition(
					account.Status.Conditions,
					v1.ConditionFalse,
					stateFinalizing,
					"Account has dependent resources that must be deleted first",
				)
				if err := r.Status().Update(ctx, account); err != nil {
					return ctrl.Result{}, fmt.Errorf("update status: %w", err)
				}
				return ctrl.Result{Requeue: true}, nil
			}

			log.Info("Removing Account finalizer")
			if ok := controllerutil.RemoveFinalizer(account, accountFinalizer); !ok {
				return ctrl.Result{}, errors.New("failed to remove finalizer")
			}
			if err := r.Update(ctx, account); err != nil {
				return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
			}
		} else {
			log.Info("Account marked for deletion and already finalized. Ignoring.")
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(account, accountFinalizer) {
		log.Info("Adding Account finalizer.")
		if ok := controllerutil.AddFinalizer(account, accountFinalizer); !ok {
			return ctrl.Result{}, errors.New("failed to add finalizer to account resource")
		}

		if err := r.Update(ctx, account); err != nil {
			return ctrl.Result{}, fmt.Errorf("update account resource to add finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Update ready status for non-deleted accounts
	account.Status.ObservedGeneration = account.Generation
	account.Status.Conditions = updateReadyCondition(
		account.Status.Conditions,
		v1.ConditionTrue,
		stateReady,
		"Account is ready",
	)
	if err := r.Status().Update(ctx, account); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *AccountReconciler) findDependentResources(ctx context.Context, log logr.Logger, obj client.Object) []reconcile.Request {
	var resourceList JetStreamResourceList

	var consumerList api.ConsumerList
	if err := r.List(ctx, &consumerList,
		client.InNamespace(obj.GetNamespace()),
	); err != nil {
		log.Error(err, "Failed to list consumers")
	}
	for _, i := range consumerList.Items {
		if i.Spec.Account == obj.GetName() {
			resourceList = append(resourceList, &i)
		}
	}

	var keyValueList api.KeyValueList
	if err := r.List(ctx, &keyValueList,
		client.InNamespace(obj.GetNamespace()),
	); err != nil {
		log.Error(err, "Failed to list accounts")
	}
	for _, i := range keyValueList.Items {
		if i.Spec.Account == obj.GetName() {
			resourceList = append(resourceList, &i)
		}
	}

	var objectStoreList api.ObjectStoreList
	if err := r.List(ctx, &objectStoreList,
		client.InNamespace(obj.GetNamespace()),
	); err != nil {
		log.Error(err, "Failed to list objectstores")
	}
	for _, i := range objectStoreList.Items {
		if i.Spec.Account == obj.GetName() {
			resourceList = append(resourceList, &i)
		}
	}

	var streamList api.StreamList
	if err := r.List(ctx, &streamList,
		client.InNamespace(obj.GetNamespace()),
	); err != nil {
		log.Error(err, "Failed to list streams")
	}
	for _, i := range streamList.Items {
		if i.Spec.Account == obj.GetName() {
			resourceList = append(resourceList, &i)
		}
	}

	requests := make([]reconcile.Request, 0, len(resourceList))
	for _, resource := range resourceList {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: resource.GetNamespace(),
				Name:      resource.GetName(),
			},
		})
	}

	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *AccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Account{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}
