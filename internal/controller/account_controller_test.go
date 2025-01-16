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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
)

var _ = Describe("Account Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-account"

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		account := &api.Account{}

		// Tested controller
		var controller *AccountReconciler

		BeforeEach(func(ctx SpecContext) {
			controller = &AccountReconciler{
				Scheme:              k8sClient.Scheme(),
				JetStreamController: baseController,
			}
		})

		When("the resource is marked for deletion", func() {
			var stream *api.Stream
			var streamName types.NamespacedName

			BeforeEach(func(ctx SpecContext) {
				By("creating the custom resource for the Kind Account")
				err := k8sClient.Get(ctx, typeNamespacedName, account)
				if err != nil && k8serrors.IsNotFound(err) {
					resource := &api.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resourceName,
							Namespace: "default",
						},
						Spec: api.AccountSpec{
							Servers: []string{"nats://nats.io"},
						},
					}

					Expect(k8sClient.Create(ctx, resource)).To(Succeed())
					Expect(k8sClient.Get(ctx, typeNamespacedName, account)).To(Succeed())

					controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})

					Expect(k8sClient.Get(ctx, typeNamespacedName, account)).To(Succeed())
					Expect(controllerutil.ContainsFinalizer(account, accountFinalizer)).To(BeTrue())
				}

				By("creating a dependent stream resource")
				stream = &api.Stream{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-stream",
						Namespace: "default",
					},
					Spec: api.StreamSpec{
						Name:      "test-stream",
						Replicas:  1,
						Discard:   "old",
						Storage:   "file",
						Retention: "workqueue",
						BaseStreamConfig: api.BaseStreamConfig{
							ConnectionOpts: api.ConnectionOpts{
								Account: resourceName,
							},
						},
					},
				}
				streamName = types.NamespacedName{
					Name:      stream.Name,
					Namespace: stream.Namespace,
				}
				Expect(k8sClient.Create(ctx, stream)).To(Succeed())

				By("marking the account for deletion")
				Expect(k8sClient.Delete(ctx, account)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedName, account)).To(Succeed())
			})

			AfterEach(func(ctx SpecContext) {
				By("cleaning up the stream")
				stream := &api.Stream{}
				err := k8sClient.Get(ctx, streamName, stream)
				if err == nil {
					Expect(k8sClient.Delete(ctx, stream)).To(Succeed())
				}

				By("removing the account resource")
				controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(k8sClient.Get(ctx, typeNamespacedName, account)).To(Not(Succeed()))
			})

			It("should not remove finalizer while dependent resources exist", func(ctx SpecContext) {
				By("reconciling the deletion")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeTrue())

				By("checking the account still exists")
				Expect(k8sClient.Get(ctx, typeNamespacedName, account)).To(Succeed())
				Expect(account.Finalizers).To(ContainElement(accountFinalizer))

				By("verifying the ready condition is set to false")
				Expect(account.Status.Conditions).To(HaveLen(1))
				assertReadyStateMatches(
					account.Status.Conditions[0],
					v1.ConditionFalse,
					stateFinalizing,
					"Account has dependent resources that must be deleted first",
					time.Now(),
				)
			})

			It("should remove finalizer after dependent resources are removed", func(ctx SpecContext) {
				By("removing the dependent stream")
				Expect(k8sClient.Delete(ctx, stream)).To(Succeed())

				By("reconciling the deletion")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking the account is deleted")
				err = k8sClient.Get(ctx, typeNamespacedName, account)
				Expect(err).To(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())
			})

			It("should remove finalizer after dependent resources are updated", func(ctx SpecContext) {
				By("updating the dependent stream to remove account reference")
				Expect(k8sClient.Get(ctx, streamName, stream)).To(Succeed())
				stream.Spec.Account = ""
				Expect(k8sClient.Update(ctx, stream)).To(Succeed())

				By("reconciling the deletion")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking the account is deleted")
				err = k8sClient.Get(ctx, typeNamespacedName, account)
				Expect(err).To(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())
			})
		})
	})
})
