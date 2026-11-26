package controller

import (
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
