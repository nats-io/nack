package jetstream

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	jsmapi "github.com/nats-io/jsm.go/api"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	clientsetfake "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclientsetfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

func TestProcessConsumer(t *testing.T) {
	t.Parallel()

	updateObject := func(a k8stesting.Action) (handled bool, o runtime.Object, err error) {
		ua, ok := a.(k8stesting.UpdateAction)
		if !ok {
			return false, nil, nil
		}

		return true, ua.GetObject(), nil
	}

	t.Run("create consumer", func(t *testing.T) {
		t.Parallel()

		jc := clientsetfake.NewSimpleClientset()
		wantEvents := 2
		rec := record.NewFakeRecorder(wantEvents)
		ctrl := NewController(Options{
			Ctx:            context.Background(),
			KubeIface:      k8sclientsetfake.NewSimpleClientset(),
			JetstreamIface: jc,
			Recorder:       rec,
		})

		ns, name := "default", "my-consumer"

		informer := ctrl.informerFactory.Jetstream().V1beta2().Consumers()
		err := informer.Informer().GetStore().Add(&apis.Consumer{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Generation: 1,
			},
			Spec: apis.ConsumerSpec{
				DurableName:       name,
				DeliverPolicy:     "byStartTime",
				OptStartTime:      time.Now().Format(time.RFC3339),
				AckPolicy:         "explicit",
				AckWait:           "1m",
				ReplayPolicy:      "original",
				SampleFreq:        "50",
				HeartbeatInterval: "30s",
				BackOff:           []string{"500ms", "1s"},
				HeadersOnly:       true,
				MaxRequestExpires: "5m",
				MemStorage:        true,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "consumers", updateObject)

		notFoundErr := jsmapi.ApiError{Code: 404}
		jsmc := &mockJsmClient{
			loadConsumerErr: notFoundErr,
			newConsumerErr:  nil,
			newConsumer:     &mockConsumer{},
		}
		if err := ctrl.processConsumer(ns, name, func(n *natsContext) (jsmClient, error) {
			return jsmc, nil
		}); err != nil {
			t.Fatal(err)
		}

		if got := len(rec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		for i := 0; i < len(rec.Events); i++ {
			gotEvent := <-rec.Events
			if !strings.Contains(gotEvent, "Creat") {
				t.Error("unexpected event")
				t.Fatalf("got=%s; want=%s", gotEvent, "Creating/Created...")
			}
		}
	})

	t.Run("create consumer, invalid configuration", func(t *testing.T) {
		t.Parallel()

		jc := clientsetfake.NewSimpleClientset()
		wantEvents := 1
		rec := record.NewFakeRecorder(wantEvents)
		ctrl := NewController(Options{
			Ctx:            context.Background(),
			KubeIface:      k8sclientsetfake.NewSimpleClientset(),
			JetstreamIface: jc,
			Recorder:       rec,
		})

		ns, name := "default", "my-consumer"

		informer := ctrl.informerFactory.Jetstream().V1beta2().Consumers()
		err := informer.Informer().GetStore().Add(&apis.Consumer{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Generation: 1,
			},
			Spec: apis.ConsumerSpec{
				DurableName:   name,
				DeliverPolicy: "invalid",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "consumers", updateObject)

		notFoundErr := jsmapi.ApiError{Code: 404}
		jsmc := &mockJsmClient{
			loadConsumerErr: notFoundErr,
			newConsumerErr:  nil,
			newConsumer:     &mockConsumer{},
		}
		if err := ctrl.processConsumer(ns, name, testWrapJSMC(jsmc)); err == nil || !strings.Contains(err.Error(), `failed to create consumer "my-consumer" on stream `) {
			t.Fatal(err)
		}

		if got := len(rec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		gotEvent := <-rec.Events
		if !strings.Contains(gotEvent, "Creating") {
			t.Error("unexpected event")
			t.Fatalf("got=%s; want=%s", gotEvent, "Creating...")
		}
	})

	t.Run("update consumer", func(t *testing.T) {
		t.Parallel()

		jc := clientsetfake.NewSimpleClientset()
		wantEvents := 2
		rec := record.NewFakeRecorder(wantEvents)
		ctrl := NewController(Options{
			Ctx:            context.Background(),
			KubeIface:      k8sclientsetfake.NewSimpleClientset(),
			JetstreamIface: jc,
			Recorder:       rec,
		})

		ns, name := "default", "my-consumer"

		informer := ctrl.informerFactory.Jetstream().V1beta2().Consumers()
		err := informer.Informer().GetStore().Add(&apis.Consumer{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Generation: 2,
			},
			Spec: apis.ConsumerSpec{
				DurableName: name,
			},
			Status: apis.Status{
				ObservedGeneration: 1,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "consumers", updateObject)

		jsmc := &mockJsmClient{
			loadConsumerErr: nil,
			loadConsumer:    &mockConsumer{},
		}
		if err := ctrl.processConsumer(ns, name, testWrapJSMC(jsmc)); err != nil {
			t.Fatal(err)
		}

		if got := len(rec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		for i := 0; i < len(rec.Events); i++ {
			gotEvent := <-rec.Events
			if !strings.Contains(gotEvent, "Updat") {
				t.Error("unexpected event")
				t.Fatalf("got=%s; want=%s", gotEvent, "Updating/Updated...")
			}
		}
	})

	t.Run("delete consumer", func(t *testing.T) {
		t.Parallel()

		jc := clientsetfake.NewSimpleClientset()
		wantEvents := 1
		rec := record.NewFakeRecorder(wantEvents)
		ctrl := NewController(Options{
			Ctx:            context.Background(),
			KubeIface:      k8sclientsetfake.NewSimpleClientset(),
			JetstreamIface: jc,
			Recorder:       rec,
		})

		ts := k8smeta.Unix(1600216923, 0)
		ns, name := "default", "my-consumer"

		informer := ctrl.informerFactory.Jetstream().V1beta2().Consumers()
		err := informer.Informer().GetStore().Add(&apis.Consumer{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:         ns,
				Name:              name,
				DeletionTimestamp: &ts,
			},
			Spec: apis.ConsumerSpec{
				DurableName: name,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "consumers", updateObject)

		jsmc := &mockJsmClient{
			loadConsumerErr: nil,
			loadConsumer:    &mockConsumer{},
		}
		if err := ctrl.processConsumer(ns, name, testWrapJSMC(jsmc)); err != nil {
			t.Fatal(err)
		}

		if got := len(rec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		gotEvent := <-rec.Events
		if !strings.Contains(gotEvent, "Deleting") {
			t.Error("unexpected event")
			t.Fatalf("got=%s; want=%s", gotEvent, "Deleting...")
		}
	})

	t.Run("process error", func(t *testing.T) {
		t.Parallel()

		jc := clientsetfake.NewSimpleClientset()
		wantEvents := 1
		rec := record.NewFakeRecorder(wantEvents)
		ctrl := NewController(Options{
			Ctx:            context.Background(),
			KubeIface:      k8sclientsetfake.NewSimpleClientset(),
			JetstreamIface: jc,
			Recorder:       rec,
		})

		ns, name := "default", "my-consumer"

		informer := ctrl.informerFactory.Jetstream().V1beta2().Consumers()
		err := informer.Informer().GetStore().Add(&apis.Consumer{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Generation: 1,
			},
			Spec: apis.ConsumerSpec{
				DurableName: name,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "consumers", func(a k8stesting.Action) (handled bool, o runtime.Object, err error) {
			ua, ok := a.(k8stesting.UpdateAction)
			if !ok {
				return false, nil, nil
			}
			obj := ua.GetObject()

			str, ok := obj.(*apis.Consumer)
			if !ok {
				t.Error("unexpected object type")
				t.Fatalf("got=%T; want=%T", obj, &apis.Consumer{})
			}

			if got, want := len(str.Status.Conditions), 1; got != want {
				t.Error("unexpected number of conditions")
				t.Fatalf("got=%d; want=%d", got, want)
			}
			if got, want := str.Status.Conditions[0].Reason, "Errored"; got != want {
				t.Error("unexpected condition reason")
				t.Fatalf("got=%s; want=%s", got, want)
			}

			return true, obj, nil
		})

		jsmc := &mockJsmClient{
			loadConsumerErr: errors.New("failed to load consumer"),
		}
		if err := ctrl.processConsumer(ns, name, testWrapJSMC(jsmc)); err == nil {
			t.Fatal("unexpected success")
		}
	})
}

func TestConsumerSpecToOpts(t *testing.T) {
	tests := map[string]struct {
		name     string
		given    apis.ConsumerSpec
		expected jsmapi.ConsumerConfig
		errCheck func(t *testing.T, err error)
	}{
		"valid consumer spec": {
			given: apis.ConsumerSpec{
				DurableName:       "my-consumer",
				DeliverPolicy:     "byStartSequence",
				OptStartSeq:       10,
				AckPolicy:         "explicit",
				AckWait:           "1m",
				ReplayPolicy:      "original",
				SampleFreq:        "50",
				HeartbeatInterval: "30s",
				BackOff:           []string{"500ms", "1s"},
				HeadersOnly:       true,
				MaxRequestExpires: "5m",
				MemStorage:        true,
			},
			expected: jsmapi.ConsumerConfig{
				AckPolicy:         jsmapi.AckExplicit,
				AckWait:           1 * time.Minute,
				DeliverPolicy:     jsmapi.DeliverByStartSequence,
				Durable:           "my-consumer",
				Heartbeat:         30 * time.Second,
				BackOff:           []time.Duration{500 * time.Millisecond, 1 * time.Second},
				OptStartSeq:       10,
				ReplayPolicy:      jsmapi.ReplayOriginal,
				SampleFrequency:   "50%",
				HeadersOnly:       true,
				MaxRequestExpires: 5 * time.Minute,
				MemoryStorage:     true,
			},
		},
		"valid consumer spec, defaults only": {
			given: apis.ConsumerSpec{
				DurableName: "my-consumer",
			},
			expected: jsmapi.ConsumerConfig{
				Durable: "my-consumer",
			},
		},
		"invalid deliver policy value": {
			given: apis.ConsumerSpec{
				DurableName:   "my-consumer",
				DeliverPolicy: "invalid",
			},
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid value for 'deliverPolicy': 'invalid'")
			},
		},
		"missing start time for deliver policy byStartTime": {
			given: apis.ConsumerSpec{
				DurableName:   "my-consumer",
				DeliverPolicy: "byStartTime",
			},
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "'optStartTime' is required for deliver policy 'byStartTime'")
			},
		},
		"deliver policy lastPerSubject": {
			given: apis.ConsumerSpec{
				DurableName:   "my-consumer",
				DeliverPolicy: "lastPerSubject",
			},
			expected: jsmapi.ConsumerConfig{
				Durable: "my-consumer",
				DeliverPolicy: jsmapi.DeliverLastPerSubject,
			},
		},
		"invalid ack policy": {
			given: apis.ConsumerSpec{
				DurableName: "my-consumer",
				AckPolicy:   "invalid",
			},
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid value for 'ackPolicy': 'invalid'")
			},
		},
		"invalid replay policy": {
			given: apis.ConsumerSpec{
				DurableName:  "my-consumer",
				ReplayPolicy: "invalid",
			},
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid value for 'replayPolicy': 'invalid'")
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			res, err := consumerSpecToOpts(test.given)
			if test.errCheck != nil {
				test.errCheck(t, err)
				return
			}
			require.NoError(t, err)
			var config jsmapi.ConsumerConfig
			for _, opt := range res {
				err := opt(&config)
				require.NoError(t, err)
			}
			assert.Equal(t, test.expected, config)
		})
	}
}

func testWrapJSMC(jsm jsmClient) jsmClientFunc {
	return func(n *natsContext) (jsmClient, error) {
		return jsm, nil
	}
}
