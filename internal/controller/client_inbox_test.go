package controller

import (
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestBuildOptionsInboxPrefix(t *testing.T) {
	for _, tc := range []struct {
		name, prefix, want string
	}{
		{"unset keeps the default", "", ""},
		{"plain prefix", "_INBOX.nack", "_INBOX.nack"},
		{"trailing dot is trimmed", "_INBOX.nack.", "_INBOX.nack"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &NatsConfig{ServerURL: "nats://localhost:4222", InboxPrefix: tc.prefix}
			opts, err := cfg.buildOptions()
			if err != nil {
				t.Fatal(err)
			}

			var got nats.Options
			for _, opt := range opts {
				if err := opt(&got); err != nil {
					t.Fatal(err)
				}
			}
			if got.InboxPrefix != tc.want {
				t.Fatalf("InboxPrefix = %q, want %q", got.InboxPrefix, tc.want)
			}
		})
	}
}

func TestOverlayInboxPrefix(t *testing.T) {
	base := &NatsConfig{InboxPrefix: "_INBOX.global"}
	base.Overlay(&NatsConfig{})
	if base.InboxPrefix != "_INBOX.global" {
		t.Fatalf("empty overlay changed the prefix to %q", base.InboxPrefix)
	}
	base.Overlay(&NatsConfig{InboxPrefix: "_INBOX.account"})
	if base.InboxPrefix != "_INBOX.account" {
		t.Fatalf("overlay prefix not applied, got %q", base.InboxPrefix)
	}
}

func TestNatsConfigFromOptsAccountInboxPrefix(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := api.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, global, account, want string
	}{
		{"global only", "_INBOX.global", "", "_INBOX.global"},
		{"account overrides global", "_INBOX.global", "_INBOX.account", "_INBOX.account"},
		{"account without global", "", "_INBOX.account", "_INBOX.account"},
		{"neither", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			acc := &api.Account{
				ObjectMeta: metav1.ObjectMeta{Name: "acc", Namespace: "ns"},
				Spec:       api.AccountSpec{InboxPrefix: tc.account},
			}
			c := &jsController{
				Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(acc).Build(),
				config: &NatsConfig{ServerURL: "nats://localhost:4222", InboxPrefix: tc.global},
			}

			cfg, err := c.natsConfigFromOpts(api.ConnectionOpts{Account: "acc"}, "ns")
			if err != nil {
				t.Fatal(err)
			}
			if cfg.InboxPrefix != tc.want {
				t.Fatalf("InboxPrefix = %q, want %q", cfg.InboxPrefix, tc.want)
			}
		})
	}
}
