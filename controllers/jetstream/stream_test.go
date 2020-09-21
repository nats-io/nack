package jetstream

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1"
	clientsetfake "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/fake"
	informers "github.com/nats-io/nack/pkg/jetstream/generated/informers/externalversions"

	k8sapis "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	k8sclientsetfake "k8s.io/client-go/kubernetes/fake"
	k8stypedfake "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

func TestMain(m *testing.M) {
	// Disable error logs.
	utilruntime.ErrorHandlers = []func(error){
		func(err error) {},
	}

	os.Exit(m.Run())
}

func TestShouldEnqueueStream(t *testing.T) {
	t.Parallel()

	ts := k8smeta.NewTime(time.Now())

	cases := []struct {
		name string
		prev *apis.Stream
		next *apis.Stream

		want bool
	}{
		{
			name: "deletion time changed",
			prev: &apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{},
			},
			next: &apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{
					DeletionTimestamp: &ts,
				},
			},
			want: true,
		},
		{
			name: "spec changed",
			prev: &apis.Stream{
				Spec: apis.StreamSpec{
					MaxAge: "1h",
				},
			},
			next: &apis.Stream{
				Spec: apis.StreamSpec{
					MaxAge: "2h",
				},
			},
			want: true,
		},
		{
			name: "no change",
			prev: &apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{},
				Spec: apis.StreamSpec{
					MaxAge: "1h",
				},
			},
			next: &apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{},
				Spec: apis.StreamSpec{
					MaxAge: "1h",
				},
			},
			want: false,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := shouldEnqueueStream(c.prev, c.next)
			if got != c.want {
				t.Fatalf("got=%t; want=%t", got, c.want)
			}
		})
	}
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

	updateStream := func(a k8stesting.Action) (handled bool, o runtime.Object, err error) {
		ua, ok := a.(k8stesting.UpdateAction)
		if !ok {
			return false, nil, nil
		}

		return true, ua.GetObject(), nil
	}

	t.Run("delete stream", func(t *testing.T) {
		jc := clientsetfake.NewSimpleClientset()
		informerFactory := informers.NewSharedInformerFactory(jc, 0)
		informer := informerFactory.Jetstream().V1().Streams()

		ts := k8smeta.Unix(1600216923, 0)
		name := "my-stream"

		err := informer.Informer().GetStore().Add(
			&apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace:         "default",
					Name:              name,
					DeletionTimestamp: &ts,
					Finalizers:        []string{streamFinalizerKey},
				},
				Spec: apis.StreamSpec{
					Name: name,
				},
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "streams", updateStream)

		wantEvents := 3
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

		if err := ctrl.processStream("default", name); err != nil {
			t.Fatal(err)
		}

		if got := len(frec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		<-frec.Events
		<-frec.Events
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

		err := informer.Informer().GetStore().Add(
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
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "streams", updateStream)

		wantEvents := 4
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

		if err := ctrl.processStream("default", name); err != nil {
			t.Fatal(err)
		}

		if got := len(frec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		<-frec.Events
		<-frec.Events
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

		err := informer.Informer().GetStore().Add(
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
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "streams", updateStream)

		wantEvents := 4
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

		if err := ctrl.processStream("default", name); err != nil {
			t.Fatal(err)
		}

		if got := len(frec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		<-frec.Events
		<-frec.Events
		for i := 0; i < len(frec.Events); i++ {
			gotEvent := <-frec.Events
			if !strings.Contains(gotEvent, "Creat") {
				t.Error("unexpected event")
				t.Fatalf("got=%s; want=%s", gotEvent, "Creating/Created...")
			}
		}
	})

	t.Run("create stream with creds secret", func(t *testing.T) {
		kc := k8sclientsetfake.NewSimpleClientset()
		jc := clientsetfake.NewSimpleClientset()
		informerFactory := informers.NewSharedInformerFactory(jc, 0)
		informer := informerFactory.Jetstream().V1().Streams()

		secretName, secretKey := "mysecret", "nats-creds"

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

		kc.CoreV1().(*k8stypedfake.FakeCoreV1).PrependReactor("get", "secrets", getSecret)

		ns, name := "default", "my-stream"

		err := informer.Informer().GetStore().Add(
			&apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace:  "default",
					Name:       name,
					Generation: 1,
				},
				Spec: apis.StreamSpec{
					Name: name,
					CredentialsSecret: apis.StreamCredentialsSecret{
						Name: secretName,
						Key:  "nats-creds",
					},
				},
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "streams", updateStream)

		wantEvents := 4
		frec := record.NewFakeRecorder(wantEvents)
		ctrl := &Controller{
			ctx:          context.Background(),
			streamLister: informer.Lister(),
			ji:           jc.JetstreamV1(),
			ki:           kc.CoreV1(),
			rec:          frec,

			sc: &mockStreamClient{
				existsOK: false,
			},
		}

		if err := ctrl.processStream(ns, name); err != nil {
			t.Fatal(err)
		}

		if got := len(frec.Events); got != wantEvents {
			t.Error("unexpected number of events")
			t.Fatalf("got=%d; want=%d", got, wantEvents)
		}

		<-frec.Events
		<-frec.Events
		for i := 0; i < len(frec.Events); i++ {
			gotEvent := <-frec.Events
			if !strings.Contains(gotEvent, "Creat") {
				t.Error("unexpected event")
				t.Fatalf("got=%s; want=%s", gotEvent, "Creating/Created...")
			}
		}
	})
}

func TestProcessNextQueueItem(t *testing.T) {
	t.Parallel()

	t.Run("bad item key", func(t *testing.T) {
		t.Parallel()

		limiter := workqueue.DefaultControllerRateLimiter()
		q := workqueue.NewNamedRateLimitingQueue(limiter, "StreamsTest")
		defer q.ShutDown()

		ctrl := &Controller{
			streamQueue: q,
		}

		key := "this/is/a/bad/key"
		q.Add(key)

		ctrl.processNextQueueItem()

		if got, want := q.Len(), 0; got != want {
			t.Error("unexpected number of items in queue")
			t.Fatalf("got=%d; want=%d", got, want)
		}

		if got, want := q.NumRequeues(key), 0; got != want {
			t.Error("unexpected number of requeues")
			t.Fatalf("got=%d; want=%d", got, want)
		}
	})

	t.Run("process error", func(t *testing.T) {
		t.Parallel()

		limiter := workqueue.DefaultControllerRateLimiter()
		q := workqueue.NewNamedRateLimitingQueue(limiter, "StreamsTest")
		defer q.ShutDown()

		jc := clientsetfake.NewSimpleClientset()
		informerFactory := informers.NewSharedInformerFactory(jc, 0)
		informer := informerFactory.Jetstream().V1().Streams()

		ns, name := "default", "mystream"

		err := informer.Informer().GetStore().Add(
			&apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace:  ns,
					Name:       name,
					Generation: 1,
				},
				Spec: apis.StreamSpec{
					Name: name,
				},
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		ctrl := &Controller{
			ctx:          context.Background(),
			streamQueue:  q,
			streamLister: informer.Lister(),
			ji:           jc.JetstreamV1(),
			sc: &mockStreamClient{
				connectErr: fmt.Errorf("bad connect"),
			},
		}

		key := fmt.Sprintf("%s/%s", ns, name)
		q.Add(key)

		maxGets := maxQueueRetries + 1
		numRequeues := -1
		for i := 0; i < maxGets; i++ {
			if i == maxGets-1 {
				numRequeues = q.NumRequeues(key)
			}

			ctrl.processNextQueueItem()
		}

		if got, want := q.Len(), 0; got != want {
			t.Error("unexpected number of items in queue")
			t.Fatalf("got=%d; want=%d", got, want)
		}

		if got, want := numRequeues, 10; got != want {
			t.Error("unexpected number of requeues")
			t.Fatalf("got=%d; want=%d", got, want)
		}
	})

	t.Run("process ok", func(t *testing.T) {
		t.Parallel()

		limiter := workqueue.DefaultControllerRateLimiter()
		q := workqueue.NewNamedRateLimitingQueue(limiter, "StreamsTest")
		defer q.ShutDown()

		jc := clientsetfake.NewSimpleClientset()
		informerFactory := informers.NewSharedInformerFactory(jc, 0)
		informer := informerFactory.Jetstream().V1().Streams()

		ns, name := "default", "mystream"

		ctrl := &Controller{
			ctx:          context.Background(),
			streamQueue:  q,
			streamLister: informer.Lister(),
			ji:           jc.JetstreamV1(),
			sc: &mockStreamClient{
				connectErr: fmt.Errorf("bad connect"),
			},
		}

		key := fmt.Sprintf("%s/%s", ns, name)
		q.Add(key)

		numRequeues := q.NumRequeues(key)
		ctrl.processNextQueueItem()

		if got, want := q.Len(), 0; got != want {
			t.Error("unexpected number of items in queue")
			t.Fatalf("got=%d; want=%d", got, want)
		}

		if got, want := numRequeues, 0; got != want {
			t.Error("unexpected number of requeues")
			t.Fatalf("got=%d; want=%d", got, want)
		}
	})
}

func TestUpsertStreamCondition(t *testing.T) {
	var cs []apis.StreamCondition

	cs = upsertStreamCondition(cs, apis.StreamCondition{
		Type:               streamReadyCondType,
		Status:             k8sapis.ConditionTrue,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Synced",
		Message:            "Stream is synced with spec",
	})
	if got, want := len(cs), 1; got != want {
		t.Error("unexpected len conditions")
		t.Fatalf("got=%d; want=%d", got, want)
	}
	if got, want := cs[0].Reason, "Synced"; got != want {
		t.Error("unexpected reason")
		t.Fatalf("got=%s; want=%s", got, want)
	}

	cs = upsertStreamCondition(cs, apis.StreamCondition{
		Type:               streamReadyCondType,
		Status:             k8sapis.ConditionFalse,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Errored",
		Message:            "invalid foo",
	})
	if got, want := len(cs), 1; got != want {
		t.Error("unexpected len conditions")
		t.Fatalf("got=%d; want=%d", got, want)
	}
	if got, want := cs[0].Reason, "Errored"; got != want {
		t.Error("unexpected reason")
		t.Fatalf("got=%s; want=%s", got, want)
	}

	cs = upsertStreamCondition(cs, apis.StreamCondition{
		Type:               "Foo",
		Status:             k8sapis.ConditionTrue,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Bar",
		Message:            "bar ok",
	})
	if got, want := len(cs), 2; got != want {
		t.Error("unexpected len conditions")
		t.Fatalf("got=%d; want=%d", got, want)
	}
	if got, want := cs[1].Reason, "Bar"; got != want {
		t.Error("unexpected reason")
		t.Fatalf("got=%s; want=%s", got, want)
	}
}
