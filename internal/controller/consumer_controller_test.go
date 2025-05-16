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

	jsmapi "github.com/nats-io/jsm.go/api"
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

var _ = Describe("Consumer Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-consumer"

		const streamName = "orders"
		const consumerName = "test-consumer"

		const alternateResource = "alternate-consumer"
		const alternateNamespace = "alternate-namespace"

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		consumer := &api.Consumer{}

		emptyStreamConfig := jetstream.StreamConfig{
			Name:      streamName,
			Replicas:  1,
			Retention: jetstream.WorkQueuePolicy,
			Discard:   jetstream.DiscardOld,
			Storage:   jetstream.FileStorage,
		}

		emptyConsumerConfig := jetstream.ConsumerConfig{
			Durable: consumerName,
		}

		// Tested controller
		var controller *ConsumerReconciler

		BeforeEach(func(ctx SpecContext) {
			By("creating the custom resource for the Kind Consumer")
			err := k8sClient.Get(ctx, typeNamespacedName, consumer)
			if err != nil && k8serrors.IsNotFound(err) {
				resource := &api.Consumer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: api.ConsumerSpec{
						AckPolicy:     "explicit",
						DeliverPolicy: "all",
						DurableName:   consumerName,
						Description:   "test consumer",
						StreamName:    streamName,
						ReplayPolicy:  "instant",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				// Fetch consumer
				Expect(k8sClient.Get(ctx, typeNamespacedName, consumer)).To(Succeed())
			}

			By("creating the underlying stream")
			_, err = jsClient.CreateStream(ctx, emptyStreamConfig)
			Expect(err).ToNot(HaveOccurred())

			By("setting up the tested controller")
			controller = &ConsumerReconciler{
				Scheme:              k8sClient.Scheme(),
				JetStreamController: baseController,
			}
		})

		AfterEach(func(ctx SpecContext) {
			By("removing the consumer resource")
			resource := &api.Consumer{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err != nil {
				Expect(err).To(MatchError(k8serrors.IsNotFound, "Is not found"))
			} else {
				if controllerutil.ContainsFinalizer(resource, consumerFinalizer) {
					By("removing the finalizer")
					controllerutil.RemoveFinalizer(resource, consumerFinalizer)
					Expect(k8sClient.Update(ctx, resource)).To(Succeed())
				}

				By("removing the consumer resource")
				Expect(k8sClient.Delete(ctx, resource)).
					To(SatisfyAny(
						Succeed(),
						MatchError(k8serrors.IsNotFound, "is not found"),
					))
			}

			By("deleting the nats consumer")
			Expect(jsClient.DeleteConsumer(ctx, streamName, consumerName)).
				To(SatisfyAny(
					Succeed(),
					MatchError(jetstream.ErrStreamNotFound),
					MatchError(jetstream.ErrConsumerNotFound),
				))

			By("deleting the consumers nats stream")
			Expect(jsClient.DeleteStream(ctx, streamName)).
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
				Eventually(func(ctx SpecContext) *api.Consumer {
					_, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					got := &api.Consumer{}
					Expect(k8sClient.Get(ctx, typeNamespacedName, got)).To(Succeed())
					return got
				}).WithContext(ctx).
					Within(time.Second).
					Should(SatisfyAll(
						HaveField("Finalizers", HaveExactElements(consumerFinalizer)),
						HaveField("Status.Conditions", Not(BeEmpty())),
					))

				By("validating the ready condition")
				// Fetch consumer
				Expect(k8sClient.Get(ctx, typeNamespacedName, consumer)).To(Succeed())
				Expect(consumer.Status.Conditions).To(HaveLen(1))

				assertReadyStateMatches(consumer.Status.Conditions[0], v1.ConditionUnknown, stateReconciling, "Starting reconciliation", time.Now())
			})
		})

		When("reconciling a resource in a different namespace", func() {
			BeforeEach(func(ctx SpecContext) {
				By("creating a consumer resource in an alternate namespace while namespaced")
				alternateNamespaceResource := &api.Consumer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      alternateResource,
						Namespace: alternateNamespace,
					},
					Spec: api.ConsumerSpec{
						AckPolicy:     "explicit",
						DeliverPolicy: "all",
						DurableName:   alternateResource,
						Description:   "consumer in alternate namespace",
						StreamName:    streamName,
						ReplayPolicy:  "instant",
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
				alternateConsumer := &api.Consumer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      alternateResource,
						Namespace: alternateNamespace,
					},
				}
				err := k8sClient.Delete(ctx, alternateConsumer)
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

				By("checking the consumer doesn't exist in NATS")
				_, err = jsClient.Consumer(ctx, streamName, alternateResource)
				Expect(err).To(MatchError(jetstream.ErrConsumerNotFound))

				By("verifying the resource still exists in the alternate namespace")
				alternateConsumer := &api.Consumer{}
				Expect(k8sClient.Get(ctx, alternateNamespacedName, alternateConsumer)).To(Succeed())

				By("checking no conditions were set on the resource")
				Expect(alternateConsumer.Status.Conditions).To(BeEmpty())
			})

			It("should watch the resource in alternate namespace when not namespaced", func(ctx SpecContext) {
				By("reconciling with a non-namespaced controller")
				testNatsConfig := &NatsConfig{ServerURL: clientUrl}
				alternateBaseController, err := NewJSController(k8sClient, testNatsConfig, &Config{})
				Expect(err).NotTo(HaveOccurred())

				alternateController := &ConsumerReconciler{
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
				By("initializing the stream resource")

				By("setting the finalizer")
				Expect(controllerutil.AddFinalizer(consumer, consumerFinalizer)).To(BeTrue())
				Expect(k8sClient.Update(ctx, consumer)).To(Succeed())

				By("setting an unknown ready state")
				consumer.Status.Conditions = []api.Condition{{
					Type:               readyCondType,
					Status:             v1.ConditionUnknown,
					Reason:             "Test",
					Message:            "start condition",
					LastTransitionTime: time.Now().Format(time.RFC3339Nano),
				}}
				Expect(k8sClient.Status().Update(ctx, consumer)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedName, consumer)).To(Succeed())
			})

			When("the underlying stream does not exist", func() {
				It("should set false ready state and error", func(ctx SpecContext) {
					By("setting a not existing stream on the resource")
					consumer.Spec.StreamName = "not-existing"
					Expect(k8sClient.Update(ctx, consumer)).To(Succeed())

					By("running Reconcile")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).To(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking for expected ready state")
					Expect(k8sClient.Get(ctx, typeNamespacedName, consumer)).To(Succeed())
					Expect(consumer.Status.Conditions).To(HaveLen(1))
					assertReadyStateMatches(
						consumer.Status.Conditions[0],
						v1.ConditionFalse,
						stateErrored,
						"stream", // Not existing stream as message
						time.Now(),
					)
				})
			})

			It("should create a new consumer", func(ctx SpecContext) {
				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				// Fetch resource
				Expect(k8sClient.Get(ctx, typeNamespacedName, consumer)).To(Succeed())

				By("checking if the ready state was updated")
				Expect(consumer.Status.Conditions).To(HaveLen(1))
				assertReadyStateMatches(consumer.Status.Conditions[0], v1.ConditionTrue, stateReady, "created or updated", time.Now())

				By("checking if the observed generation matches")
				Expect(consumer.Status.ObservedGeneration).To(Equal(consumer.Generation))

				By("checking if the consumer was created")
				natsconsumer, err := jsClient.Consumer(ctx, streamName, consumerName)
				Expect(err).NotTo(HaveOccurred())
				consumerInfo, err := natsconsumer.Info(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(consumerInfo.Config.Name).To(Equal(consumerName))
				Expect(consumerInfo.Config.Description).To(Equal("test consumer"))
				Expect(consumerInfo.Created).To(BeTemporally("~", time.Now(), time.Second))
			})

			It("should update an existing consumer", func(ctx SpecContext) {
				By("reconciling once to create the consumer")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				// Fetch resource
				Expect(k8sClient.Get(ctx, typeNamespacedName, consumer)).To(Succeed())
				previousTransitionTime := consumer.Status.Conditions[0].LastTransitionTime

				By("updating the resource")
				consumer.Spec.Description = "new description"
				Expect(k8sClient.Update(ctx, consumer)).To(Succeed())

				By("reconciling the updated resource")
				result, err = controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				// Fetch resource
				Expect(k8sClient.Get(ctx, typeNamespacedName, consumer)).To(Succeed())

				By("checking if the state transition time was not updated")
				Expect(consumer.Status.Conditions).To(HaveLen(1))
				Expect(consumer.Status.Conditions[0].LastTransitionTime).To(Equal(previousTransitionTime))

				By("checking if the observed generation matches")
				Expect(consumer.Status.ObservedGeneration).To(Equal(consumer.Generation))

				By("checking if the consumer was updated")
				natsStream, err := jsClient.Consumer(ctx, streamName, consumerName)
				Expect(err).NotTo(HaveOccurred())

				streamInfo, err := natsStream.Info(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(streamInfo.Config.Description).To(Equal("new description"))
				// Other fields unchanged
				Expect(streamInfo.Config.ReplayPolicy).To(Equal(jetstream.ReplayInstantPolicy))
			})

			When("PreventUpdate is set", func() {
				BeforeEach(func(ctx SpecContext) {
					By("setting preventUpdate on the resource")
					consumer.Spec.PreventUpdate = true
					Expect(k8sClient.Update(ctx, consumer)).To(Succeed())
				})
				It("should create the consumer", func(ctx SpecContext) {
					By("running Reconcile")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking that consumer was created")
					_, err = jsClient.Consumer(ctx, streamName, consumerName)
					Expect(err).ToNot(HaveOccurred())
				})
				It("should not update the consumer", func(ctx SpecContext) {
					By("creating the consumer")
					_, err := jsClient.CreateConsumer(ctx, streamName, emptyConsumerConfig)
					Expect(err).NotTo(HaveOccurred())

					By("running Reconcile")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking that consumer was not updated")
					c, err := jsClient.Consumer(ctx, streamName, consumerName)
					Expect(err).NotTo(HaveOccurred())
					Expect(c.CachedInfo().Config.Description).To(BeEmpty())
				})
			})

			When("read-only mode is enabled", func() {
				BeforeEach(func(ctx SpecContext) {
					By("setting read only on the controller")
					readOnly, err := NewJSController(k8sClient, &NatsConfig{ServerURL: clientUrl}, &Config{ReadOnly: true})
					Expect(err).NotTo(HaveOccurred())
					controller = &ConsumerReconciler{
						Scheme:              k8sClient.Scheme(),
						JetStreamController: readOnly,
					}
				})

				It("should not create the consumer", func(ctx SpecContext) {
					By("running Reconcile")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking that no consumer was created")
					_, err = jsClient.Consumer(ctx, streamName, consumerName)
					Expect(err).To(MatchError(jetstream.ErrConsumerNotFound))
				})
				It("should not update the consumer", func(ctx SpecContext) {
					By("creating the consumer")
					_, err := jsClient.CreateConsumer(ctx, streamName, emptyConsumerConfig)
					Expect(err).NotTo(HaveOccurred())

					By("running Reconcile")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking that consumer was not updated")
					s, err := jsClient.Consumer(ctx, streamName, consumerName)
					Expect(err).NotTo(HaveOccurred())
					Expect(s.CachedInfo().Config.Description).To(BeEmpty())
				})
			})

			When("namespace restriction is enabled", func() {
				BeforeEach(func(ctx SpecContext) {
					By("setting a namespace on the resource")
					namespaced, err := NewJSController(k8sClient, &NatsConfig{ServerURL: clientUrl}, &Config{Namespace: alternateNamespace})
					Expect(err).NotTo(HaveOccurred())
					controller = &ConsumerReconciler{
						Scheme:              k8sClient.Scheme(),
						JetStreamController: namespaced,
					}
				})

				It("should not create the consumer", func(ctx SpecContext) {
					By("running Reconcile")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking that no consumer was created")
					_, err = jsClient.Consumer(ctx, streamName, consumerName)
					Expect(err).To(MatchError(jetstream.ErrConsumerNotFound))
				})
				It("should not update the consumer", func(ctx SpecContext) {
					By("creating the consumer")
					_, err := jsClient.CreateConsumer(ctx, streamName, emptyConsumerConfig)
					Expect(err).NotTo(HaveOccurred())

					By("running Reconcile")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking that consumer was not updated")
					s, err := jsClient.Consumer(ctx, streamName, consumerName)
					Expect(err).NotTo(HaveOccurred())
					Expect(s.CachedInfo().Config.Description).To(BeEmpty())
				})
			})

			When("the resource is marked for deletion", func() {
				BeforeEach(func(ctx SpecContext) {
					By("marking the resource for deletion")
					Expect(k8sClient.Delete(ctx, consumer)).To(Succeed())
					Expect(k8sClient.Get(ctx, typeNamespacedName, consumer)).To(Succeed()) // re-fetch after update
				})

				It("should succeed deleting a not existing consumer", func(ctx SpecContext) {
					By("reconciling")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking that the resource is deleted")
					Eventually(k8sClient.Get).
						WithArguments(ctx, typeNamespacedName, consumer).
						ShouldNot(Succeed())
				})

				It("should succeed deleting a consumer of a deleted stream", func(ctx SpecContext) {
					By("Setting not existing stream")
					consumer.Spec.StreamName = "deleted-stream"
					Expect(k8sClient.Update(ctx, consumer)).To(Succeed())

					By("reconciling")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking that the resource is deleted")
					Eventually(k8sClient.Get).
						WithArguments(ctx, typeNamespacedName, consumer).
						ShouldNot(Succeed())
				})

				When("the underlying consumer exists", func() {
					BeforeEach(func(ctx SpecContext) {
						By("creating the consumer on the nats server")
						_, err := jsClient.CreateConsumer(ctx, streamName, emptyConsumerConfig)
						Expect(err).NotTo(HaveOccurred())
					})

					AfterEach(func(ctx SpecContext) {
						err := jsClient.DeleteConsumer(ctx, streamName, consumerName)
						if err != nil {
							Expect(err).To(MatchError(jetstream.ErrConsumerNotFound))
						}
					})

					It("should delete the consumer", func(ctx SpecContext) {
						By("reconciling")
						result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
						Expect(err).NotTo(HaveOccurred())
						Expect(result.IsZero()).To(BeTrue())

						By("checking that the consumer is deleted")
						_, err = jsClient.Consumer(ctx, streamName, consumerName)
						Expect(err).To(MatchError(jetstream.ErrConsumerNotFound))

						By("checking that the resource is deleted")
						Eventually(k8sClient.Get).
							WithArguments(ctx, typeNamespacedName, consumer).
							ShouldNot(Succeed())
					})

					When("PreventDelete is set", func() {
						BeforeEach(func(ctx SpecContext) {
							By("setting preventDelete on the resource")
							consumer.Spec.PreventDelete = true
							Expect(k8sClient.Update(ctx, consumer)).To(Succeed())
						})
						It("Should delete the resource and not delete the nats consumer", func(ctx SpecContext) {
							By("reconciling")
							result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
							Expect(err).NotTo(HaveOccurred())
							Expect(result.IsZero()).To(BeTrue())

							By("checking that the consumer is not deleted")
							_, err = jsClient.Consumer(ctx, streamName, consumerName)
							Expect(err).NotTo(HaveOccurred())

							By("checking that the resource is deleted")
							Eventually(k8sClient.Get).
								WithArguments(ctx, typeNamespacedName, consumer).
								ShouldNot(Succeed())
						})
					})

					When("read only is set", func() {
						BeforeEach(func(ctx SpecContext) {
							By("setting read only on the controller")
							readOnly, err := NewJSController(k8sClient, &NatsConfig{ServerURL: testServer.ClientURL()}, &Config{ReadOnly: true})
							Expect(err).NotTo(HaveOccurred())
							controller = &ConsumerReconciler{
								Scheme:              k8sClient.Scheme(),
								JetStreamController: readOnly,
							}
						})
						It("should delete the resource and not delete the consumer", func(ctx SpecContext) {
							By("reconciling")
							result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
							Expect(err).NotTo(HaveOccurred())
							Expect(result.IsZero()).To(BeTrue())

							By("checking that the consumer is not deleted")
							_, err = jsClient.Consumer(ctx, streamName, consumerName)
							Expect(err).NotTo(HaveOccurred())

							By("checking that the resource is deleted")
							Eventually(k8sClient.Get).
								WithArguments(ctx, typeNamespacedName, consumer).
								ShouldNot(Succeed())
						})
					})

					When("controller is restricted to different namespace", func() {
						BeforeEach(func(ctx SpecContext) {
							namespaced, err := NewJSController(k8sClient, &NatsConfig{ServerURL: testServer.ClientURL()}, &Config{Namespace: alternateNamespace})
							Expect(err).NotTo(HaveOccurred())
							controller = &ConsumerReconciler{
								Scheme:              k8sClient.Scheme(),
								JetStreamController: namespaced,
							}
						})
						It("should not delete the resource and consumer", func(ctx SpecContext) {
							By("reconciling")
							result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
							Expect(err).NotTo(HaveOccurred())
							Expect(result.IsZero()).To(BeTrue())

							By("checking that the consumer is not deleted")
							_, err = jsClient.Consumer(ctx, streamName, consumerName)
							Expect(err).NotTo(HaveOccurred())

							By("checking that the finalizer is not removed")
							Expect(k8sClient.Get(ctx, typeNamespacedName, consumer)).To(Succeed())
							Expect(consumer.Finalizers).To(ContainElement(consumerFinalizer))
						})
					})
				})
			})

			It("should create consumer on different server as specified in spec", func(ctx SpecContext) {
				By("setting up the alternative server")
				altServer := CreateTestServer()
				defer altServer.Shutdown()

				connPool := newConnPool(0)
				conn, err := connPool.Get(&NatsConfig{ServerURL: altServer.ClientURL()}, true)
				Expect(err).NotTo(HaveOccurred())
				domain := ""

				// Setup altClient for alternate server
				altClient, err := CreateJetStreamClient(conn, true, domain)
				defer conn.Close()
				Expect(err).NotTo(HaveOccurred())

				By("setting up the stream on the alternative server")
				_, err = altClient.CreateStream(ctx, emptyStreamConfig)
				Expect(err).NotTo(HaveOccurred())

				By("setting the server in the consumer spec")
				consumer.Spec.Servers = []string{altServer.ClientURL()}
				Expect(k8sClient.Update(ctx, consumer)).To(Succeed())

				By("checking precondition, that the consumer does not yet exist")
				_, err = jsClient.Consumer(ctx, streamName, consumerName)
				Expect(err).To(MatchError(jetstream.ErrConsumerNotFound))

				By("reconciling the resource")
				result, err := controller.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking if the consumer was created on the alternative server")
				got, err := altClient.Consumer(ctx, streamName, consumerName)
				Expect(err).NotTo(HaveOccurred())
				Expect(got.CachedInfo().Created).To(BeTemporally("~", time.Now(), time.Second))

				By("checking that the consumer was NOT created on the original server")
				_, err = jsClient.Consumer(ctx, streamName, consumerName)
				Expect(err).To(MatchError(jetstream.ErrConsumerNotFound))
			})
		})
	})
})

func Test_consumerSpecToConfig(t *testing.T) {
	date := time.Date(2024, 12, 3, 16, 55, 5, 0, time.UTC)
	dateString := date.Format(time.RFC3339)

	tests := []struct {
		name    string
		spec    *api.ConsumerSpec
		want    *jsmapi.ConsumerConfig
		wantErr bool
	}{
		{
			name:    "empty spec",
			spec:    &api.ConsumerSpec{},
			want:    &jsmapi.ConsumerConfig{},
			wantErr: false,
		},
		{
			name: "full spec",
			spec: &api.ConsumerSpec{
				AckPolicy:          "explicit",
				AckWait:            "10ns",
				BackOff:            []string{"1s", "5m"},
				DeliverGroup:       "",
				DeliverPolicy:      "byStartSequence",
				DeliverSubject:     "",
				Description:        "test consumer",
				DurableName:        "test-consumer",
				FilterSubject:      "",
				FilterSubjects:     []string{"time.us.east", "time.us.west"},
				FlowControl:        false,
				HeadersOnly:        true,
				HeartbeatInterval:  "",
				MaxAckPending:      6,
				MaxDeliver:         3,
				MaxRequestBatch:    7,
				MaxRequestExpires:  "8s",
				MaxRequestMaxBytes: 1024,
				MaxWaiting:         5,
				MemStorage:         true,
				OptStartSeq:        17,
				OptStartTime:       "",
				RateLimitBps:       512,
				ReplayPolicy:       "instant",
				Replicas:           9,
				SampleFreq:         "25%",
				StreamName:         "",
				Metadata: map[string]string{
					"meta": "data",
				},
				BaseStreamConfig: api.BaseStreamConfig{
					PreventDelete: false,
					PreventUpdate: false,
					ConnectionOpts: api.ConnectionOpts{
						Account: "",
						Creds:   "",
						Nkey:    "",
						TLS:     api.TLS{},
						Servers: nil,
					},
				},
			},
			want: &jsmapi.ConsumerConfig{
				Durable:            "test-consumer",
				Description:        "test consumer",
				DeliverPolicy:      jsmapi.DeliverByStartSequence,
				OptStartSeq:        17,
				AckPolicy:          jsmapi.AckExplicit,
				AckWait:            10 * time.Nanosecond,
				MaxDeliver:         3,
				BackOff:            []time.Duration{time.Second, 5 * time.Minute},
				FilterSubject:      "",
				ReplayPolicy:       jsmapi.ReplayInstant,
				RateLimit:          512,
				SampleFrequency:    "25%",
				MaxWaiting:         5,
				MaxAckPending:      6,
				HeadersOnly:        true,
				MaxRequestBatch:    7,
				MaxRequestExpires:  8 * time.Second,
				MaxRequestMaxBytes: 1024,
				InactiveThreshold:  0, // TODO no value?
				Replicas:           9,
				MemoryStorage:      true,
				FilterSubjects:     []string{"time.us.east", "time.us.west"},
				Metadata: map[string]string{
					"meta": "data",
				},
			},
			wantErr: false,
		},
		{
			name: "full spec alt",
			spec: &api.ConsumerSpec{
				AckPolicy:          "all",
				AckWait:            "20ns",
				BackOff:            []string{"1s", "5m"},
				DeliverGroup:       "",
				DeliverPolicy:      "byStartTime",
				DeliverSubject:     "",
				Description:        "test consumer",
				DurableName:        "test-consumer",
				FilterSubject:      "time.us.>",
				FlowControl:        true,
				HeadersOnly:        false,
				HeartbeatInterval:  "",
				MaxAckPending:      5,
				MaxDeliver:         6,
				MaxRequestBatch:    7,
				MaxRequestExpires:  "8s",
				MaxRequestMaxBytes: 1024,
				MaxWaiting:         5,
				MemStorage:         false,
				OptStartSeq:        17,
				OptStartTime:       dateString,
				RateLimitBps:       1024,
				ReplayPolicy:       "original",
				Replicas:           9,
				SampleFreq:         "30%",
				StreamName:         "",
				Metadata: map[string]string{
					"meta": "data",
				},
				BaseStreamConfig: api.BaseStreamConfig{
					PreventDelete: false,
					PreventUpdate: false,
					ConnectionOpts: api.ConnectionOpts{
						Account: "",
						Creds:   "",
						Nkey:    "",
						TLS:     api.TLS{},
						Servers: nil,
					},
				},
			},
			want: &jsmapi.ConsumerConfig{
				Durable:            "test-consumer",
				Description:        "test consumer",
				DeliverPolicy:      jsmapi.DeliverByStartTime,
				OptStartSeq:        0,
				OptStartTime:       &date,
				AckPolicy:          jsmapi.AckAll,
				AckWait:            20 * time.Nanosecond,
				MaxDeliver:         6,
				BackOff:            []time.Duration{time.Second, 5 * time.Minute},
				FlowControl:        true,
				FilterSubject:      "time.us.>",
				ReplayPolicy:       jsmapi.ReplayOriginal,
				RateLimit:          1024,
				SampleFrequency:    "30%",
				MaxWaiting:         5,
				MaxAckPending:      5,
				HeadersOnly:        false,
				MaxRequestBatch:    7,
				MaxRequestExpires:  8 * time.Second,
				MaxRequestMaxBytes: 1024,
				InactiveThreshold:  0, // TODO no value?
				Replicas:           9,
				MemoryStorage:      false,
				Metadata: map[string]string{
					"meta": "data",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cOpts, err := consumerSpecToConfig(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("consumerSpecToConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got := &jsmapi.ConsumerConfig{}
			for _, o := range cOpts {
				o(got)
			}

			assert.EqualValues(t, tt.want, got, "consumerSpecToConfig(%v)", tt.spec)
		})
	}
}
