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

var _ = Describe("ObjectStore Controller", func() {
	// The test objectStore resource
	const resourceName = "test-objectstore"
	const objectStoreName = "orders"

	const alternateResource = "alternate-objectstore"
	const alternateNamespace = "alternate-namespace"

	typeNamespacedName := types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}
	objectStore := &api.ObjectStore{}

	// The tested controller
	var controller *ObjectStoreReconciler

	// Config to create minimal nats ObjectStore store
	emptyObjectStoreConfig := jetstream.ObjectStoreConfig{
		Bucket:   objectStoreName,
		Replicas: 1,
		Storage:  jetstream.FileStorage,
	}

	BeforeEach(func(ctx SpecContext) {
		By("creating a test objectstore resource")
		err := k8sClient.Get(ctx, typeNamespacedName, objectStore)
		if err != nil && k8serrors.IsNotFound(err) {
			resource := &api.ObjectStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: api.ObjectStoreSpec{
					Bucket:      objectStoreName,
					Replicas:    1,
					TTL:         "5m",
					Compression: true,
					Description: "test objectstore",
					Storage:     "file",
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			// Re-fetch ObjectStore
			Expect(k8sClient.Get(ctx, typeNamespacedName, objectStore)).To(Succeed())
		}

		By("checking precondition: nats objectstore does not exist")
		_, err = jsClient.ObjectStore(ctx, objectStoreName)
		Expect(err).To(MatchError(jetstream.ErrBucketNotFound))

		By("setting up the tested controller")
		controller = &ObjectStoreReconciler{
			Scheme:              k8sClient.Scheme(),
			JetStreamController: baseController,
		}
	})

	AfterEach(func(ctx SpecContext) {
		By("removing the test objectstore resource")
		resource := &api.ObjectStore{}
		err := k8sClient.Get(ctx, typeNamespacedName, resource)
		if err != nil {
			Expect(err).To(MatchError(k8serrors.IsNotFound, "Is not found"))
		} else {
			if controllerutil.ContainsFinalizer(resource, objectStoreFinalizer) {
				By("removing the finalizer")
				controllerutil.RemoveFinalizer(resource, objectStoreFinalizer)
				Expect(k8sClient.Update(ctx, resource)).To(Succeed())
			}

			By("removing the objectstore resource")
			Expect(k8sClient.Delete(ctx, resource)).
				To(SatisfyAny(
					Succeed(),
					MatchError(k8serrors.IsNotFound, "is not found"),
				))
		}

		By("deleting the nats objectstore store")
		Expect(jsClient.DeleteObjectStore(ctx, objectStoreName)).
			To(SatisfyAny(
				Succeed(),
				MatchError(jetstream.ErrStreamNotFound),
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
			Eventually(func(ctx SpecContext) *api.ObjectStore {
				_, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				got := &api.ObjectStore{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, got)).To(Succeed())
				return got
			}).WithContext(ctx).
				Should(SatisfyAll(
					HaveField("Status.Conditions", Not(BeEmpty())),
				))

			By("validating the ready condition")
			// Fetch ObjectStore
			Expect(k8sClient.Get(ctx, typeNamespacedName, objectStore)).To(Succeed())
			Expect(objectStore.Status.Conditions).To(HaveLen(1))

			assertReadyStateMatches(objectStore.Status.Conditions[0], v1.ConditionUnknown, stateReconciling, "Starting reconciliation", time.Now())
		})
	})

	When("reconciling a resource in a different namespace", func() {
		BeforeEach(func(ctx SpecContext) {
			By("creating an objectstore resource in an alternate namespace while namespaced")
			alternateNamespaceResource := &api.ObjectStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alternateResource,
					Namespace: alternateNamespace,
				},
				Spec: api.ObjectStoreSpec{
					Bucket:      alternateResource,
					Replicas:    1,
					TTL:         "5m",
					Description: "objectstore in alternate namespace",
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
			alternateObjectStore := &api.ObjectStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alternateResource,
					Namespace: alternateNamespace,
				},
			}
			err := k8sClient.Delete(ctx, alternateObjectStore)
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

			By("checking the objectstore doesn't exist in NATS")
			_, err = jsClient.ObjectStore(ctx, alternateResource)
			Expect(err).To(MatchError(jetstream.ErrBucketNotFound))

			By("verifying the resource still exists in the alternate namespace")
			alternateObjectStore := &api.ObjectStore{}
			Expect(k8sClient.Get(ctx, alternateNamespacedName, alternateObjectStore)).To(Succeed())

			By("checking no conditions were set on the resource")
			Expect(alternateObjectStore.Status.Conditions).To(BeEmpty())
		})

		It("should watch the resource in alternate namespace when not namespaced", func(ctx SpecContext) {
			By("reconciling with a non-namespaced controller")
			testNatsConfig := &NatsConfig{ServerURL: clientUrl}
			alternateBaseController, err := NewJSController(k8sClient, testNatsConfig, &Config{})
			Expect(err).NotTo(HaveOccurred())

			alternateController := &ObjectStoreReconciler{
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
			By("initializing the objectstore resource")

			By("setting the finalizer")
			Expect(controllerutil.AddFinalizer(objectStore, objectStoreFinalizer)).To(BeTrue())
			Expect(k8sClient.Update(ctx, objectStore)).To(Succeed())

			By("setting an unknown ready state")
			objectStore.Status.Conditions = []api.Condition{{
				Type:               readyCondType,
				Status:             v1.ConditionUnknown,
				Reason:             "Test",
				Message:            "start condition",
				LastTransitionTime: time.Now().Format(time.RFC3339Nano),
			}}
			Expect(k8sClient.Status().Update(ctx, objectStore)).To(Succeed())
		})

		It("should create a new objectstore store", func(ctx SpecContext) {
			By("running Reconcile")
			result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			// Fetch resource
			Expect(k8sClient.Get(ctx, typeNamespacedName, objectStore)).To(Succeed())

			By("checking if the ready state was updated")
			Expect(objectStore.Status.Conditions).To(HaveLen(1))
			assertReadyStateMatches(objectStore.Status.Conditions[0], v1.ConditionTrue, stateReady, "created or updated", time.Now())

			By("checking if the observed generation matches")
			Expect(objectStore.Status.ObservedGeneration).To(Equal(objectStore.Generation))

			By("checking if the objectstore store was created")
			natsObjectStore, err := jsClient.ObjectStore(ctx, objectStoreName)
			Expect(err).NotTo(HaveOccurred())
			objectstoreStatus, err := natsObjectStore.Status(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(objectstoreStatus.Bucket()).To(Equal(objectStoreName))
			Expect(objectstoreStatus.TTL()).To(Equal(5 * time.Minute))
			Expect(objectstoreStatus.IsCompressed()).To(BeTrue())
		})

		When("PreventUpdate is set", func() {
			BeforeEach(func(ctx SpecContext) {
				By("setting preventDelete on the resource")
				objectStore.Spec.PreventUpdate = true
				Expect(k8sClient.Update(ctx, objectStore)).To(Succeed())
			})
			It("should create the objectstore", func(ctx SpecContext) {
				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that objectstore was created")
				_, err = jsClient.ObjectStore(ctx, objectStoreName)
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not update the objectstore", func(ctx SpecContext) {
				By("creating the objectstore")
				_, err := jsClient.CreateObjectStore(ctx, emptyObjectStoreConfig)
				Expect(err).NotTo(HaveOccurred())

				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that objectstore was not updated")
				natsObjectStore, err := jsClient.ObjectStore(ctx, objectStoreName)
				Expect(err).NotTo(HaveOccurred())
				s, err := natsObjectStore.Status(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(s.IsCompressed()).To(BeFalse())
			})
		})

		When("read-only mode is enabled", func() {
			BeforeEach(func(ctx SpecContext) {
				By("setting read only on the controller")
				readOnly, err := NewJSController(k8sClient, &NatsConfig{ServerURL: clientUrl}, &Config{ReadOnly: true})
				Expect(err).NotTo(HaveOccurred())
				controller = &ObjectStoreReconciler{
					Scheme:              k8sClient.Scheme(),
					JetStreamController: readOnly,
				}
			})

			It("should not create the objectstore", func(ctx SpecContext) {
				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that no objectstore was created")
				_, err = jsClient.ObjectStore(ctx, objectStoreName)
				Expect(err).To(MatchError(jetstream.ErrBucketNotFound))
			})
			It("should not update the objectstore", func(ctx SpecContext) {
				By("creating the objectstore")
				_, err := jsClient.CreateObjectStore(ctx, emptyObjectStoreConfig)
				Expect(err).NotTo(HaveOccurred())

				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that objectstore was not updated")
				natsObjectStore, err := jsClient.ObjectStore(ctx, objectStoreName)
				Expect(err).NotTo(HaveOccurred())
				s, err := natsObjectStore.Status(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(s.IsCompressed()).To(BeFalse())
			})
		})

		When("namespace restriction is enabled", func() {
			BeforeEach(func(ctx SpecContext) {
				By("setting a namespace on the resource")
				namespaced, err := NewJSController(k8sClient, &NatsConfig{ServerURL: clientUrl}, &Config{Namespace: alternateNamespace})
				Expect(err).NotTo(HaveOccurred())
				controller = &ObjectStoreReconciler{
					Scheme:              k8sClient.Scheme(),
					JetStreamController: namespaced,
				}
			})

			It("should not create the objectstore", func(ctx SpecContext) {
				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that no objectstore was created")
				_, err = jsClient.ObjectStore(ctx, objectStoreName)
				Expect(err).To(MatchError(jetstream.ErrBucketNotFound))
			})
			It("should not update the objectstore", func(ctx SpecContext) {
				By("creating the objectstore")
				_, err := jsClient.CreateObjectStore(ctx, emptyObjectStoreConfig)
				Expect(err).NotTo(HaveOccurred())

				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that objectstore was not updated")
				natsObjectStore, err := jsClient.ObjectStore(ctx, objectStoreName)
				Expect(err).NotTo(HaveOccurred())
				s, err := natsObjectStore.Status(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(s.IsCompressed()).To(BeFalse())
			})
		})

		It("should update an existing objectstore", func(ctx SpecContext) {
			By("reconciling once to create the objectstore")
			result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			// Fetch resource
			Expect(k8sClient.Get(ctx, typeNamespacedName, objectStore)).To(Succeed())
			previousTransitionTime := objectStore.Status.Conditions[0].LastTransitionTime

			By("updating the resource")
			objectStore.Spec.Description = "new description"
			objectStore.Spec.TTL = "1h"
			Expect(k8sClient.Update(ctx, objectStore)).To(Succeed())

			By("reconciling the updated resource")
			result, err = controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			// Fetch resource
			Expect(k8sClient.Get(ctx, typeNamespacedName, objectStore)).To(Succeed())

			By("checking if the state transition time was not updated")
			Expect(objectStore.Status.Conditions).To(HaveLen(1))
			Expect(objectStore.Status.Conditions[0].LastTransitionTime).To(Equal(previousTransitionTime))

			By("checking if the observed generation matches")
			Expect(objectStore.Status.ObservedGeneration).To(Equal(objectStore.Generation))

			By("checking if the objectstore was updated")
			natsObjectStore, err := jsClient.ObjectStore(ctx, objectStoreName)
			Expect(err).NotTo(HaveOccurred())

			objectStoreStatus, err := natsObjectStore.Status(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(objectStoreStatus.Bucket()).To(Equal(objectStoreName))
			Expect(objectStoreStatus.TTL()).To(Equal(1 * time.Hour))
			Expect(objectStoreStatus.IsCompressed()).To(BeTrue())
		})

		It("should set an error state when the nats server is not available", func(ctx SpecContext) {
			By("setting up controller with unavailable nats server")
			// Setup client for not running server
			// Use actual test server to ensure port not used by other service on test instance
			sv := CreateTestServer()
			disconnectedController, err := NewJSController(k8sClient, &NatsConfig{ServerURL: sv.ClientURL()}, &Config{})
			Expect(err).NotTo(HaveOccurred())
			sv.Shutdown()

			controller := &ObjectStoreReconciler{
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
			err = k8sClient.Get(ctx, typeNamespacedName, objectStore)
			Expect(err).NotTo(HaveOccurred())

			By("checking if the status was updated")
			Expect(objectStore.Status.Conditions).To(HaveLen(1))
			assertReadyStateMatches(
				objectStore.Status.Conditions[0],
				v1.ConditionFalse,
				stateErrored,
				"create or update objectstore:",
				time.Now(),
			)

			By("checking if the observed generation does not match")
			Expect(objectStore.Status.ObservedGeneration).ToNot(Equal(objectStore.Generation))
		})

		When("the resource is marked for deletion", func() {
			BeforeEach(func(ctx SpecContext) {
				By("marking the resource for deletion")
				Expect(k8sClient.Delete(ctx, objectStore)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedName, objectStore)).To(Succeed()) // re-fetch after update
			})

			It("should succeed deleting a not existing objectstore", func(ctx SpecContext) {
				By("reconciling")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that the resource is deleted")
				Eventually(k8sClient.Get).
					WithArguments(ctx, typeNamespacedName, objectStore).
					ShouldNot(Succeed())
			})

			When("the underlying objectstore exists", func() {
				BeforeEach(func(ctx SpecContext) {
					By("creating the objectstore on the nats server")
					_, err := jsClient.CreateObjectStore(ctx, emptyObjectStoreConfig)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func(ctx SpecContext) {
					err := jsClient.DeleteObjectStore(ctx, objectStoreName)
					if err != nil {
						Expect(err).To(MatchError(jetstream.ErrStreamNotFound))
					}
				})

				It("should delete the objectstore", func(ctx SpecContext) {
					By("reconciling")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking that the objectstore is deleted")
					_, err = jsClient.ObjectStore(ctx, objectStoreName)
					Expect(err).To(MatchError(jetstream.ErrBucketNotFound))

					By("checking that the resource is deleted")
					Eventually(k8sClient.Get).
						WithArguments(ctx, typeNamespacedName, objectStore).
						ShouldNot(Succeed())
				})

				When("PreventDelete is set", func() {
					BeforeEach(func(ctx SpecContext) {
						By("setting preventDelete on the resource")
						objectStore.Spec.PreventDelete = true
						Expect(k8sClient.Update(ctx, objectStore)).To(Succeed())
					})
					It("Should delete the resource and not delete the nats objectstore", func(ctx SpecContext) {
						By("reconciling")
						result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
						Expect(err).NotTo(HaveOccurred())
						Expect(result.IsZero()).To(BeTrue())

						By("checking that the objectstore is not deleted")
						_, err = jsClient.ObjectStore(ctx, objectStoreName)
						Expect(err).NotTo(HaveOccurred())

						By("checking that the resource is deleted")
						Eventually(k8sClient.Get).
							WithArguments(ctx, typeNamespacedName, objectStore).
							ShouldNot(Succeed())
					})
				})

				When("read only is set", func() {
					BeforeEach(func(ctx SpecContext) {
						By("setting read only on the controller")
						readOnly, err := NewJSController(k8sClient, &NatsConfig{ServerURL: clientUrl}, &Config{ReadOnly: true})
						Expect(err).NotTo(HaveOccurred())
						controller = &ObjectStoreReconciler{
							Scheme:              k8sClient.Scheme(),
							JetStreamController: readOnly,
						}
					})
					It("should delete the resource and not delete the objectstore", func(ctx SpecContext) {
						By("reconciling")
						result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
						Expect(err).NotTo(HaveOccurred())
						Expect(result.IsZero()).To(BeTrue())

						By("checking that the objectstore is not deleted")
						_, err = jsClient.ObjectStore(ctx, objectStoreName)
						Expect(err).NotTo(HaveOccurred())

						By("checking that the resource is deleted")
						Eventually(k8sClient.Get).
							WithArguments(ctx, typeNamespacedName, objectStore).
							ShouldNot(Succeed())
					})
				})

				When("controller is restricted to different namespace", func() {
					BeforeEach(func(ctx SpecContext) {
						namespaced, err := NewJSController(k8sClient, &NatsConfig{ServerURL: clientUrl}, &Config{Namespace: alternateNamespace})
						Expect(err).NotTo(HaveOccurred())
						controller = &ObjectStoreReconciler{
							Scheme:              k8sClient.Scheme(),
							JetStreamController: namespaced,
						}
					})
					It("should not delete the resource and objectstore", func(ctx SpecContext) {
						By("reconciling")
						result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
						Expect(err).NotTo(HaveOccurred())
						Expect(result.IsZero()).To(BeTrue())

						By("checking that the objectstore is not deleted")
						_, err = jsClient.ObjectStore(ctx, objectStoreName)
						Expect(err).NotTo(HaveOccurred())

						By("checking that the finalizer is not removed")
						Expect(k8sClient.Get(ctx, typeNamespacedName, objectStore)).To(Succeed())
						Expect(objectStore.Finalizers).To(ContainElement(objectStoreFinalizer))
					})
				})
			})
		})

		It("should update objectstore on different server as specified in spec", func(ctx SpecContext) {
			By("setting up the alternative server")
			// Setup altClient for alternate server
			altServer := CreateTestServer()
			defer altServer.Shutdown()

			By("setting the server in the objectstore spec")
			objectStore.Spec.Servers = []string{altServer.ClientURL()}
			Expect(k8sClient.Update(ctx, objectStore)).To(Succeed())

			By("checking precondition, that the objectstore does not yet exist")
			_, err := jsClient.ObjectStore(ctx, objectStoreName)
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

			By("checking if the objectstore was created on the alternative server")
			altClient, err := CreateJetStreamClient(conn, true, domain)
			defer conn.Close()
			Expect(err).NotTo(HaveOccurred())

			_, err = altClient.ObjectStore(ctx, objectStoreName)
			Expect(err).NotTo(HaveOccurred())

			By("checking that the objectstore was NOT created on the original server")
			_, err = jsClient.ObjectStore(ctx, objectStoreName)
			Expect(err).To(MatchError(jetstream.ErrBucketNotFound))
		})
	})
})

func Test_mapobjectstoreSpecToConfig(t *testing.T) {
	tests := []struct {
		name    string
		spec    *api.ObjectStoreSpec
		want    jetstream.ObjectStoreConfig
		wantErr bool
	}{
		{
			name:    "empty spec",
			spec:    &api.ObjectStoreSpec{},
			want:    jetstream.ObjectStoreConfig{},
			wantErr: false,
		},
		{
			name: "full spec",
			spec: &api.ObjectStoreSpec{
				Description: "objectstore description",
				MaxBytes:    1048576,
				TTL:         "1h",
				Bucket:      "objectstore-name",
				Placement: &api.StreamPlacement{
					Cluster: "test-cluster",
					Tags:    []string{"tag"},
				},
				Replicas:    3,
				Compression: true,
				Storage:     "memory",
				Metadata: map[string]string{
					"foo": "bar",
				},
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
			want: jetstream.ObjectStoreConfig{
				Bucket:      "objectstore-name",
				Description: "objectstore description",
				MaxBytes:    1048576,
				TTL:         time.Hour,
				Storage:     jetstream.MemoryStorage,
				Replicas:    3,
				Placement: &jetstream.Placement{
					Cluster: "test-cluster",
					Tags:    []string{"tag"},
				},
				Compression: true,
				Metadata: map[string]string{
					"foo": "bar",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			got, err := objectStoreSpecToConfig(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("objectStoreSpecToConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare nested structs
			assert.EqualValues(tt.want, got)
		})
	}
}
