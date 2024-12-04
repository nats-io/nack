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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
)

var _ = Describe("Stream Controller", func() {
	When("reconciling a resource", func() {
		const resourceName = "test-stream"

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
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
						Name:        "test-stream",
						Replicas:    1,
						Subjects:    []string{"tests.*"},
						Description: "test stream",
						Retention:   "workqueue",
						Discard:     "old",
						Storage:     "file",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func(ctx SpecContext) {
			// Remove test stream resource
			resource := &api.Stream{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Stream")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			By("Deleting the stream")
			err = jsClient.DeleteStream(ctx, resource.Spec.Name)
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

			By("Checking if the finalizer was set")
			Expect(stream.Finalizers).To(ContainElement(streamFinalizer))

			By("Checking if the stream was created")
			natsStream, err := jsClient.Stream(ctx, "test-stream")
			Expect(err).NotTo(HaveOccurred())
			streamInfo, err := natsStream.Info(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(streamInfo.Config.Name).To(Equal("test-stream"))
			Expect(streamInfo.Created).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("should successfully update the stream and update the status", func(ctx SpecContext) {

			By("Creating the stream with empty subjects and description")
			_, err := jsClient.CreateStream(ctx, jetstream.StreamConfig{
				Name:      "test-stream",
				Replicas:  1,
				Retention: jetstream.WorkQueuePolicy,
				Discard:   jetstream.DiscardOld,
				Storage:   jetstream.FileStorage,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling the created resource")
			controllerReconciler := &StreamReconciler{
				baseController,
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

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

			By("Checking if the stream was updated")
			natsStream, err := jsClient.Stream(ctx, "test-stream")
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
			sv := CreateTestServer()
			// Is there an easier way to create a failing js client?
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
			Expect(cond.Message).To(Equal("create or update stream: context deadline exceeded"))
			Expect(cond.LastTransitionTime).NotTo(BeEmpty())

			By("Checking if the finalizer was set")
			Expect(stream.Finalizers).To(ContainElement(streamFinalizer))
		})

		PIt("should delete stream marked for deletion", func(ctx SpecContext) {

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
