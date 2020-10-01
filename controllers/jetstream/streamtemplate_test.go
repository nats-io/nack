package jetstream

import (
	"context"
	"strings"
	"testing"

	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta1"
	clientsetfake "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/fake"

	jsmapi "github.com/nats-io/jsm.go/api"

	k8sapis "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclientsetfake "k8s.io/client-go/kubernetes/fake"
	k8stypedfake "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

func TestProcessStreamTemplate(t *testing.T) {
	t.Parallel()

	updateObject := func(a k8stesting.Action) (handled bool, o runtime.Object, err error) {
		ua, ok := a.(k8stesting.UpdateAction)
		if !ok {
			return false, nil, nil
		}

		return true, ua.GetObject(), nil
	}

	t.Run("create stream template", func(t *testing.T) {
		t.Parallel()

		jc := clientsetfake.NewSimpleClientset()
		wantEvents := 4
		rec := record.NewFakeRecorder(wantEvents)
		ctrl := NewController(Options{
			Ctx:            context.Background(),
			KubeIface:      k8sclientsetfake.NewSimpleClientset(),
			JetstreamIface: jc,
			Recorder:       rec,
		})

		ns, name := "default", "my-stream-template"

		informer := ctrl.informerFactory.Jetstream().V1beta1().StreamTemplates()
		err := informer.Informer().GetStore().Add(&apis.StreamTemplate{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Generation: 1,
			},
			Spec: apis.StreamTemplateSpec{
				StreamSpec: apis.StreamSpec{
					Name:   name,
					MaxAge: "1h",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "streamtemplates", updateObject)

		notFoundErr := jsmapi.ApiError{Code: 404}
		jsmc := &mockJsmClient{
			loadStreamTemplateErr: notFoundErr,
			newStreamTemplateErr:  nil,
			newStreamTemplate:     &mockDeleter{},
		}
		if err := ctrl.processStreamTemplate(ns, name, jsmc); err != nil {
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

	t.Run("create stream template with credentials", func(t *testing.T) {
		t.Parallel()

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

		ns, name := "default", "my-stream-template"

		informer := ctrl.informerFactory.Jetstream().V1beta1().StreamTemplates()
		err := informer.Informer().GetStore().Add(&apis.StreamTemplate{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Generation: 1,
			},
			Spec: apis.StreamTemplateSpec{
				StreamSpec: apis.StreamSpec{
					Name:   name,
					MaxAge: "1h",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "streamtemplates", updateObject)
		kc.CoreV1().(*k8stypedfake.FakeCoreV1).PrependReactor("get", "secrets", getSecret)

		notFoundErr := jsmapi.ApiError{Code: 404}
		jsmc := &mockJsmClient{
			loadStreamTemplateErr: notFoundErr,
			newStreamTemplateErr:  nil,
			newStreamTemplate:     &mockDeleter{},
		}
		if err := ctrl.processStreamTemplate(ns, name, jsmc); err != nil {
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

	t.Run("update stream template", func(t *testing.T) {
		t.Parallel()

		jc := clientsetfake.NewSimpleClientset()
		wantEvents := 3
		rec := record.NewFakeRecorder(wantEvents)
		ctrl := NewController(Options{
			Ctx:            context.Background(),
			KubeIface:      k8sclientsetfake.NewSimpleClientset(),
			JetstreamIface: jc,
			Recorder:       rec,
		})

		ns, name := "default", "my-stream-template"

		informer := ctrl.informerFactory.Jetstream().V1beta1().StreamTemplates()
		err := informer.Informer().GetStore().Add(&apis.StreamTemplate{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Generation: 1,
			},
			Spec: apis.StreamTemplateSpec{
				StreamSpec: apis.StreamSpec{
					Name:   name,
					MaxAge: "1h",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "streamtemplates", updateObject)

		jsmc := &mockJsmClient{
			loadStreamTemplateErr: nil,
			loadStreamTemplate:    &mockDeleter{},
		}
		if err := ctrl.processStreamTemplate(ns, name, jsmc); err != nil {
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
			if !strings.Contains(gotEvent, "Updating") {
				t.Error("unexpected event")
				t.Fatalf("got=%s; want=%s", gotEvent, "Updating...")
			}
		}
	})

	t.Run("delete stream template", func(t *testing.T) {
		t.Parallel()

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
		ns, name := "default", "my-stream-template"

		informer := ctrl.informerFactory.Jetstream().V1beta1().StreamTemplates()
		err := informer.Informer().GetStore().Add(&apis.StreamTemplate{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:         ns,
				Name:              name,
				Generation:        1,
				DeletionTimestamp: &ts,
			},
			Spec: apis.StreamTemplateSpec{
				StreamSpec: apis.StreamSpec{
					Name:   name,
					MaxAge: "1h",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "streamtemplates", updateObject)

		jsmc := &mockJsmClient{
			loadStreamTemplateErr: nil,
			loadStreamTemplate:    &mockDeleter{},
		}
		if err := ctrl.processStreamTemplate(ns, name, jsmc); err != nil {
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
			if !strings.Contains(gotEvent, "Deleting") {
				t.Error("unexpected event")
				t.Fatalf("got=%s; want=%s", gotEvent, "Deleting...")
			}
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

		ns, name := "default", "my-stream-template"

		informer := ctrl.informerFactory.Jetstream().V1beta1().StreamTemplates()
		err := informer.Informer().GetStore().Add(&apis.StreamTemplate{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace:  ns,
				Name:       name,
				Generation: 1,
			},
			Spec: apis.StreamTemplateSpec{
				StreamSpec: apis.StreamSpec{
					Name:   name,
					MaxAge: "1h",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		jc.PrependReactor("update", "streamtemplates", func(a k8stesting.Action) (handled bool, o runtime.Object, err error) {
			ua, ok := a.(k8stesting.UpdateAction)
			if !ok {
				return false, nil, nil
			}
			obj := ua.GetObject()

			str, ok := obj.(*apis.StreamTemplate)
			if !ok {
				t.Error("unexpected object type")
				t.Fatalf("got=%T; want=%T", obj, &apis.StreamTemplate{})
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

		// jsmc := &mockJsmClient{
		// 	connectErr: errors.New("nats connect failed"),
		// }
		// if err := ctrl.processStreamTemplate(ns, name, jsmc); err == nil {
		// 	t.Fatal("unexpected success")
		// }
	})
}
