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
	When("reconciling a resource", func() {
		const resourceName = "test-stream"
		const streamName = "orders"

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}

		emptyStreamConfig := jetstream.StreamConfig{
			Name:      streamName,
			Replicas:  1,
			Retention: jetstream.WorkQueuePolicy,
			Discard:   jetstream.DiscardOld,
			Storage:   jetstream.FileStorage,
		}

		stream := &api.Stream{}

		BeforeEach(func(ctx SpecContext) {
			By("creating the custom resource for the Kind Stream")
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
		})

		AfterEach(func(ctx SpecContext) {
			// Get and remove test stream resource if it exists
			resource := &api.Stream{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err != nil {
				Expect(err).To(MatchError(k8serrors.IsNotFound, "Is not found"))
			} else {
				By("Removing the finalizer")
				controllerutil.RemoveFinalizer(resource, streamFinalizer)
				Expect(k8sClient.Update(ctx, resource)).To(Succeed())

				By("Cleanup the specific resource instance Stream")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}

			By("Deleting the stream")
			err = jsClient.DeleteStream(ctx, streamName)
			if err != nil {
				Expect(err).To(MatchError(jetstream.ErrStreamNotFound))
			}
		})

		It("should successfully create the stream", func(ctx SpecContext) {
			By("Reconciling the created resource")
			controllerReconciler := &StreamReconciler{
				baseController,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Fetch resource
			err = k8sClient.Get(ctx, typeNamespacedName, stream)
			Expect(err).NotTo(HaveOccurred())

			By("Checking if the ready state was updated")
			readyCondition := api.Condition{
				Type:    readyCondType,
				Status:  v1.ConditionTrue,
				Reason:  "Reconciling",
				Message: "Stream successfully created or updated",
			}
			gotCondition := stream.Status.Conditions[0]
			// Unset LastTransitionTime to be equal for assertion
			gotCondition.LastTransitionTime = ""
			Expect(gotCondition).To(Equal(readyCondition))

			By("Checking if the observed generation matches")
			Expect(stream.Status.ObservedGeneration).To(Equal(stream.Generation))

			By("Checking if the finalizer was set")
			Expect(stream.Finalizers).To(ContainElement(streamFinalizer))

			By("Checking if the stream was created")
			natsStream, err := jsClient.Stream(ctx, streamName)
			Expect(err).NotTo(HaveOccurred())
			streamInfo, err := natsStream.Info(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(streamInfo.Config.Name).To(Equal(streamName))
			Expect(streamInfo.Created).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("should successfully update the stream and update the status", func(ctx SpecContext) {

			By("Creating the stream with empty subjects and description")
			_, err := jsClient.CreateStream(ctx, emptyStreamConfig)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the created resource")
			controllerReconciler := &StreamReconciler{
				baseController,
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Fetch resource
			err = k8sClient.Get(ctx, typeNamespacedName, stream)
			Expect(err).NotTo(HaveOccurred())

			By("Checking if the ready state was updated")
			readyCondition := api.Condition{
				Type:    readyCondType,
				Status:  v1.ConditionTrue,
				Reason:  "Reconciling",
				Message: "Stream successfully created or updated",
			}
			gotCondition := stream.Status.Conditions[0]
			// Unset LastTransitionTime to be equal for assertion
			gotCondition.LastTransitionTime = ""
			Expect(gotCondition).To(Equal(readyCondition))

			By("Checking if the observed generation matches")
			Expect(stream.Status.ObservedGeneration).To(Equal(stream.Generation))

			By("Checking if the stream was updated")
			natsStream, err := jsClient.Stream(ctx, streamName)
			Expect(err).NotTo(HaveOccurred())

			streamInfo, err := natsStream.Info(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(streamInfo.Config.Description).To(Equal("test stream"))
			Expect(streamInfo.Config.Subjects).To(Equal([]string{"tests.*"}))
		})

		It("should not fail on not existing resource", func(ctx SpecContext) {
			By("Reconciling the created resource")
			controllerReconciler := &StreamReconciler{
				baseController,
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "fake",
					Name:      "not-existing",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("should set an error state, register finalizer and re-queue when the nats server is not available", func(ctx SpecContext) {
			By("Reconciling the created resource")

			// Setup client for not running server
			// Use actual test server to ensure port not used by other service on test instance
			sv := CreateTestServer()
			controller, err := NewJSController(k8sClient, &NatsConfig{ServerURL: sv.ClientURL()}, &Config{})
			Expect(err).NotTo(HaveOccurred())
			sv.Shutdown()

			controllerReconciler := &StreamReconciler{
				controller,
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(err).To(HaveOccurred()) // Will be re-queued with back-off

			// Fetch resource
			err = k8sClient.Get(ctx, typeNamespacedName, stream)
			Expect(err).NotTo(HaveOccurred())

			By("Checking if the status was updated")
			Expect(stream.Status.Conditions).To(HaveLen(1))

			cond := stream.Status.Conditions[0]
			Expect(cond.Type).To(Equal(readyCondType))
			Expect(cond.Status).To(Equal(v1.ConditionFalse))
			Expect(cond.Reason).To(Equal("Errored"))
			Expect(cond.Message).To(HavePrefix("create or update stream:"))
			Expect(cond.LastTransitionTime).NotTo(BeEmpty())

			By("Checking if the observed generation does not match")
			Expect(stream.Status.ObservedGeneration).ToNot(Equal(stream.Generation))

			By("Checking if the finalizer was set")
			Expect(stream.Finalizers).To(ContainElement(streamFinalizer))
		})

		It("should delete stream marked for deletion", func(ctx SpecContext) {

			By("Reconciling the created resource once to ensure the finalizer is set and stream created")
			controllerReconciler := &StreamReconciler{
				baseController,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Stream exists
			_, err = jsClient.Stream(ctx, streamName)
			Expect(err).NotTo(HaveOccurred())

			By("Marking the resource as deleted")
			Expect(k8sClient.Delete(ctx, stream)).To(Succeed())

			By("Reconciling the deleted resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking if the resource was removed")
			Eventually(func() error {
				found := &api.Stream{}
				return k8sClient.Get(ctx, typeNamespacedName, found)
			}, 5*time.Second, time.Second).ShouldNot(Succeed())

			By("Checking if the stream was deleted")
			_, err = jsClient.Stream(ctx, streamName)
			Expect(err).To(MatchError(jetstream.ErrStreamNotFound))
		})

		It("should succeed deleting stream resource where the underlying stream was already deleted", func(ctx SpecContext) {

			By("Reconciling the created resource once to ensure the finalizer is set and stream created")
			controllerReconciler := &StreamReconciler{
				baseController,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Deleting the managed stream")
			err = jsClient.DeleteStream(ctx, streamName)
			Expect(err).NotTo(HaveOccurred())

			By("Marking the resource as deleted")
			Expect(k8sClient.Delete(ctx, stream)).To(Succeed())

			By("Reconciling the deleted resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking if the resource was removed")
			Eventually(func() error {
				found := &api.Stream{}
				return k8sClient.Get(ctx, typeNamespacedName, found)
			}, 5*time.Second, time.Second).ShouldNot(Succeed())
		})

		When("PreventDelete is set", func() {
			It("should not delete stream marked for deletion", func(ctx SpecContext) {

				By("Setting preventDelete on the resource")
				stream.Spec.PreventDelete = true
				Expect(k8sClient.Update(ctx, stream)).To(Succeed())

				By("Reconciling the updated resource once")
				controllerReconciler := &StreamReconciler{
					baseController,
				}

				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				// Stream exists
				_, err = jsClient.Stream(ctx, streamName)
				Expect(err).NotTo(HaveOccurred())

				By("Marking the resource as deleted")
				Expect(k8sClient.Delete(ctx, stream)).To(Succeed())

				By("Reconciling the deleted resource")
				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Checking if the resource was removed")
				Eventually(func() error {
					found := &api.Stream{}
					return k8sClient.Get(ctx, typeNamespacedName, found)
				}, 5*time.Second, time.Second).ShouldNot(Succeed())

				By("Checking if the stream was *not* deleted")
				_, err = jsClient.Stream(ctx, streamName)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("PreventUpdate is set", func() {
			BeforeEach(func(ctx SpecContext) {
				By("Setting prevent update")

				stream.Spec.PreventUpdate = true
				Expect(k8sClient.Update(ctx, stream)).To(Succeed())
			})

			It("should not create stream", func(ctx SpecContext) {
				By("Reconciling the updated resource")
				controllerReconciler := &StreamReconciler{
					baseController,
				}
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Checking that the stream was not created")
				_, err = jsClient.Stream(ctx, streamName)
				Expect(err).To(MatchError(jetstream.ErrStreamNotFound))

				By("Checking that the streams status was not updated")
				err = k8sClient.Get(ctx, typeNamespacedName, stream)
				Expect(err).NotTo(HaveOccurred())
				Expect(stream.Status.ObservedGeneration).To(BeNumerically("<", stream.Generation))
			})

			It("should not update stream", func(ctx SpecContext) {
				By("Creating the stream with empty subjects and description")
				_, err := jsClient.CreateStream(ctx, emptyStreamConfig)
				Expect(err).NotTo(HaveOccurred())

				By("Reconciling the resource")
				controllerReconciler := &StreamReconciler{
					baseController,
				}
				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Checking that the stream was not updated")
				s, err := jsClient.Stream(ctx, streamName)
				Expect(err).NotTo(HaveOccurred())
				Expect(s.CachedInfo().Config.Description).To(BeEmpty())

				By("Checking that the streams generation was not updated")
				err = k8sClient.Get(ctx, typeNamespacedName, stream)
				Expect(err).NotTo(HaveOccurred())
				Expect(stream.Status.ObservedGeneration).To(BeNumerically("<", stream.Generation))
			})
		})

		DescribeTableSubtree("controller restricted to not perform any updates on resource",
			func(getController func(SpecContext) StreamReconciler) {
				var controller StreamReconciler

				BeforeEach(func(ctx SpecContext) {
					controller = getController(ctx)
				})

				It("should not create stream", func(ctx SpecContext) {
					By("reconciling the resource")
					_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())

					By("Checking that the stream was not created")
					_, err = jsClient.Stream(ctx, streamName)
					Expect(err).To(MatchError(jetstream.ErrStreamNotFound))
				})
				It("should not update stream", func(ctx SpecContext) {
					By("creating the stream")
					_, err := jsClient.CreateStream(ctx, emptyStreamConfig)
					Expect(err).NotTo(HaveOccurred())

					By("reconciling the resource")
					_, err = controller.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())

					By("Checking that the stream was not updated")
					s, err := jsClient.Stream(ctx, streamName)
					Expect(err).NotTo(HaveOccurred())
					Expect(s.CachedInfo().Config.Description).To(BeEmpty())

					By("Checking that the streams generation was not updated")
					err = k8sClient.Get(ctx, typeNamespacedName, stream)
					Expect(err).NotTo(HaveOccurred())
					Expect(stream.Status.ObservedGeneration).To(BeNumerically("<", stream.Generation))
				})
				It("should not delete stream", func(ctx SpecContext) {

					By("creating the stream")
					_, err := jsClient.CreateStream(ctx, emptyStreamConfig)
					Expect(err).NotTo(HaveOccurred())

					By("Marking the resource as deleted")
					Expect(k8sClient.Delete(ctx, stream)).To(Succeed())

					By("reconciling the resource")
					_, err = controller.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
					Expect(err).NotTo(HaveOccurred())

					By("Checking if the resource was removed")
					Eventually(func() error {
						found := &api.Stream{}
						return k8sClient.Get(ctx, typeNamespacedName, found)
					}, 5*time.Second, time.Second).ShouldNot(Succeed())

					By("Checking if the stream was *not* deleted")
					_, err = jsClient.Stream(ctx, streamName)
					Expect(err).NotTo(HaveOccurred())
				})

			},
			// Test cases provide functions for providing a configured the controller and setup preconditions
			Entry("read only", func(ctx SpecContext) StreamReconciler {
				readOnly, err := NewJSController(k8sClient, &NatsConfig{ServerURL: testServer.ClientURL()}, &Config{ReadOnly: true})
				Expect(err).NotTo(HaveOccurred())
				return StreamReconciler{readOnly}
			}),
			Entry("namespaced", func(ctx SpecContext) StreamReconciler {
				readOnly, err := NewJSController(k8sClient, &NatsConfig{ServerURL: testServer.ClientURL()}, &Config{Namespace: "some-namespace"})
				Expect(err).NotTo(HaveOccurred())
				return StreamReconciler{readOnly}
			}),
		)

		It("should update stream on different server as specified in spec", func(ctx SpecContext) {
			By("Setting up the alternative server")
			// Setup altClient for alternate server
			altServer := CreateTestServer()
			defer altServer.Shutdown()

			By("Setting the server in the stream spec")
			stream.Spec.Servers = []string{altServer.ClientURL()}
			Expect(k8sClient.Update(ctx, stream)).To(Succeed())

			By("Checking precondition, that the stream does not yet exist")
			got, err := jsClient.Stream(ctx, streamName)
			Expect(err).To(MatchError(jetstream.ErrStreamNotFound))

			By("Reconciling the resource")
			controllerReconciler := &StreamReconciler{
				baseController,
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(err).NotTo(HaveOccurred())

			By("Checking if the stream was created on the alternative server")
			altClient, closer, err := CreateJetStreamClient(&NatsConfig{ServerURL: altServer.ClientURL()}, true)
			defer closer.Close()
			Expect(err).NotTo(HaveOccurred())

			got, err = altClient.Stream(ctx, streamName)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.CachedInfo().Created).To(BeTemporally("~", time.Now(), time.Second))

			By("Checking that the stream was NOT created on the original server")
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
			//if !reflect.DeepEqual(got, tt.want) {
			//	t.Errorf("mapSpecToConfig() \ngot  = %v\nwant = %v", got, tt.want)
			//}
		})
	}
}
