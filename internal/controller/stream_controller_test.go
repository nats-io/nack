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
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
)

var _ = Describe("Stream Controller", func() {

	// The test stream resource
	const resourceName = "test-stream"
	const streamName = "orders"
	typeNamespacedName := types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}
	stream := &api.Stream{}

	// The tested controller
	var controller *StreamReconciler

	// Config to create minimal nats stream
	emptyStreamConfig := jetstream.StreamConfig{
		Name:      streamName,
		Replicas:  1,
		Retention: jetstream.WorkQueuePolicy,
		Discard:   jetstream.DiscardOld,
		Storage:   jetstream.FileStorage,
	}

	BeforeEach(func(ctx SpecContext) {
		By("creating a test stream resource")
		err := k8sClient.Get(ctx, typeNamespacedName, stream)
		if err != nil && k8serrors.IsNotFound(err) {
			resource := &api.Stream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: api.StreamSpec{
					Name:        streamName,
					Replicas:    1,
					Subjects:    []string{"tests.*"},
					Description: "test stream",
					Retention:   "workqueue",
					Discard:     "old",
					Storage:     "file",
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			// Re-fetch stream
			Expect(k8sClient.Get(ctx, typeNamespacedName, stream)).To(Succeed())
		}

		By("checking precondition: nats stream does not exist")
		_, err = jsClient.Stream(ctx, streamName)
		Expect(err).To(MatchError(jetstream.ErrStreamNotFound))

		By("setting up the tested controller")
		controller = &StreamReconciler{
			baseController,
		}
	})

	AfterEach(func(ctx SpecContext) {
		By("removing the test stream resource")
		resource := &api.Stream{}
		err := k8sClient.Get(ctx, typeNamespacedName, resource)
		if err != nil {
			Expect(err).To(MatchError(k8serrors.IsNotFound, "Is not found"))
		} else {
			if controllerutil.ContainsFinalizer(resource, streamFinalizer) {
				By("removing the finalizer")
				controllerutil.RemoveFinalizer(resource, streamFinalizer)
				Expect(k8sClient.Update(ctx, resource)).To(Succeed())
			}

			By("removing the stream resource")
			Expect(k8sClient.Delete(ctx, resource)).
				To(SatisfyAny(
					Succeed(),
					MatchError(k8serrors.IsNotFound, "is not found"),
				))
		}

		By("deleting the nats stream")
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
			Eventually(func(ctx SpecContext) *api.Stream {
				_, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				got := &api.Stream{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, got)).To(Succeed())
				return got
			}).WithContext(ctx).
				Should(SatisfyAll(
					HaveField("Finalizers", HaveExactElements(streamFinalizer)),
					HaveField("Status.Conditions", Not(BeEmpty())),
				))

			By("validating the ready condition")
			// Fetch stream
			Expect(k8sClient.Get(ctx, typeNamespacedName, stream)).To(Succeed())
			Expect(stream.Status.Conditions).To(HaveLen(1))

			assertReadyStateMatches(stream.Status.Conditions[0], v1.ConditionUnknown, "Reconciling", "Starting reconciliation", time.Now())
		})

	})

	When("reconciling an initialized resource", func() {

		BeforeEach(func(ctx SpecContext) {
			By("initializing the stream resource")

			By("setting the finalizer")
			Expect(controllerutil.AddFinalizer(stream, streamFinalizer)).To(BeTrue())
			Expect(k8sClient.Update(ctx, stream)).To(Succeed())

			By("setting an unknown ready state")
			stream.Status.Conditions = []api.Condition{{
				Type:               readyCondType,
				Status:             v1.ConditionUnknown,
				Reason:             "Test",
				Message:            "start condition",
				LastTransitionTime: time.Now().Format(time.RFC3339Nano),
			}}
			Expect(k8sClient.Status().Update(ctx, stream)).To(Succeed())

		})

		It("should create a new stream", func(ctx SpecContext) {

			By("running Reconcile")
			result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			// Fetch resource
			Expect(k8sClient.Get(ctx, typeNamespacedName, stream)).To(Succeed())

			By("checking if the ready state was updated")
			Expect(stream.Status.Conditions).To(HaveLen(1))
			assertReadyStateMatches(stream.Status.Conditions[0], v1.ConditionTrue, "Reconciling", "created or updated", time.Now())

			By("checking if the observed generation matches")
			Expect(stream.Status.ObservedGeneration).To(Equal(stream.Generation))

			By("checking if the stream was created")
			natsStream, err := jsClient.Stream(ctx, streamName)
			Expect(err).NotTo(HaveOccurred())
			streamInfo, err := natsStream.Info(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(streamInfo.Config.Name).To(Equal(streamName))
			Expect(streamInfo.Config.Description).To(Equal("test stream"))
			Expect(streamInfo.Created).To(BeTemporally("~", time.Now(), time.Second))
		})

		When("PreventUpdate is set", func() {

			BeforeEach(func(ctx SpecContext) {
				By("setting preventDelete on the resource")
				stream.Spec.PreventUpdate = true
				Expect(k8sClient.Update(ctx, stream)).To(Succeed())
			})
			It("should not create the stream", func(ctx SpecContext) {
				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that no stream was created")
				_, err = jsClient.Stream(ctx, streamName)
				Expect(err).To(MatchError(jetstream.ErrStreamNotFound))
			})
			It("should not update the stream", func(ctx SpecContext) {
				By("creating the stream")
				_, err := jsClient.CreateStream(ctx, emptyStreamConfig)
				Expect(err).NotTo(HaveOccurred())

				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that stream was not updated")
				s, err := jsClient.Stream(ctx, streamName)
				Expect(s.CachedInfo().Config.Description).To(BeEmpty())
			})
		})

		When("read-only mode is enabled", func() {

			BeforeEach(func(ctx SpecContext) {
				By("setting read only on the controller")
				readOnly, err := NewJSController(k8sClient, &NatsConfig{ServerURL: testServer.ClientURL()}, &Config{ReadOnly: true})
				Expect(err).NotTo(HaveOccurred())
				controller = &StreamReconciler{
					JetStreamController: readOnly,
				}
			})

			It("should not create the stream", func(ctx SpecContext) {
				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that no stream was created")
				_, err = jsClient.Stream(ctx, streamName)
				Expect(err).To(MatchError(jetstream.ErrStreamNotFound))
			})
			It("should not update the stream", func(ctx SpecContext) {
				By("creating the stream")
				_, err := jsClient.CreateStream(ctx, emptyStreamConfig)
				Expect(err).NotTo(HaveOccurred())

				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that stream was not updated")
				s, err := jsClient.Stream(ctx, streamName)
				Expect(s.CachedInfo().Config.Description).To(BeEmpty())
			})
		})

		When("namespace restriction is enabled", func() {

			BeforeEach(func(ctx SpecContext) {
				By("setting a namespace on the resource")
				namespaced, err := NewJSController(k8sClient, &NatsConfig{ServerURL: testServer.ClientURL()}, &Config{Namespace: "other-namespace"})
				Expect(err).NotTo(HaveOccurred())
				controller = &StreamReconciler{
					JetStreamController: namespaced,
				}
			})

			It("should not create the stream", func(ctx SpecContext) {
				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that no stream was created")
				_, err = jsClient.Stream(ctx, streamName)
				Expect(err).To(MatchError(jetstream.ErrStreamNotFound))
			})
			It("should not update the stream", func(ctx SpecContext) {
				By("creating the stream")
				_, err := jsClient.CreateStream(ctx, emptyStreamConfig)
				Expect(err).NotTo(HaveOccurred())

				By("running Reconcile")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that stream was not updated")
				s, err := jsClient.Stream(ctx, streamName)
				Expect(s.CachedInfo().Config.Description).To(BeEmpty())
			})
		})

		It("should update an existing stream", func(ctx SpecContext) {
			By("reconciling once to create the stream")
			result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			// Fetch resource
			Expect(k8sClient.Get(ctx, typeNamespacedName, stream)).To(Succeed())
			previousTransitionTime := stream.Status.Conditions[0].LastTransitionTime

			By("updating the resource")
			stream.Spec.Description = "new description"
			Expect(k8sClient.Update(ctx, stream)).To(Succeed())

			By("reconciling the updated resource")
			result, err = controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			// Fetch resource
			Expect(k8sClient.Get(ctx, typeNamespacedName, stream)).To(Succeed())

			By("checking if the state transition time was not updated")
			Expect(stream.Status.Conditions).To(HaveLen(1))
			Expect(stream.Status.Conditions[0].LastTransitionTime).To(Equal(previousTransitionTime))

			By("checking if the observed generation matches")
			Expect(stream.Status.ObservedGeneration).To(Equal(stream.Generation))

			By("checking if the stream was updated")
			natsStream, err := jsClient.Stream(ctx, streamName)
			Expect(err).NotTo(HaveOccurred())

			streamInfo, err := natsStream.Info(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(streamInfo.Config.Description).To(Equal("new description"))
			// Other fields unchanged
			Expect(streamInfo.Config.Subjects).To(Equal([]string{"tests.*"}))
		})

		It("should set an error state when the nats server is not available", func(ctx SpecContext) {

			By("setting up controller with unavailable nats server")
			// Setup client for not running server
			// Use actual test server to ensure port not used by other service on test instance
			sv := CreateTestServer()
			base, err := NewJSController(k8sClient, &NatsConfig{ServerURL: sv.ClientURL()}, &Config{})
			Expect(err).NotTo(HaveOccurred())
			sv.Shutdown()

			controller := &StreamReconciler{
				base,
			}

			By("reconciling resource")
			result, err := controller.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(err).To(HaveOccurred()) // Will be re-queued with back-off

			// Fetch resource
			err = k8sClient.Get(ctx, typeNamespacedName, stream)
			Expect(err).NotTo(HaveOccurred())

			By("checking if the status was updated")
			Expect(stream.Status.Conditions).To(HaveLen(1))
			assertReadyStateMatches(
				stream.Status.Conditions[0],
				v1.ConditionFalse,
				"Errored",
				"create or update stream:",
				time.Now(),
			)

			By("checking if the observed generation does not match")
			Expect(stream.Status.ObservedGeneration).ToNot(Equal(stream.Generation))
		})

		When("the resource is marked for deletion", func() {

			BeforeEach(func(ctx SpecContext) {
				By("marking the resource for deletion")
				Expect(k8sClient.Delete(ctx, stream)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedName, stream)).To(Succeed()) // re-fetch after update
			})

			It("should succeed deleting a not existing stream", func(ctx SpecContext) {
				By("reconciling")
				result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.IsZero()).To(BeTrue())

				By("checking that the resource is deleted")
				Eventually(k8sClient.Get).
					WithArguments(ctx, typeNamespacedName, stream).
					ShouldNot(Succeed())
			})

			When("the underlying stream exists", func() {
				BeforeEach(func(ctx SpecContext) {
					By("creating the stream on the nats server")
					_, err := jsClient.CreateStream(ctx, emptyStreamConfig)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func(ctx SpecContext) {
					err := jsClient.DeleteStream(ctx, streamName)
					if err != nil {
						Expect(err).To(MatchError(jetstream.ErrStreamNotFound))
					}
				})

				It("should delete the stream", func(ctx SpecContext) {
					By("reconciling")
					result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.IsZero()).To(BeTrue())

					By("checking that the stream is deleted")
					_, err = jsClient.Stream(ctx, streamName)
					Expect(err).To(MatchError(jetstream.ErrStreamNotFound))

					By("checking that the resource is deleted")
					Eventually(k8sClient.Get).
						WithArguments(ctx, typeNamespacedName, stream).
						ShouldNot(Succeed())
				})

				When("PreventDelete is set", func() {
					BeforeEach(func(ctx SpecContext) {
						By("setting preventDelete on the resource")
						stream.Spec.PreventDelete = true
						Expect(k8sClient.Update(ctx, stream)).To(Succeed())
					})
					It("Should delete the resource and not delete the nats stream", func(ctx SpecContext) {
						By("reconciling")
						result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
						Expect(err).NotTo(HaveOccurred())
						Expect(result.IsZero()).To(BeTrue())

						By("checking that the stream is not deleted")
						_, err = jsClient.Stream(ctx, streamName)
						Expect(err).NotTo(HaveOccurred())

						By("checking that the resource is deleted")
						Eventually(k8sClient.Get).
							WithArguments(ctx, typeNamespacedName, stream).
							ShouldNot(Succeed())
					})
				})

				When("read only is set", func() {
					BeforeEach(func(ctx SpecContext) {
						By("setting read only on the controller")
						readOnly, err := NewJSController(k8sClient, &NatsConfig{ServerURL: testServer.ClientURL()}, &Config{ReadOnly: true})
						Expect(err).NotTo(HaveOccurred())
						controller = &StreamReconciler{
							JetStreamController: readOnly,
						}
					})
					It("should delete the resource and not delete the stream", func(ctx SpecContext) {

						By("reconciling")
						result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
						Expect(err).NotTo(HaveOccurred())
						Expect(result.IsZero()).To(BeTrue())

						By("checking that the stream is not deleted")
						_, err = jsClient.Stream(ctx, streamName)
						Expect(err).NotTo(HaveOccurred())

						By("checking that the resource is deleted")
						Eventually(k8sClient.Get).
							WithArguments(ctx, typeNamespacedName, stream).
							ShouldNot(Succeed())
					})
				})

				When("controller is restricted to different namespace", func() {
					BeforeEach(func(ctx SpecContext) {
						namespaced, err := NewJSController(k8sClient, &NatsConfig{ServerURL: testServer.ClientURL()}, &Config{Namespace: "other-namespace"})
						Expect(err).NotTo(HaveOccurred())
						controller = &StreamReconciler{
							JetStreamController: namespaced,
						}
					})
					It("should not delete the resource and stream", func(ctx SpecContext) {

						By("reconciling")
						result, err := controller.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
						Expect(err).NotTo(HaveOccurred())
						Expect(result.IsZero()).To(BeTrue())

						By("checking that the stream is not deleted")
						_, err = jsClient.Stream(ctx, streamName)
						Expect(err).NotTo(HaveOccurred())

						By("checking that the finalizer is not removed")
						Expect(k8sClient.Get(ctx, typeNamespacedName, stream)).To(Succeed())
						Expect(stream.Finalizers).To(ContainElement(streamFinalizer))
					})
				})
			})
		})

		It("should update stream on different server as specified in spec", func(ctx SpecContext) {
			By("setting up the alternative server")
			// Setup altClient for alternate server
			altServer := CreateTestServer()
			defer altServer.Shutdown()

			By("setting the server in the stream spec")
			stream.Spec.Servers = []string{altServer.ClientURL()}
			Expect(k8sClient.Update(ctx, stream)).To(Succeed())

			By("checking precondition, that the stream does not yet exist")
			got, err := jsClient.Stream(ctx, streamName)
			Expect(err).To(MatchError(jetstream.ErrStreamNotFound))

			By("reconciling the resource")
			result, err := controller.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			By("checking if the stream was created on the alternative server")
			altClient, closer, err := CreateJetStreamClient(&NatsConfig{ServerURL: altServer.ClientURL()}, true)
			defer closer.Close()
			Expect(err).NotTo(HaveOccurred())

			got, err = altClient.Stream(ctx, streamName)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.CachedInfo().Created).To(BeTemporally("~", time.Now(), time.Second))

			By("checking that the stream was NOT created on the original server")
			_, err = jsClient.Stream(ctx, streamName)
			Expect(err).To(MatchError(jetstream.ErrStreamNotFound))

		})
	})
})

func Test_mapSpecToConfig(t *testing.T) {

	date := time.Date(2024, 12, 03, 16, 55, 5, 0, time.UTC)
	dateString := date.Format(time.RFC3339)

	tests := []struct {
		name    string
		spec    *api.StreamSpec
		want    jetstream.StreamConfig
		wantErr bool
	}{
		{
			name:    "emtpy spec",
			spec:    &api.StreamSpec{},
			want:    jetstream.StreamConfig{},
			wantErr: false,
		},
		{
			name: "full spec",
			spec: &api.StreamSpec{
				Account:           "",
				AllowDirect:       true,
				AllowRollup:       true,
				Creds:             "",
				DenyDelete:        true,
				DenyPurge:         true,
				Description:       "stream description",
				DiscardPerSubject: true,
				PreventDelete:     false,
				PreventUpdate:     false,
				Discard:           "new",
				DuplicateWindow:   "5s",
				MaxAge:            "30s",
				MaxBytes:          -1,
				MaxConsumers:      -1,
				MaxMsgs:           -1,
				MaxMsgSize:        -1,
				MaxMsgsPerSubject: 10,
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
				Name:  "stream-name",
				Nkey:  "",
				NoAck: true,
				Placement: &api.StreamPlacement{
					Cluster: "test-cluster",
					Tags:    []string{"tag"},
				},
				Replicas: 3,
				Republish: &api.RePublish{
					Source:      "re-publish-source",
					Destination: "re-publish-dest",
					HeadersOnly: true,
				},
				SubjectTransform: &api.SubjectTransform{
					Source: "transform-source",
					Dest:   "transform-dest",
				},
				FirstSequence: 42,
				Compression:   "s2",
				Metadata: map[string]string{
					"meta": "data",
				},
				Retention: "interest",
				Servers:   nil,
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
				Storage:  "file",
				Subjects: []string{"orders.*"},
				TLS:      api.TLS{},
			},
			want: jetstream.StreamConfig{
				Name:                 "stream-name",
				Description:          "stream description",
				Subjects:             []string{"orders.*"},
				Retention:            jetstream.InterestPolicy,
				MaxConsumers:         -1,
				MaxMsgs:              -1,
				MaxBytes:             -1,
				Discard:              jetstream.DiscardNew,
				DiscardNewPerSubject: true,
				MaxAge:               time.Second * 30,
				MaxMsgsPerSubject:    10,
				MaxMsgSize:           -1,
				Storage:              jetstream.FileStorage,
				Replicas:             3,
				NoAck:                true,
				Duplicates:           time.Second * 5,
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
				Sealed:      false,
				DenyDelete:  true,
				DenyPurge:   true,
				AllowRollup: true,
				Compression: jetstream.S2Compression,
				FirstSeq:    42,
				SubjectTransform: &jetstream.SubjectTransformConfig{
					Source:      "transform-source",
					Destination: "transform-dest",
				},
				RePublish: &jetstream.RePublish{
					Source:      "re-publish-source",
					Destination: "re-publish-dest",
					HeadersOnly: true,
				},
				AllowDirect:    true,
				MirrorDirect:   false,
				ConsumerLimits: jetstream.StreamConsumerLimits{},
				Metadata: map[string]string{
					"meta": "data",
				},
				Template: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			got, err := mapSpecToConfig(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapSpecToConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare nested structs
			assert.EqualValues(tt.want, got)
		})
	}
}
