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
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
)

var _ = Describe("KeyValue Controller", func() {
	// The test keyValue resource
	const resourceName = "test-kv"
	const keyValueName = "orders"

	const alternateResource = "alternate-kv"
	const alternateNamespace = "alternate-namespace"

	typeNamespacedName := types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}
	keyValue := &api.KeyValue{}

	// The tested controller
	var controller *KeyValueReconciler

	// Config to create minimal nats KeyValue store
	emptyKeyValueConfig := jetstream.KeyValueConfig{
		Bucket:   keyValueName,
		Replicas: 1,
		Storage:  jetstream.FileStorage,
	}

	BeforeEach(func(ctx SpecContext) {
		By("creating a test keyvalue resource")
		err := k8sClient.Get(ctx, typeNamespacedName, keyValue)
		if err != nil && k8serrors.IsNotFound(err) {
			resource := &api.KeyValue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: api.KeyValueSpec{
					Bucket:      keyValueName,
					Replicas:    1,
					History:     10,
					TTL:         "5m",
					Compression: true,
					Description: "test keyvalue",
					Storage:     "file",
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			// Re-fetch KeyValue
			Expect(k8sClient.Get(ctx, typeNamespacedName, keyValue)).To(Succeed())
		}

		By("checking precondition: nats keyvalue does not exist")
		_, err = jsClient.KeyValue(ctx, keyValueName)
		Expect(err).To(MatchError(jetstream.ErrBucketNotFound))

		By("setting up the tested controller")
		controller = &KeyValueReconciler{
			Scheme:              k8sClient.Scheme(),
			JetStreamController: baseController,
		}
	})

	AfterEach(func(ctx SpecContext) {
		By("removing the test keyvalue resource")
		resource := &api.KeyValue{}
		err := k8sClient.Get(ctx, typeNamespacedName, resource)
		if err != nil {
			Expect(err).To(MatchError(k8serrors.IsNotFound, "Is not found"))
		} else {
			if controllerutil.ContainsFinalizer(resource, keyValueFinalizer) {
				By("removing the finalizer")
				controllerutil.RemoveFinalizer(resource, keyValueFinalizer)
				Expect(k8sClient.Update(ctx, resource)).To(Succeed())
			}

			By("removing the keyvalue resource")
			Expect(k8sClient.Delete(ctx, resource)).
				To(SatisfyAny(
					Succeed(),
					MatchError(k8serrors.IsNotFound, "is not found"),
				))
		}

		By("deleting the nats keyvalue store")
		Expect(jsClient.DeleteKeyValue(ctx, keyValueName)).
			To(SatisfyAny(
				Succeed(),
				MatchError(jetstream.ErrBucketNotFound),
			))
	})

	When("reconciling a not existing resource", func() {
		It("should stop reconciliation without error", func(ctx SpecContext) {
			By("reconciling the created resource")
			result, err := controller.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "fake",
					Name:      "not-existing",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
	})

	When("reconciling a not initialized resource", func() {
		It("should initialize a new resource", func(ctx SpecContext) {
			By("re-queueing until it is initialized")
			// Initialization can require multiple reconciliation loops
			Eventually(func(ctx SpecContext) *api.KeyValue {
				_, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				got := &api.KeyValue{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, got)).To(Succeed())
				return got
			}).WithContext(ctx).
				Should(SatisfyAll(
					HaveField("Finalizers", HaveExactElements(keyValueFinalizer)),
					HaveField("Status.Conditions", Not(BeEmpty())),
				))

			By("validating the ready condition")
			// Fetch KeyValue
			Expect(k8sClient.Get(ctx, typeNamespacedName, keyValue)).To(Succeed())
			Expect(keyValue.Status.Conditions).To(HaveLen(1))

			assertReadyStateMatches(keyValue.Status.Conditions[0], v1.ConditionUnknown, stateReconciling, "Starting reconciliation", time.Now())
		})
	})

	When("reconciling a resource in a different namespace", func() {
		BeforeEach(func(ctx SpecContext) {
			By("creating a keyvalue resource in an alternate namespace while namespaced")
			alternateNamespaceResource := &api.KeyValue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alternateResource,
					Namespace: alternateNamespace,
				},
				Spec: api.KeyValueSpec{
					Bucket:      alternateResource,
					Replicas:    1,
					History:     10,
					TTL:         "5m",
					Compression: true,
					Description: "keyvalue in alternate namespace",
					Storage:     "file",
				},
			}

			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: alternateNamespace,
				},
			}
			err := k8sClient.Create(ctx, ns)
			if err != nil && !k8serrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(k8sClient.Create(ctx, alternateNamespaceResource)).To(Succeed())
		})

		AfterEach(func(ctx SpecContext) {
			By("cleaning up the resource in alternate namespace")
			alternateKeyValue := &api.KeyValue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alternateResource,
					Namespace: alternateNamespace,
				},
			}
			err := k8sClient.Delete(ctx, alternateKeyValue)
			if err != nil && !k8serrors.IsNotFound(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should not watch the resource in alternate namespace", func(ctx SpecContext) {
			By("reconciling with no explicit namespace restriction")
			alternateNamespacedName := types.NamespacedName{
				Namespace: alternateNamespace,
				Name:      alternateResource,
			}

			By("running reconciliation for the resource in alternate namespace")
			result, err := controller.Reconcile(ctx, reconcile.Request{
				NamespacedName: alternateNamespacedName,
			})

			By("verifying reconciliation completes without error")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			By("checking the keyvalue doesn't exist in NATS")
			_, err = jsClient.KeyValue(ctx, alternateResource)
			Expect(err).To(MatchError(jetstream.ErrBucketNotFound))

			By("verifying the resource still exists in the alternate namespace")
			alternateKeyValue := &api.KeyValue{}
			Expect(k8sClient.Get(ctx, alternateNamespacedName, alternateKeyValue)).To(Succeed())

			By("checking no conditions were set on the resource")
			Expect(alternateKeyValue.Status.Conditions).To(BeEmpty())
		})

		It("should watch the resource in alternate namespace when not namespaced", func(ctx SpecContext) {
			By("reconciling with a non-namespaced controller")
			testNatsConfig := &NatsConfig{ServerURL: clientUrl}
			alternateBaseController, err := NewJSController(k8sClient, testNatsConfig, &Config{})
			Expect(err).NotTo(HaveOccurred())

			alternateController := &KeyValueReconciler{
				Scheme:              k8sClient.Scheme(),
				JetStreamController: alternateBaseController,
			}

			resourceNames := []types.NamespacedName{
				typeNamespacedName,
				{
					Namespace: alternateNamespace,
					Name:      alternateResource,
				},
			}

			By("running reconciliation for the resources in all namespaces")
			for _, n := range resourceNames {
				result, err := alternateController.Reconcile(ctx, reconcile.Request{
					NamespacedName: n,
				})

				By("verifying reconciliation completes without error")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(Equal(ctrl.Result{}))
			}
		})
	})

	When("reconciling an initialized resource", func() {
		BeforeEach(func(ctx SpecContext) {
			By("initializing the keyvalue resource")

			By("setting the finalizer")
			Expect(controllerutil.AddFinalizer(keyValue, keyValueFinalizer)).To(BeTrue())
			Expect(k8sClient.Update(ctx, keyValue)).To(Succeed())

			By("setting an unknown ready state")
			keyValue.Status.Conditions = []api.Condition{{
				Type:               readyCondType,
				Status:             v1.ConditionUnknown,
				Reason:             "Test",
				Message:            "start condition",
				LastTransitionTime: time.Now().Format(time.RFC3339Nano),
			}}
			Expect(k8sClient.Status().Update(ctx, keyValue)).To(Succeed())
		})

		It("should create a new keyvalue store", func(ctx SpecContext) {
			By("running Reconcile")
			result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			// Fetch resource
			Expect(k8sClient.Get(ctx, typeNamespacedName, keyValue)).To(Succeed())

			By("checking if the ready state was updated")
			Expect(keyValue.Status.Conditions).To(HaveLen(1))
			assertReadyStateMatches(keyValue.Status.Conditions[0], v1.ConditionTrue, stateReady, "created or updated", time.Now())

			By("checking if the observed generation matches")
			Expect(keyValue.Status.ObservedGeneration).To(Equal(keyValue.Generation))

			By("checking if the keyvalue store was created")
			natsKeyValue, err := jsClient.KeyValue(ctx, keyValueName)
			Expect(err).NotTo(HaveOccurred())
			kvStatus, err := natsKeyValue.Status(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(kvStatus.Bucket()).To(Equal(keyValueName))
			Expect(kvStatus.History()).To(Equal(int64(10)))
			Expect(kvStatus.TTL()).To(Equal(5 * time.Minute))
			Expect(kvStatus.IsCompressed()).To(BeTrue())
		})

		When("PreventUpdate is set", func() {
			BeforeEach(func(ctx SpecContext) {
				By("setting preventDelete on the resource")
				keyValue.Spec.PreventUpdate = true
				Expect(k8sClient.Update(ctx, keyValue)).To(Succeed())
			})
			It("should create the keyvalue", func(ctx SpecContext) {
				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that keyvalue was created")
				_, err = jsClient.KeyValue(ctx, keyValueName)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not update the keyvalue", func(ctx SpecContext) {
				By("creating the keyvalue")
				_, err := jsClient.CreateKeyValue(ctx, emptyKeyValueConfig)
				Expect(err).NotTo(HaveOccurred())

				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that keyvalue was not updated")
				natsKeyValue, err := jsClient.KeyValue(ctx, keyValueName)
				Expect(err).NotTo(HaveOccurred())
				s, err := natsKeyValue.Status(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(s.IsCompressed()).To(BeFalse())
				Expect(s.History()).To(BeEquivalentTo(int64(1)))
			})
		})

		When("read-only mode is enabled", func() {
			BeforeEach(func(ctx SpecContext) {
				By("setting read only on the controller")
				readOnly, err := NewJSController(k8sClient, &NatsConfig{ServerURL: clientUrl}, &Config{ReadOnly: true})
				Expect(err).NotTo(HaveOccurred())
				controller = &KeyValueReconciler{
					Scheme:              k8sClient.Scheme(),
					JetStreamController: readOnly,
				}
			})

			It("should not create the keyvalue", func(ctx SpecContext) {
				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that no keyvalue was created")
				_, err = jsClient.KeyValue(ctx, keyValueName)
				Expect(err).To(MatchError(jetstream.ErrBucketNotFound))
			})
			It("should not update the keyvalue", func(ctx SpecContext) {
				By("creating the keyvalue")
				_, err := jsClient.CreateKeyValue(ctx, emptyKeyValueConfig)
				Expect(err).NotTo(HaveOccurred())

				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that keyvalue was not updated")
				natsKeyValue, err := jsClient.KeyValue(ctx, keyValueName)
				Expect(err).NotTo(HaveOccurred())
				s, err := natsKeyValue.Status(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(s.IsCompressed()).To(BeFalse())
				Expect(s.History()).To(BeEquivalentTo(int64(1)))
			})
		})

		When("namespace restriction is enabled", func() {
			BeforeEach(func(ctx SpecContext) {
				By("setting a namespace on the resource")
				namespaced, err := NewJSController(k8sClient, &NatsConfig{ServerURL: clientUrl}, &Config{Namespace: alternateNamespace})
				Expect(err).NotTo(HaveOccurred())
				controller = &KeyValueReconciler{
					Scheme:              k8sClient.Scheme(),
					JetStreamController: namespaced,
				}
			})

			It("should not create the keyvalue", func(ctx SpecContext) {
				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that no keyvalue was created")
				_, err = jsClient.KeyValue(ctx, keyValueName)
				Expect(err).To(MatchError(jetstream.ErrBucketNotFound))
			})
			It("should not update the keyvalue", func(ctx SpecContext) {
				By("creating the keyvalue")
				_, err := jsClient.CreateKeyValue(ctx, emptyKeyValueConfig)
				Expect(err).NotTo(HaveOccurred())

				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that keyvalue was not updated")
				natsKeyValue, err := jsClient.KeyValue(ctx, keyValueName)
				Expect(err).NotTo(HaveOccurred())
				s, err := natsKeyValue.Status(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(s.IsCompressed()).To(BeFalse())
				Expect(s.History()).To(BeEquivalentTo(int64(1)))
			})
		})

		It("should update an existing keyvalue", func(ctx SpecContext) {
			By("reconciling once to create the keyvalue")
			result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			// Fetch resource
			Expect(k8sClient.Get(ctx, typeNamespacedName, keyValue)).To(Succeed())
			previousTransitionTime := keyValue.Status.Conditions[0].LastTransitionTime

			By("updating the resource")
			keyValue.Spec.Description = "new description"
			keyValue.Spec.History = 50
			keyValue.Spec.TTL = "1h"
			Expect(k8sClient.Update(ctx, keyValue)).To(Succeed())

			By("reconciling the updated resource")
			result, err = controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			// Fetch resource
			Expect(k8sClient.Get(ctx, typeNamespacedName, keyValue)).To(Succeed())

			By("checking if the state transition time was not updated")
			Expect(keyValue.Status.Conditions).To(HaveLen(1))
			Expect(keyValue.Status.Conditions[0].LastTransitionTime).To(Equal(previousTransitionTime))

			By("checking if the observed generation matches")
			Expect(keyValue.Status.ObservedGeneration).To(Equal(keyValue.Generation))

			By("checking if the keyvalue was updated")
			natsKeyValue, err := jsClient.KeyValue(ctx, keyValueName)
			Expect(err).NotTo(HaveOccurred())

			keyValueStatus, err := natsKeyValue.Status(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(keyValueStatus.Bucket()).To(Equal(keyValueName))
			Expect(keyValueStatus.History()).To(Equal(int64(50)))
			Expect(keyValueStatus.TTL()).To(Equal(1 * time.Hour))
			Expect(keyValueStatus.IsCompressed()).To(BeTrue())
		})

		It("should set an error state when the nats server is not available", func(ctx SpecContext) {
			By("setting up controller with unavailable nats server")
			// Setup client for not running server
			// Use actual test server to ensure port not used by other service on test instance
			sv := CreateTestServer()
			disconnectedController, err := NewJSController(k8sClient, &NatsConfig{ServerURL: sv.ClientURL()}, &Config{})
			Expect(err).NotTo(HaveOccurred())
			sv.Shutdown()

			controller := &KeyValueReconciler{
				Scheme:              k8sClient.Scheme(),
				JetStreamController: disconnectedController,
			}

			By("reconciling resource")
			result, err := controller.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(err).To(HaveOccurred()) // Will be re-queued with back-off

			// Fetch resource
			err = k8sClient.Get(ctx, typeNamespacedName, keyValue)
			Expect(err).NotTo(HaveOccurred())

			By("checking if the status was updated")
			Expect(keyValue.Status.Conditions).To(HaveLen(1))
			assertReadyStateMatches(
				keyValue.Status.Conditions[0],
				v1.ConditionFalse,
				stateErrored,
				"create or update keyvalue:",
				time.Now(),
			)

			By("checking if the observed generation does not match")
			Expect(keyValue.Status.ObservedGeneration).ToNot(Equal(keyValue.Generation))
		})

		When("the resource is marked for deletion", func() {
			BeforeEach(func(ctx SpecContext) {
				By("marking the resource for deletion")
				Expect(k8sClient.Delete(ctx, keyValue)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedName, keyValue)).To(Succeed()) // re-fetch after update
			})

			It("should succeed deleting a not existing keyvalue", func(ctx SpecContext) {
				By("reconciling")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that the resource is deleted")
				Eventually(k8sClient.Get).
					WithArguments(ctx, typeNamespacedName, keyValue).
					ShouldNot(Succeed())
			})

			When("the underlying keyvalue exists", func() {
				BeforeEach(func(ctx SpecContext) {
					By("creating the keyvalue on the nats server")
					_, err := jsClient.CreateKeyValue(ctx, emptyKeyValueConfig)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func(ctx SpecContext) {
					err := jsClient.DeleteKeyValue(ctx, keyValueName)
					if err != nil {
						Expect(err).To(MatchError(jetstream.ErrBucketNotFound))
					}
				})

				It("should delete the keyvalue", func(ctx SpecContext) {
					By("reconciling")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking that the keyvalue is deleted")
					_, err = jsClient.KeyValue(ctx, keyValueName)
					Expect(err).To(MatchError(jetstream.ErrBucketNotFound))

					By("checking that the resource is deleted")
					Eventually(k8sClient.Get).
						WithArguments(ctx, typeNamespacedName, keyValue).
						ShouldNot(Succeed())
				})

				When("PreventDelete is set", func() {
					BeforeEach(func(ctx SpecContext) {
						By("setting preventDelete on the resource")
						keyValue.Spec.PreventDelete = true
						Expect(k8sClient.Update(ctx, keyValue)).To(Succeed())
					})
					It("Should delete the resource and not delete the nats keyvalue", func(ctx SpecContext) {
						By("reconciling")
						result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
						Expect(err).NotTo(HaveOccurred())
						Expect(result.IsZero()).To(BeTrue())

						By("checking that the keyvalue is not deleted")
						_, err = jsClient.KeyValue(ctx, keyValueName)
						Expect(err).NotTo(HaveOccurred())

						By("checking that the resource is deleted")
						Eventually(k8sClient.Get).
							WithArguments(ctx, typeNamespacedName, keyValue).
							ShouldNot(Succeed())
					})
				})

				When("read only is set", func() {
					BeforeEach(func(ctx SpecContext) {
						By("setting read only on the controller")
						readOnly, err := NewJSController(k8sClient, &NatsConfig{ServerURL: clientUrl}, &Config{ReadOnly: true})
						Expect(err).NotTo(HaveOccurred())
						controller = &KeyValueReconciler{
							Scheme:              k8sClient.Scheme(),
							JetStreamController: readOnly,
						}
					})
					It("should delete the resource and not delete the keyvalue", func(ctx SpecContext) {
						By("reconciling")
						result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
						Expect(err).NotTo(HaveOccurred())
						Expect(result.IsZero()).To(BeTrue())

						By("checking that the keyvalue is not deleted")
						_, err = jsClient.KeyValue(ctx, keyValueName)
						Expect(err).NotTo(HaveOccurred())

						By("checking that the resource is deleted")
						Eventually(k8sClient.Get).
							WithArguments(ctx, typeNamespacedName, keyValue).
							ShouldNot(Succeed())
					})
				})

				When("controller is restricted to different namespace", func() {
					BeforeEach(func(ctx SpecContext) {
						namespaced, err := NewJSController(k8sClient, &NatsConfig{ServerURL: clientUrl}, &Config{Namespace: alternateNamespace})
						Expect(err).NotTo(HaveOccurred())
						controller = &KeyValueReconciler{
							Scheme:              k8sClient.Scheme(),
							JetStreamController: namespaced,
						}
					})
					It("should not delete the resource and keyvalue", func(ctx SpecContext) {
						By("reconciling")
						result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
						Expect(err).NotTo(HaveOccurred())
						Expect(result.IsZero()).To(BeTrue())

						By("checking that the keyvalue is not deleted")
						_, err = jsClient.KeyValue(ctx, keyValueName)
						Expect(err).NotTo(HaveOccurred())

						By("checking that the finalizer is not removed")
						Expect(k8sClient.Get(ctx, typeNamespacedName, keyValue)).To(Succeed())
						Expect(keyValue.Finalizers).To(ContainElement(keyValueFinalizer))
					})
				})
			})
		})

		It("should update keyvalue on different server as specified in spec", func(ctx SpecContext) {
			By("setting up the alternative server")
			// Setup altClient for alternate server
			altServer := CreateTestServer()
			defer altServer.Shutdown()

			By("setting the server in the keyvalue spec")
			keyValue.Spec.Servers = []string{altServer.ClientURL()}
			Expect(k8sClient.Update(ctx, keyValue)).To(Succeed())

			By("checking precondition, that the keyvalue does not yet exist")
			_, err := jsClient.KeyValue(ctx, keyValueName)
			Expect(err).To(MatchError(jetstream.ErrBucketNotFound))

			By("reconciling the resource")
			result, err := controller.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			connPool := newConnPool(0)
			conn, err := connPool.Get(&NatsConfig{ServerURL: altServer.ClientURL()}, true)
			Expect(err).NotTo(HaveOccurred())
			domain := ""

			By("checking if the keyvalue was created on the alternative server")
			altClient, err := CreateJetStreamClient(conn, true, domain)
			defer conn.Close()
			Expect(err).NotTo(HaveOccurred())

			_, err = altClient.KeyValue(ctx, keyValueName)
			Expect(err).NotTo(HaveOccurred())

			By("checking that the keyvalue was NOT created on the original server")
			_, err = jsClient.KeyValue(ctx, keyValueName)
			Expect(err).To(MatchError(jetstream.ErrBucketNotFound))
		})
	})
})

func Test_mapKVSpecToConfig(t *testing.T) {
	date := time.Date(2024, 12, 3, 16, 55, 5, 0, time.UTC)
	dateString := date.Format(time.RFC3339)

	tests := []struct {
		name    string
		spec    *api.KeyValueSpec
		want    jetstream.KeyValueConfig
		wantErr bool
	}{
		{
			name:    "empty spec",
			spec:    &api.KeyValueSpec{},
			want:    jetstream.KeyValueConfig{},
			wantErr: false,
		},
		{
			name: "full spec",
			spec: &api.KeyValueSpec{
				Description:  "kv description",
				History:      20,
				MaxValueSize: 1024,
				MaxBytes:     1048576,
				TTL:          "1h",
				Mirror: &api.StreamSource{
					Name:                  "mirror",
					OptStartSeq:           5,
					OptStartTime:          dateString,
					FilterSubject:         "orders",
					ExternalAPIPrefix:     "api",
					ExternalDeliverPrefix: "deliver",
					SubjectTransforms: []*api.SubjectTransform{{
						Source: "transform-source",
						Dest:   "transform-dest",
					}},
				},
				Bucket: "kv-name",
				Placement: &api.StreamPlacement{
					Cluster: "test-cluster",
					Tags:    []string{"tag"},
				},
				Replicas: 3,
				RePublish: &api.RePublish{
					Source:      "re-publish-source",
					Destination: "re-publish-dest",
					HeadersOnly: true,
				},
				Compression: true,
				Sources: []*api.StreamSource{{
					Name:                  "source",
					OptStartSeq:           5,
					OptStartTime:          dateString,
					FilterSubject:         "orders",
					ExternalAPIPrefix:     "api",
					ExternalDeliverPrefix: "deliver",
					SubjectTransforms: []*api.SubjectTransform{{
						Source: "transform-source",
						Dest:   "transform-dest",
					}},
				}},
				Storage: "memory",
				BaseStreamConfig: api.BaseStreamConfig{
					PreventDelete: false,
					PreventUpdate: false,
					ConnectionOpts: api.ConnectionOpts{
						Account: "",
						Creds:   "",
						Nkey:    "",
						Servers: nil,
						TLS:     api.TLS{},
					},
				},
			},
			want: jetstream.KeyValueConfig{
				Bucket:       "kv-name",
				Description:  "kv description",
				MaxBytes:     1048576,
				TTL:          time.Hour,
				MaxValueSize: 1024,
				History:      20,
				Storage:      jetstream.MemoryStorage,
				Replicas:     3,
				Placement: &jetstream.Placement{
					Cluster: "test-cluster",
					Tags:    []string{"tag"},
				},
				Mirror: &jetstream.StreamSource{
					Name:          "mirror",
					OptStartSeq:   5,
					OptStartTime:  &date,
					FilterSubject: "orders",
					SubjectTransforms: []jetstream.SubjectTransformConfig{{
						Source:      "transform-source",
						Destination: "transform-dest",
					}},
					External: &jetstream.ExternalStream{
						APIPrefix:     "api",
						DeliverPrefix: "deliver",
					},
					Domain: "",
				},
				Sources: []*jetstream.StreamSource{{
					Name:          "source",
					OptStartSeq:   5,
					OptStartTime:  &date,
					FilterSubject: "orders",
					SubjectTransforms: []jetstream.SubjectTransformConfig{{
						Source:      "transform-source",
						Destination: "transform-dest",
					}},
					External: &jetstream.ExternalStream{
						APIPrefix:     "api",
						DeliverPrefix: "deliver",
					},
					Domain: "",
				}},
				Compression: true,
				RePublish: &jetstream.RePublish{
					Source:      "re-publish-source",
					Destination: "re-publish-dest",
					HeadersOnly: true,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			got, err := keyValueSpecToConfig(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("keyValueSpecToConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare nested structs
			assert.EqualValues(tt.want, got)
		})
	}
}
