package jetstream

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	jsmapi "github.com/nats-io/jsm.go/api"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta1"

	k8sapis "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/workqueue"
)

func TestMain(m *testing.M) {
	// Disable error logs.
	utilruntime.ErrorHandlers = []func(error){
		func(err error) {},
	}

	os.Exit(m.Run())
}

func TestGetStorageType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		storage string

		wantType jsmapi.StorageType
		wantErr  bool
	}{
		{storage: "memory", wantType: jsmapi.MemoryStorage},
		{storage: "file", wantType: jsmapi.FileStorage},
		{storage: "junk", wantErr: true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.storage, func(t *testing.T) {
			t.Parallel()

			got, err := getStorageType(c.storage)
			if err != nil && !c.wantErr {
				t.Error("unexpected error")
				t.Fatalf("got=%s; want=nil", err)
			} else if err == nil && c.wantErr {
				t.Error("unexpected success")
				t.Fatalf("got=nil; want=err")
			}

			if got != c.wantType {
				t.Error("unexpected storage type")
				t.Fatalf("got=%v; want=%v", got, c.wantType)
			}
		})
	}
}

func TestAddFinalizer(t *testing.T) {
	t.Parallel()

	fs := []string{"foo", "bar"}
	fs = addFinalizer(fs, "fizz")
	fs = addFinalizer(fs, "fizz")
	fs = addFinalizer(fs, "fizz")

	want := []string{"foo", "bar", "fizz"}
	if !reflect.DeepEqual(fs, want) {
		t.Error("unexpected finalizers")
		t.Fatalf("got=%v; want=%v", fs, want)
	}
}

func TestRemoveFinalizer(t *testing.T) {
	t.Parallel()

	fs := []string{"foo", "bar"}
	fs = removeFinalizer(fs, "bar")
	fs = removeFinalizer(fs, "bar")
	fs = removeFinalizer(fs, "bar")

	want := []string{"foo"}
	if !reflect.DeepEqual(fs, want) {
		t.Error("unexpected finalizers")
		t.Fatalf("got=%v; want=%v", fs, want)
	}
}

func TestEnqueueWork(t *testing.T) {
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

	if err := enqueueWork(q, s); err != nil {
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

func TestProcessQueueNext(t *testing.T) {
	t.Parallel()

	t.Run("bad item key", func(t *testing.T) {
		t.Parallel()

		limiter := workqueue.DefaultControllerRateLimiter()
		q := workqueue.NewNamedRateLimitingQueue(limiter, "StreamsTest")
		defer q.ShutDown()

		key := "this/is/a/bad/key"
		q.Add(key)

		processQueueNext(q, &mockJsmClient{}, func(ns, name string, c jsmClient) error {
			return nil
		})

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

		ns, name := "default", "mystream"
		key := fmt.Sprintf("%s/%s", ns, name)
		q.Add(key)

		maxGets := maxQueueRetries + 1
		numRequeues := -1
		for i := 0; i < maxGets; i++ {
			if i == maxGets-1 {
				numRequeues = q.NumRequeues(key)
			}

			processQueueNext(q, &mockJsmClient{}, func(ns, name string, c jsmClient) error {
				return fmt.Errorf("processing error")
			})
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

		ns, name := "default", "mystream"
		key := fmt.Sprintf("%s/%s", ns, name)
		q.Add(key)

		numRequeues := q.NumRequeues(key)
		processQueueNext(q, &mockJsmClient{}, func(ns, name string, c jsmClient) error {
			return nil
		})

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

func TestUpsertCondition(t *testing.T) {
	t.Parallel()

	var cs []apis.Condition

	cs = upsertCondition(cs, apis.Condition{
		Type:               readyCondType,
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

	cs = upsertCondition(cs, apis.Condition{
		Type:               readyCondType,
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

	cs = upsertCondition(cs, apis.Condition{
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

func TestShouldEnqueue(t *testing.T) {
	t.Parallel()

	ts := k8smeta.NewTime(time.Now())

	cases := []struct {
		name string
		prev interface{}
		next interface{}

		want bool
	}{
		{
			name: "stream deleted",
			prev: &apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace: "default",
					Name:      "obj-name",
				},
			},
			next: &apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace:         "default",
					Name:              "obj-name",
					DeletionTimestamp: &ts,
				},
			},
			want: true,
		},
		{
			name: "stream spec changed",
			prev: &apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace: "default",
					Name:      "obj-name",
				},
				Spec: apis.StreamSpec{
					Name: "foo",
				},
			},
			next: &apis.Stream{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace: "default",
					Name:      "obj-name",
				},
				Spec: apis.StreamSpec{
					Name: "bar",
				},
			},
			want: true,
		},
		{
			name: "stream template deleted",
			prev: &apis.StreamTemplate{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace: "default",
					Name:      "obj-name",
				},
			},
			next: &apis.StreamTemplate{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace:         "default",
					Name:              "obj-name",
					DeletionTimestamp: &ts,
				},
			},
			want: true,
		},
		{
			name: "stream template spec changed",
			prev: &apis.StreamTemplate{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace: "default",
					Name:      "obj-name",
				},
				Spec: apis.StreamTemplateSpec{
					StreamSpec: apis.StreamSpec{
						Name: "foo",
					},
				},
			},
			next: &apis.StreamTemplate{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace: "default",
					Name:      "obj-name",
				},
				Spec: apis.StreamTemplateSpec{
					StreamSpec: apis.StreamSpec{
						Name: "bar",
					},
				},
			},
			want: true,
		},
		{
			name: "consumer deleted",
			prev: &apis.Consumer{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace: "default",
					Name:      "obj-name",
				},
			},
			next: &apis.Consumer{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace:         "default",
					Name:              "obj-name",
					DeletionTimestamp: &ts,
				},
			},
			want: true,
		},
		{
			name: "consumer spec changed",
			prev: &apis.Consumer{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace: "default",
					Name:      "obj-name",
				},
				Spec: apis.ConsumerSpec{
					DurableName: "foo",
				},
			},
			next: &apis.Consumer{
				ObjectMeta: k8smeta.ObjectMeta{
					Namespace: "default",
					Name:      "obj-name",
				},
				Spec: apis.ConsumerSpec{
					DurableName: "bar",
				},
			},
			want: true,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := shouldEnqueue(c.prev, c.next)
			if got != c.want {
				t.Fatalf("got=%t; want=%t", got, c.want)
			}
		})
	}
}
