package jetstream

import (
	"context"
	"errors"
	"strings"
	"testing"

	jsmapi "github.com/nats-io/jsm.go/api"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta1"
	clientsetfake "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/fake"

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

		informer := ctrl.informerFactory.Jetstream().V1beta1().Consumers()
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

		jc.PrependReactor("update", "consumers", updateObject)

		notFoundErr := jsmapi.ApiError{Code: 404}
		jsmc := &mockJsmClient{
			loadConsumerErr: notFoundErr,
			newConsumerErr:  nil,
			newConsumer:     &mockDeleter{},
		}
		if err := ctrl.processConsumer(ns, name, jsmc); err != nil {
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

	t.Run("update consumer", func(t *testing.T) {
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

		informer := ctrl.informerFactory.Jetstream().V1beta1().Consumers()
		err := informer.Informer().GetStore().Add(&apis.Consumer{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Finalizers: []string{consumerFinalizerKey},
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
			loadConsumer:    &mockDeleter{},
		}
		if err := ctrl.processConsumer(ns, name, jsmc); err != nil {
			t.Fatal(err)
		}

		if got := len(rec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		gotEvent := <-rec.Events
		if !strings.Contains(gotEvent, "Updating") {
			t.Error("unexpected event")
			t.Fatalf("got=%s; want=%s", gotEvent, "Updating...")
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

		informer := ctrl.informerFactory.Jetstream().V1beta1().Consumers()
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

		jc.PrependReactor("update", "consumers", updateObject)

		jsmc := &mockJsmClient{
			loadConsumerErr: nil,
			loadConsumer:    &mockDeleter{},
		}
		if err := ctrl.processConsumer(ns, name, jsmc); err != nil {
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

		informer := ctrl.informerFactory.Jetstream().V1beta1().Consumers()
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
		if err := ctrl.processConsumer(ns, name, jsmc); err == nil {
			t.Fatal("unexpected success")
		}
	})
}
