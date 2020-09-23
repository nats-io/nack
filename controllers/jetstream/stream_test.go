package jetstream

import (
	"context"
	"strings"
	"testing"
	"time"

	jsmapi "github.com/nats-io/jsm.go/api"

	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1"
	clientsetfake "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/fake"
	informers "github.com/nats-io/nack/pkg/jetstream/generated/informers/externalversions"

	k8sapis "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclientsetfake "k8s.io/client-go/kubernetes/fake"
	k8stypedfake "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

func TestProcessStream(t *testing.T) {
	t.Parallel()

	updateStream := func(a k8stesting.Action) (handled bool, o runtime.Object, err error) {
		ua, ok := a.(k8stesting.UpdateAction)
		if !ok {
			return false, nil, nil
		}

		return true, ua.GetObject(), nil
	}

	t.Run("create stream", func(t *testing.T) {
		kc := k8sclientsetfake.NewSimpleClientset()
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
					Name:    name,
					MaxAge:  "1h",
					Storage: "memory",
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
			ki:           kc.CoreV1(),
			ji:           jc.JetstreamV1(),
			rec:          frec,
		}

		notFoundErr := jsmapi.ApiError{
			Code: 404,
		}
		jsmc := &mockJsmClient{
			loadStreamErr: notFoundErr,
		}
		if err := ctrl.processStream("default", name, jsmc); err != nil {
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

	t.Run("create stream with credentials", func(t *testing.T) {
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
					Name:    name,
					MaxAge:  "1h",
					Storage: "memory",
					CredentialsSecret: apis.CredentialsSecret{
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
		}

		notFoundErr := jsmapi.ApiError{
			Code: 404,
		}
		jsmc := &mockJsmClient{
			loadStreamErr: notFoundErr,
		}
		if err := ctrl.processStream(ns, name, jsmc); err != nil {
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

	t.Run("update stream", func(t *testing.T) {
		kc := k8sclientsetfake.NewSimpleClientset()
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
					Name:   name,
					MaxAge: "1h",
				},
				Status: apis.Status{
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
			ki:           kc.CoreV1(),
			ji:           jc.JetstreamV1(),
			rec:          frec,
		}

		jsmc := &mockJsmClient{
			loadStreamErr:    nil,
			loadStreamStream: &mockStream{},
		}
		if err := ctrl.processStream("default", name, jsmc); err != nil {
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

	t.Run("delete stream", func(t *testing.T) {
		kc := k8sclientsetfake.NewSimpleClientset()
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
			ki:           kc.CoreV1(),
			ji:           jc.JetstreamV1(),
			rec:          frec,
		}

		jsmc := &mockJsmClient{
			loadStreamErr:    nil,
			loadStreamStream: &mockStream{},
		}
		if err := ctrl.processStream("default", name, jsmc); err != nil {
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
