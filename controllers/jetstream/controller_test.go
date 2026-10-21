package jetstream

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	jsmapi "github.com/nats-io/jsm.go/api"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	clientsetfake "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/fake"

	k8sapis "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	k8sclientsetfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"
)

func TestMain(m *testing.M) {
	// Disable error logs.
	utilruntime.ErrorHandlers = []utilruntime.ErrorHandler{
		func(ctx context.Context, err error, msg string, args ...any) {},
	}

	os.Exit(m.Run())
}

func TestGetAccountOverridesRejectsInvalidAuth(t *testing.T) {
	authSecret := &apis.SecretRef{Name: "account-auth"}
	tests := []struct {
		name       string
		spec       apis.AccountSpec
		secretData map[string][]byte
		wantErr    string
	}{
		{
			name: "empty credentials",
			spec: apis.AccountSpec{Creds: &apis.CredsSecret{
				File:   "user.creds",
				Secret: authSecret,
			}},
			secretData: map[string][]byte{"user.creds": {}},
			wantErr:    `account "test-account" credentials key "user.creds" in secret "account-auth" is empty`,
		},
		{
			name: "missing nkey seed",
			spec: apis.AccountSpec{NKey: &apis.NKeySecret{
				Seed:   "seed",
				Secret: authSecret,
			}},
			secretData: map[string][]byte{},
			wantErr:    `account "test-account" nkey seed key "seed" not found in secret "account-auth"`,
		},
		{
			name: "empty token",
			spec: apis.AccountSpec{Token: &apis.TokenSecret{
				Token:  "token",
				Secret: *authSecret,
			}},
			secretData: map[string][]byte{"token": {}},
			wantErr:    `account "test-account" token key "token" in secret "account-auth" is empty`,
		},
		{
			name: "missing password",
			spec: apis.AccountSpec{User: &apis.User{
				User:     "username",
				Password: "password",
				Secret:   *authSecret,
			}},
			secretData: map[string][]byte{"username": []byte("user")},
			wantErr:    `account "test-account" password key "password" not found in secret "account-auth"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &apis.Account{
				ObjectMeta: k8smeta.ObjectMeta{Name: "test-account", Namespace: "default"},
				Spec:       tt.spec,
			}
			secret := &k8sapis.Secret{
				ObjectMeta: k8smeta.ObjectMeta{Name: authSecret.Name, Namespace: account.Namespace},
				Data:       tt.secretData,
			}
			controller := &Controller{
				ctx:      context.Background(),
				opts:     Options{CRDConnect: true},
				ji:       clientsetfake.NewSimpleClientset(account).JetstreamV1beta2(),
				ki:       k8sclientsetfake.NewSimpleClientset(secret).CoreV1(),
				cacheDir: t.TempDir(),
			}

			overrides, err := controller.getAccountOverrides(account.Name, account.Namespace)

			if overrides != nil {
				t.Fatalf("got overrides %v; want nil", overrides)
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("got error %v; want %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetAccountOverridesWritesNKeySeedFile(t *testing.T) {
	seed := []byte("SUABCDEFGHIJKLMNOP")
	account := &apis.Account{
		ObjectMeta: k8smeta.ObjectMeta{Name: "test-account", Namespace: "default"},
		Spec: apis.AccountSpec{NKey: &apis.NKeySecret{
			Seed:   "seed",
			Secret: &apis.SecretRef{Name: "account-auth"},
		}},
	}
	secret := &k8sapis.Secret{
		ObjectMeta: k8smeta.ObjectMeta{Name: "account-auth", Namespace: account.Namespace},
		Data:       map[string][]byte{"seed": seed},
	}
	controller := &Controller{
		ctx:      context.Background(),
		opts:     Options{CRDConnect: true},
		ji:       clientsetfake.NewSimpleClientset(account).JetstreamV1beta2(),
		ki:       k8sclientsetfake.NewSimpleClientset(secret).CoreV1(),
		cacheDir: t.TempDir(),
	}

	overrides, err := controller.getAccountOverrides(account.Name, account.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	writtenSeed, err := os.ReadFile(overrides.nkey)
	if err != nil {
		t.Fatal(err)
	}
	if string(writtenSeed) != string(seed) {
		t.Fatalf("got seed %q; want %q", writtenSeed, seed)
	}
	info, err := os.Stat(overrides.nkey)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("got nkey mode %o; want 600", info.Mode().Perm())
	}
}

func TestGetAccountOverridesAllowsEmptyPassword(t *testing.T) {
	tests := []struct {
		name        string
		passwordKey string
		secretData  map[string][]byte
	}{
		{
			name:       "omitted password selector",
			secretData: map[string][]byte{"username": []byte("user")},
		},
		{
			name:        "empty password secret value",
			passwordKey: "password",
			secretData:  map[string][]byte{"username": []byte("user"), "password": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &apis.Account{
				ObjectMeta: k8smeta.ObjectMeta{Name: "test-account", Namespace: "default"},
				Spec: apis.AccountSpec{User: &apis.User{
					User:     "username",
					Password: tt.passwordKey,
					Secret:   apis.SecretRef{Name: "account-auth"},
				}},
			}
			secret := &k8sapis.Secret{
				ObjectMeta: k8smeta.ObjectMeta{Name: "account-auth", Namespace: account.Namespace},
				Data:       tt.secretData,
			}
			controller := &Controller{
				ctx:      context.Background(),
				opts:     Options{CRDConnect: true},
				ji:       clientsetfake.NewSimpleClientset(account).JetstreamV1beta2(),
				ki:       k8sclientsetfake.NewSimpleClientset(secret).CoreV1(),
				cacheDir: t.TempDir(),
			}

			overrides, err := controller.getAccountOverrides(account.Name, account.Namespace)
			if err != nil {
				t.Fatal(err)
			}
			if overrides.user != "user" || overrides.password != "" {
				t.Fatalf("got user/password %q/%q; want user with empty password", overrides.user, overrides.password)
			}
		})
	}
}

func TestRunWithJsmcAllowsEmptyAccountPassword(t *testing.T) {
	controller := &Controller{opts: Options{CRDConnect: true}}
	var got *natsContext
	jsm := func(ctx *natsContext) (jsmClient, error) {
		got = ctx
		return &mockJsmClient{}, nil
	}

	err := controller.runWithJsmc(
		jsm,
		&accountOverrides{user: "user"},
		&jsmcSpecOverrides{},
		nil,
		func(jsmClient) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "user" || got.Password != "" {
		t.Fatalf("got user/password %q/%q; want user with empty password", got.Username, got.Password)
	}
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

func TestEnqueueWork(t *testing.T) {
	t.Parallel()

	limiter := workqueue.DefaultTypedControllerRateLimiter[any]()
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

		limiter := workqueue.DefaultTypedControllerRateLimiter[any]()
		q := workqueue.NewNamedRateLimitingQueue(limiter, "StreamsTest")
		defer q.ShutDown()

		key := "this/is/a/bad/key"
		q.Add(key)

		processQueueNext(q, testWrapJSMC(&mockJsmClient{}), func(ns, name string, c jsmClientFunc) error {
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

		limiter := workqueue.DefaultTypedControllerRateLimiter[any]()
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

			processQueueNext(q, testWrapJSMC(&mockJsmClient{}), func(ns, name string, c jsmClientFunc) error {
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

		limiter := workqueue.DefaultTypedControllerRateLimiter[any]()
		q := workqueue.NewNamedRateLimitingQueue(limiter, "StreamsTest")
		defer q.ShutDown()

		ns, name := "default", "mystream"
		key := fmt.Sprintf("%s/%s", ns, name)
		q.Add(key)

		numRequeues := q.NumRequeues(key)
		processQueueNext(q, testWrapJSMC(&mockJsmClient{}), func(ns, name string, c jsmClientFunc) error {
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

	cs = UpsertCondition(cs, apis.Condition{
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

	cs = UpsertCondition(cs, apis.Condition{
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

	cs = UpsertCondition(cs, apis.Condition{
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
