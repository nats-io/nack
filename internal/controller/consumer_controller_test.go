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
	"context"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
)

var _ = Describe("Consumer Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		consumer := &api.Consumer{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Consumer")
			err := k8sClient.Get(ctx, typeNamespacedName, consumer)
			if err != nil && errors.IsNotFound(err) {
				resource := &api.Consumer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: api.ConsumerSpec{
						AckPolicy:     "none",
						DeliverPolicy: "new",
						DurableName:   "test-consumer",
						ReplayPolicy:  "instant",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &api.Consumer{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Consumer")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ConsumerReconciler{
				baseController,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})

func Test_consumerSpecToConfig(t *testing.T) {

	date := time.Date(2024, 12, 03, 16, 55, 5, 0, time.UTC)
	dateString := date.Format(time.RFC3339)

	tests := []struct {
		name    string
		spec    *api.ConsumerSpec
		want    *jetstream.ConsumerConfig
		wantErr bool
	}{
		{
			name:    "empty spec",
			spec:    &api.ConsumerSpec{},
			want:    &jetstream.ConsumerConfig{},
			wantErr: false,
		},
		{
			name: "full spec",
			spec: &api.ConsumerSpec{
				AckPolicy:          "explicit",
				AckWait:            "10ns",
				BackOff:            []string{"1s", "5m"},
				Creds:              "",
				DeliverGroup:       "",
				DeliverPolicy:      "new",
				DeliverSubject:     "",
				Description:        "test consumer",
				PreventDelete:      false,
				PreventUpdate:      false,
				DurableName:        "test-consumer",
				FilterSubject:      "time.us.>",
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
				Nkey:               "",
				OptStartSeq:        17,
				OptStartTime:       dateString,
				RateLimitBps:       512,
				ReplayPolicy:       "instant",
				Replicas:           9,
				SampleFreq:         "25%",
				Servers:            nil,
				StreamName:         "",
				TLS:                api.TLS{},
				Account:            "",
				Metadata: map[string]string{
					"meta": "data",
				},
			},
			want: &jetstream.ConsumerConfig{
				Name:               "", // Optional, not mapped
				Durable:            "test-consumer",
				Description:        "test consumer",
				DeliverPolicy:      jetstream.DeliverNewPolicy,
				OptStartSeq:        17,
				OptStartTime:       &date,
				AckPolicy:          jetstream.AckExplicitPolicy,
				AckWait:            10 * time.Nanosecond,
				MaxDeliver:         3,
				BackOff:            []time.Duration{time.Second, 5 * time.Minute},
				FilterSubject:      "time.us.>",
				ReplayPolicy:       jetstream.ReplayInstantPolicy,
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := consumerSpecToConfig(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("consumerSpecToConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.EqualValues(t, tt.want, got, "consumerSpecToConfig(%v)", tt.spec)
		})
	}
}
