package jetstream

import (
	"context"
	"testing"

	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	clientsetfake "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/fake"

	k8sapi "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclientsetfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"
)

func TestGetAccountOverrides_NKey(t *testing.T) {
	t.Parallel()

	t.Run("nkey from secret", func(t *testing.T) {
		t.Parallel()

		ns := "default"
		accountName := "test-nkey-account"
		secretName := "test-nkey-secret"
		seedKey := "user.nk"
		seedData := "SUAXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"

		// Create the NKey secret
		nkeySecret := &k8sapi.Secret{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: ns,
				Name:      secretName,
			},
			Data: map[string][]byte{
				seedKey: []byte(seedData),
			},
		}

		// Create the Account with NKey reference
		account := &apis.Account{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: ns,
				Name:      accountName,
			},
			Spec: apis.AccountSpec{
				Servers: []string{"nats://localhost:4222"},
				NKey: &apis.NKeySecret{
					Seed: seedKey,
					Secret: &apis.SecretRef{
						Name: secretName,
					},
				},
			},
		}

		ki := k8sclientsetfake.NewSimpleClientset(nkeySecret)
		ji := clientsetfake.NewSimpleClientset(account)

		ctrl := &Controller{
			ctx:  context.Background(),
			ki:   ki.CoreV1(),
			ji:   ji.JetstreamV1beta2(),
			opts: Options{CRDConnect: true},
		}

		overrides, err := ctrl.getAccountOverrides(accountName, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got, want := overrides.nkey, seedData; got != want {
			t.Errorf("nkey mismatch: got=%q, want=%q", got, want)
		}

		// Verify other auth methods are not set
		if overrides.userCreds != "" {
			t.Errorf("expected empty userCreds, got=%q", overrides.userCreds)
		}
		if overrides.user != "" {
			t.Errorf("expected empty user, got=%q", overrides.user)
		}
		if overrides.token != "" {
			t.Errorf("expected empty token, got=%q", overrides.token)
		}
	})

	t.Run("nkey with missing seed key in secret", func(t *testing.T) {
		t.Parallel()

		ns := "default"
		accountName := "test-nkey-bad-key"
		secretName := "test-nkey-bad-secret"

		nkeySecret := &k8sapi.Secret{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: ns,
				Name:      secretName,
			},
			Data: map[string][]byte{
				"wrong-key.nk": []byte("some-seed-data"),
			},
		}

		account := &apis.Account{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: ns,
				Name:      accountName,
			},
			Spec: apis.AccountSpec{
				Servers: []string{"nats://localhost:4222"},
				NKey: &apis.NKeySecret{
					Seed: "user.nk",
					Secret: &apis.SecretRef{
						Name: secretName,
					},
				},
			},
		}

		ki := k8sclientsetfake.NewSimpleClientset(nkeySecret)
		ji := clientsetfake.NewSimpleClientset(account)

		ctrl := &Controller{
			ctx:  context.Background(),
			ki:   ki.CoreV1(),
			ji:   ji.JetstreamV1beta2(),
			opts: Options{CRDConnect: true},
		}

		overrides, err := ctrl.getAccountOverrides(accountName, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if overrides.nkey != "" {
			t.Errorf("expected empty nkey when seed key not found, got=%q", overrides.nkey)
		}
	})

	t.Run("nkey with missing secret", func(t *testing.T) {
		t.Parallel()

		ns := "default"
		accountName := "test-nkey-no-secret"

		account := &apis.Account{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: ns,
				Name:      accountName,
			},
			Spec: apis.AccountSpec{
				Servers: []string{"nats://localhost:4222"},
				NKey: &apis.NKeySecret{
					Seed: "user.nk",
					Secret: &apis.SecretRef{
						Name: "nonexistent-secret",
					},
				},
			},
		}

		ki := k8sclientsetfake.NewSimpleClientset() // No secrets
		ji := clientsetfake.NewSimpleClientset(account)

		ctrl := &Controller{
			ctx:  context.Background(),
			ki:   ki.CoreV1(),
			ji:   ji.JetstreamV1beta2(),
			opts: Options{CRDConnect: true},
		}

		_, err := ctrl.getAccountOverrides(accountName, ns)
		if err == nil {
			t.Fatal("expected error when secret does not exist, got nil")
		}
	})

	t.Run("nkey not set when CRDConnect is false", func(t *testing.T) {
		t.Parallel()

		ctrl := &Controller{
			ctx:  context.Background(),
			opts: Options{CRDConnect: false},
		}

		overrides, err := ctrl.getAccountOverrides("some-account", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if overrides.nkey != "" {
			t.Errorf("expected empty nkey when CRDConnect is false, got=%q", overrides.nkey)
		}
	})

	t.Run("nkey nil when NKey spec is nil", func(t *testing.T) {
		t.Parallel()

		ns := "default"
		accountName := "test-no-nkey"

		account := &apis.Account{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: ns,
				Name:      accountName,
			},
			Spec: apis.AccountSpec{
				Servers: []string{"nats://localhost:4222"},
			},
		}

		ki := k8sclientsetfake.NewSimpleClientset()
		ji := clientsetfake.NewSimpleClientset(account)

		ctrl := &Controller{
			ctx:  context.Background(),
			ki:   ki.CoreV1(),
			ji:   ji.JetstreamV1beta2(),
			opts: Options{CRDConnect: true},
		}

		overrides, err := ctrl.getAccountOverrides(accountName, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if overrides.nkey != "" {
			t.Errorf("expected empty nkey when NKey spec is nil, got=%q", overrides.nkey)
		}
	})
}

func TestRunWithJsmc_NKeyFromAccount(t *testing.T) {
	t.Parallel()

	ns := "default"
	accountName := "test-nkey-run-account"
	secretName := "test-nkey-run-secret"
	seedKey := "user.nk"
	seedData := "SUAXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"

	nkeySecret := &k8sapi.Secret{
		ObjectMeta: k8smeta.ObjectMeta{
			Namespace: ns,
			Name:      secretName,
		},
		Data: map[string][]byte{
			seedKey: []byte(seedData),
		},
	}

	account := &apis.Account{
		ObjectMeta: k8smeta.ObjectMeta{
			Namespace: ns,
			Name:      accountName,
		},
		Spec: apis.AccountSpec{
			Servers: []string{"nats://localhost:4222"},
			NKey: &apis.NKeySecret{
				Seed: seedKey,
				Secret: &apis.SecretRef{
					Name: secretName,
				},
			},
		},
	}

	ki := k8sclientsetfake.NewSimpleClientset(nkeySecret)
	ji := clientsetfake.NewSimpleClientset(account)

	rec := record.NewFakeRecorder(10)
	ctrl := &Controller{
		ctx: context.Background(),
		ki:  ki.CoreV1(),
		ji:  ji.JetstreamV1beta2(),
		rec: rec,
		opts: Options{
			Ctx:        context.Background(),
			CRDConnect: true,
		},
	}

	acc, err := ctrl.getAccountOverrides(accountName, ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the nkey is passed through to runWithJsmc
	var capturedNatsCtx *natsContext
	mockJsm := func(nc *natsContext) (jsmClient, error) {
		capturedNatsCtx = nc
		return &mockJsmClient{}, nil
	}

	stream := &apis.Stream{
		ObjectMeta: k8smeta.ObjectMeta{
			Namespace: ns,
			Name:      "test-stream",
		},
	}

	spec := &jsmcSpecOverrides{
		servers: []string{"nats://localhost:4222"},
	}

	err = ctrl.runWithJsmc(mockJsm, acc, spec, stream, func(c jsmClient) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedNatsCtx == nil {
		t.Fatal("expected nats context to be captured")
	}

	if got, want := capturedNatsCtx.Nkey, seedData; got != want {
		t.Errorf("nkey mismatch in natsContext: got=%q, want=%q", got, want)
	}
}
