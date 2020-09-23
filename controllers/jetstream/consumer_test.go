package jetstream

import (
	"context"
	"strings"
	"testing"
	"time"

	jsmapi "github.com/nats-io/jsm.go/api"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1"
	clientsetfake "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/fake"

	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclientsetfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	k8sapis "k8s.io/api/core/v1"
	k8stypedfake "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	"k8s.io/client-go/tools/record"
)

func TestProcessConsumer(t *testing.T) {
	t.Parallel()

	updateConsumer := func(a k8stesting.Action) (handled bool, o runtime.Object, err error) {
		ua, ok := a.(k8stesting.UpdateAction)
		if !ok {
			return false, nil, nil
		}

		return true, ua.GetObject(), nil
	}

	const secretName, secretKey = "mysecret", "nats-creds"
	getSecret := func(a k8stesting.Action) (handled bool, o runtime.Object, err error) {
		ga, ok := a.(k8stesting.GetAction)
		if !ok {
			return false, nil, nil
		}
		if ga.GetName() != secretName {
			return false, nil, nil
		}

		return true, &k8sapis.Secret{
			Data: map[string][]byte{
				secretKey: []byte("... creds..."),
			},
		}, nil
	}

	t.Run("create consumer", func(t *testing.T) {
		jc := clientsetfake.NewSimpleClientset()
		wantEvents := 4
		rec := record.NewFakeRecorder(wantEvents)
		ctrl := NewController(Options{
			Ctx:            context.Background(),
			KubeIface:      k8sclientsetfake.NewSimpleClientset(),
			JetstreamIface: jc,
			Recorder:       rec,
		})

		ns, name := "default", "my-consumer"

		informer := ctrl.informerFactory.Jetstream().V1().Consumers()
		err := informer.Informer().GetStore().Add(&apis.Consumer{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Finalizers: []string{consumerFinalizerKey},
				Generation: 1,
			},
			Spec: apis.ConsumerSpec{
				DurableName: name,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "consumers", updateConsumer)

		notFoundErr := jsmapi.ApiError{Code: 404}
		jsmc := &mockJsmClient{
			loadConsumerErr:     notFoundErr,
			newConsumerErr:      nil,
			newConsumerConsumer: &mockConsumer{},
		}
		if err := ctrl.processConsumer(ns, name, jsmc); err != nil {
			t.Fatal(err)
		}

		if got := len(rec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		<-rec.Events
		<-rec.Events
		for i := 0; i < len(rec.Events); i++ {
			gotEvent := <-rec.Events
			if !strings.Contains(gotEvent, "Creat") {
				t.Error("unexpected event")
				t.Fatalf("got=%s; want=%s", gotEvent, "Creating/Created...")
			}
		}
	})

	t.Run("create consumer with credentials", func(t *testing.T) {
		jc := clientsetfake.NewSimpleClientset()
		kc := k8sclientsetfake.NewSimpleClientset()
		wantEvents := 4
		rec := record.NewFakeRecorder(wantEvents)
		ctrl := NewController(Options{
			Ctx:            context.Background(),
			KubeIface:      kc,
			JetstreamIface: jc,
			Recorder:       rec,
		})

		ns, name := "default", "my-consumer"

		informer := ctrl.informerFactory.Jetstream().V1().Consumers()
		err := informer.Informer().GetStore().Add(&apis.Consumer{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Finalizers: []string{consumerFinalizerKey},
				Generation: 1,
			},
			Spec: apis.ConsumerSpec{
				DurableName: name,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "consumers", updateConsumer)
		kc.CoreV1().(*k8stypedfake.FakeCoreV1).PrependReactor("get", "secrets", getSecret)

		notFoundErr := jsmapi.ApiError{Code: 404}
		jsmc := &mockJsmClient{
			loadConsumerErr:     notFoundErr,
			newConsumerErr:      nil,
			newConsumerConsumer: &mockConsumer{},
		}
		if err := ctrl.processConsumer(ns, name, jsmc); err != nil {
			t.Fatal(err)
		}

		if got := len(rec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		<-rec.Events
		<-rec.Events
		for i := 0; i < len(rec.Events); i++ {
			gotEvent := <-rec.Events
			if !strings.Contains(gotEvent, "Creat") {
				t.Error("unexpected event")
				t.Fatalf("got=%s; want=%s", gotEvent, "Creating/Created...")
			}
		}
	})

	t.Run("update consumer", func(t *testing.T) {
		jc := clientsetfake.NewSimpleClientset()
		wantEvents := 3
		rec := record.NewFakeRecorder(wantEvents)
		ctrl := NewController(Options{
			Ctx:            context.Background(),
			KubeIface:      k8sclientsetfake.NewSimpleClientset(),
			JetstreamIface: jc,
			Recorder:       rec,
		})

		ns, name := "default", "my-consumer"

		informer := ctrl.informerFactory.Jetstream().V1().Consumers()
		err := informer.Informer().GetStore().Add(&apis.Consumer{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Finalizers: []string{consumerFinalizerKey},
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

		jc.PrependReactor("update", "consumers", updateConsumer)

		jsmc := &mockJsmClient{
			loadConsumerErr:      nil,
			loadConsumerConsumer: &mockConsumer{},
		}
		if err := ctrl.processConsumer(ns, name, jsmc); err != nil {
			t.Fatal(err)
		}

		if got := len(rec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		<-rec.Events
		<-rec.Events
		gotEvent := <-rec.Events
		if !strings.Contains(gotEvent, "Updating") {
			t.Error("unexpected event")
			t.Fatalf("got=%s; want=%s", gotEvent, "Updating...")
		}
	})

	t.Run("delete consumer", func(t *testing.T) {
		jc := clientsetfake.NewSimpleClientset()
		wantEvents := 3
		rec := record.NewFakeRecorder(wantEvents)
		ctrl := NewController(Options{
			Ctx:            context.Background(),
			KubeIface:      k8sclientsetfake.NewSimpleClientset(),
			JetstreamIface: jc,
			Recorder:       rec,
		})

		ts := k8smeta.Unix(1600216923, 0)
		ns, name := "default", "my-consumer"

		informer := ctrl.informerFactory.Jetstream().V1().Consumers()
		err := informer.Informer().GetStore().Add(&apis.Consumer{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:         ns,
				Name:              name,
				DeletionTimestamp: &ts,
				Finalizers:        []string{consumerFinalizerKey},
			},
			Spec: apis.ConsumerSpec{
				DurableName: name,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "consumers", updateConsumer)

		jsmc := &mockJsmClient{
			loadConsumerErr:      nil,
			loadConsumerConsumer: &mockConsumer{},
		}
		if err := ctrl.processConsumer(ns, name, jsmc); err != nil {
			t.Fatal(err)
		}

		if got := len(rec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		<-rec.Events
		<-rec.Events
		gotEvent := <-rec.Events
		if !strings.Contains(gotEvent, "Deleting") {
			t.Error("unexpected event")
			t.Fatalf("got=%s; want=%s", gotEvent, "Deleting...")
		}
	})
}

func TestShouldEnqueueConsumer(t *testing.T) {
	t.Parallel()

	ts := k8smeta.NewTime(time.Now())

	cases := []struct {
		name string
		prev *apis.Consumer
		next *apis.Consumer

		want bool
	}{
		{
			name: "deletion time changed",
			prev: &apis.Consumer{
				ObjectMeta: k8smeta.ObjectMeta{},
			},
			next: &apis.Consumer{
				ObjectMeta: k8smeta.ObjectMeta{
					DeletionTimestamp: &ts,
				},
			},
			want: true,
		},
		{
			name: "spec changed",
			prev: &apis.Consumer{
				Spec: apis.ConsumerSpec{
					StreamName: "foo",
				},
			},
			next: &apis.Consumer{
				Spec: apis.ConsumerSpec{
					StreamName: "bar",
				},
			},
			want: true,
		},
		{
			name: "no change",
			prev: &apis.Consumer{
				ObjectMeta: k8smeta.ObjectMeta{},
				Spec: apis.ConsumerSpec{
					StreamName: "foo",
				},
			},
			next: &apis.Consumer{
				ObjectMeta: k8smeta.ObjectMeta{},
				Spec: apis.ConsumerSpec{
					StreamName: "foo",
				},
			},
			want: false,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := shouldEnqueueConsumer(c.prev, c.next)
			if got != c.want {
				t.Fatalf("got=%t; want=%t", got, c.want)
			}
		})
	}
}
