package jetstream

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1"
	clientsetfake "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/fake"
	informers "github.com/nats-io/nack/pkg/jetstream/generated/informers/externalversions"

	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

func TestMain(m *testing.M) {
	testMode = true
	os.Exit(m.Run())
}

func TestValidateStreamUpdate(t *testing.T) {
	t.Parallel()

	t.Run("no spec changes, no update", func(t *testing.T) {
		s := &apis.Stream{
			Spec: apis.StreamSpec{
				Name:   "foo",
				MaxAge: "1h",
			},
		}

		if err := validateStreamUpdate(s, s); !errors.Is(err, errNothingToUpdate) {
			t.Fatalf("got=%v; want=%v", err, errNothingToUpdate)
		}
	})

	t.Run("spec changed, update ok", func(t *testing.T) {
		prev := &apis.Stream{
			Spec: apis.StreamSpec{
				Name:   "foo",
				MaxAge: "1h",
			},
		}
		next := &apis.Stream{
			Spec: apis.StreamSpec{
				Name:   "foo",
				MaxAge: "10h",
			},
		}

		if err := validateStreamUpdate(prev, next); err != nil {
			t.Fatalf("got=%v; want=nil", err)
		}
	})

	t.Run("stream name changed, update bad", func(t *testing.T) {
		prev := &apis.Stream{
			Spec: apis.StreamSpec{
				Name:   "foo",
				MaxAge: "1h",
			},
		}
		next := &apis.Stream{
			Spec: apis.StreamSpec{
				Name:   "bar",
				MaxAge: "1h",
			},
		}

		if err := validateStreamUpdate(prev, next); err == nil {
			t.Fatal("got=nil; want=err")
		}
	})
}

func TestEnqueueStreamWork(t *testing.T) {
	t.Parallel()

	limiter := workqueue.DefaultControllerRateLimiter()
	q := workqueue.NewNamedRateLimitingQueue(limiter, "StreamsTest")
	defer q.ShutDown()

	s := &apis.Stream{
		ObjectMeta: k8smeta.ObjectMeta{
			Namespace: "default",
			Name:      "my-stream",
		},
	}

	if err := enqueueStreamWork(q, s); err != nil {
		t.Fatal(err)
	}

	if got, want := q.Len(), 1; got != want {
		t.Error("unexpected queue length")
		t.Fatalf("got=%d; want=%d", got, want)
	}

	wantItem := fmt.Sprintf("%s/%s", s.Namespace, s.Name)
	gotItem, _ := q.Get()
	if gotItem != wantItem {
		t.Error("unexpected queue item")
		t.Fatalf("got=%s; want=%s", gotItem, wantItem)
	}
}

func TestProcessStream(t *testing.T) {
	t.Parallel()

	t.Run("delete stream", func(t *testing.T) {
		jc := clientsetfake.NewSimpleClientset()
		informerFactory := informers.NewSharedInformerFactory(jc, 0)
		informer := informerFactory.Jetstream().V1().Streams()

		ts := k8smeta.Unix(1600216923, 0)
		name := "my-stream"

		informer.Informer().GetStore().Add(
			&apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace:         "default",
					Name:              name,
					DeletionTimestamp: &ts,
					Finalizers: []string{streamFinalizerKey},
				},
				Spec: apis.StreamSpec{
					Name: name,
				},
			},
		)

		jc.PrependReactor("update", "streams", func(a k8stesting.Action) (handled bool, o runtime.Object, err error) {
			ua, ok := a.(k8stesting.UpdateAction)
			if !ok {
				return false, nil, nil
			}

			return true, ua.GetObject(), nil
		})

		wantEvents := 1
		frec := record.NewFakeRecorder(wantEvents)
		ctrl := &Controller{
			ctx:          context.Background(),
			streamLister: informer.Lister(),
			ji:           jc.JetstreamV1(),
			rec:          frec,

			sc: &mockStreamClient{
				existsOK: true,
			},
		}

		err := ctrl.processStream("default", name)
		if err != nil {
			t.Fatal(err)
		}

		if got := len(frec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		gotEvent := <-frec.Events
		if !strings.Contains(gotEvent, "Deleting") {
			t.Error("unexpected event")
			t.Fatalf("got=%s; want=%s", gotEvent, "Deleting...")
		}
	})

	t.Run("update stream", func(t *testing.T) {
		jc := clientsetfake.NewSimpleClientset()
		informerFactory := informers.NewSharedInformerFactory(jc, 0)
		informer := informerFactory.Jetstream().V1().Streams()

		name := "my-stream"

		informer.Informer().GetStore().Add(
			&apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace:  "default",
					Name:       name,
					Generation: 2,
				},
				Spec: apis.StreamSpec{
					Name: name,
				},
				Status: apis.StreamStatus{
					ObservedGeneration: 1,
				},
			},
		)

		jc.PrependReactor("update", "streams", func(a k8stesting.Action) (handled bool, o runtime.Object, err error) {
			ua, ok := a.(k8stesting.UpdateAction)
			if !ok {
				return false, nil, nil
			}

			return true, ua.GetObject(), nil
		})

		wantEvents := 2
		frec := record.NewFakeRecorder(wantEvents)
		ctrl := &Controller{
			ctx:          context.Background(),
			streamLister: informer.Lister(),
			ji:           jc.JetstreamV1(),
			rec:          frec,

			sc: &mockStreamClient{
				existsOK: true,
			},
		}

		err := ctrl.processStream("default", name)
		if err != nil {
			t.Fatal(err)
		}

		if got := len(frec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		for i := 0; i < len(frec.Events); i++ {
			gotEvent := <-frec.Events
			if !strings.Contains(gotEvent, "Updat") {
				t.Error("unexpected event")
				t.Fatalf("got=%s; want=%s", gotEvent, "Updating/Updated...")
			}
		}
	})

	t.Run("create stream", func(t *testing.T) {
		jc := clientsetfake.NewSimpleClientset()
		informerFactory := informers.NewSharedInformerFactory(jc, 0)
		informer := informerFactory.Jetstream().V1().Streams()

		name := "my-stream"

		informer.Informer().GetStore().Add(
			&apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace:  "default",
					Name:       name,
					Generation: 1,
				},
				Spec: apis.StreamSpec{
					Name: name,
				},
			},
		)

		jc.PrependReactor("update", "streams", func(a k8stesting.Action) (handled bool, o runtime.Object, err error) {
			ua, ok := a.(k8stesting.UpdateAction)
			if !ok {
				return false, nil, nil
			}

			return true, ua.GetObject(), nil
		})

		wantEvents := 2
		frec := record.NewFakeRecorder(wantEvents)
		ctrl := &Controller{
			ctx:          context.Background(),
			streamLister: informer.Lister(),
			ji:           jc.JetstreamV1(),
			rec:          frec,

			sc: &mockStreamClient{
				existsOK: false,
			},
		}

		err := ctrl.processStream("default", name)
		if err != nil {
			t.Fatal(err)
		}

		if got := len(frec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		for i := 0; i < len(frec.Events); i++ {
			gotEvent := <-frec.Events
			if !strings.Contains(gotEvent, "Creat") {
				t.Error("unexpected event")
				t.Fatalf("got=%s; want=%s", gotEvent, "Creating/Created...")
			}
		}
	})
}
