package controller

import (
	"fmt"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsConfig struct {
	CRDConnect  bool
	ClientName  string
	Credentials string
	NKey        string
	ServerURL   string
	CAs         []string
	Certificate string
	Key         string
	TLSFirst    bool
}

// buildOptions creates options from the config to be used in nats.Connect.
func (o *NatsConfig) buildOptions() ([]nats.Option, error) {
	opts := make([]nats.Option, 0)

	if o.ServerURL == "" {
		return nil, fmt.Errorf("server url is required")
	}

	if o.TLSFirst {
		opts = append(opts, nats.TLSHandshakeFirst())
	}

	if o.ClientName != "" {
		opts = append(opts, nats.Name(o.ClientName))
	}

	if o.Credentials != "" {
		opts = append(opts, nats.UserCredentials(o.Credentials))
	}

	if o.NKey != "" {
		opt, err := nats.NkeyOptionFromSeed(o.NKey)
		if err != nil {
			return nil, fmt.Errorf("nkey option from seed: %w", err)
		}
		opts = append(opts, opt)
	}

	if o.Certificate != "" && o.Key != "" {
		opts = append(opts, nats.ClientCert(o.Certificate, o.Key))
	}

	if len(o.CAs) > 0 {
		opts = append(opts, nats.RootCAs(o.CAs...))
	}

	return opts, nil
}

type Closable interface {
	Close()
}

func CreateJSMClient(cfg *NatsConfig, pedantic bool) (*jsm.Manager, Closable, error) {
	nc, err := createNatsConn(cfg, pedantic)
	if err != nil {
		return nil, nil, fmt.Errorf("create nats connection: %w", err)
	}

	major, minor, _, err := versionComponents(nc.ConnectedServerVersion())
	if err != nil {
		return nil, nil, fmt.Errorf("parse server version: %w", err)
	}

	// JetStream pedantic mode unsupported prior to NATS Server 2.11
	if pedantic && major < 2 || (major == 2 && minor < 11) {
		pedantic = false
	}

	jsmOpts := make([]jsm.Option, 0)
	if pedantic {
		jsmOpts = append(jsmOpts, jsm.WithPedanticRequests())
	}

	jsmClient, err := jsm.New(nc, jsmOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("new jsm client: %w", err)
	}

	return jsmClient, nc, nil
}

// CreateJetStreamClient creates new Jetstream client with a connection based on the given NatsConfig.
// Returns a jetstream.Jetstream client and the Closable of the underlying connection.
// Close should be called when the client is no longer used.
func CreateJetStreamClient(cfg *NatsConfig, pedantic bool) (jetstream.JetStream, Closable, error) {
	nc, err := createNatsConn(cfg, pedantic)
	if err != nil {
		return nil, nil, fmt.Errorf("create nats connection: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, nil, fmt.Errorf("new jetstream: %w", err)
	}
	return js, nc, nil
}

func createNatsConn(cfg *NatsConfig, pedantic bool) (*nats.Conn, error) {
	opts, err := cfg.buildOptions()
	if err != nil {
		return nil, err
	}

	// Set pedantic option
	if pedantic {
		opts = append(opts, func(options *nats.Options) error {
			options.Pedantic = true
			return nil
		})
	}

	// client should always attempt to reconnect
	opts = append(opts, nats.MaxReconnects(-1))

	return nats.Connect(cfg.ServerURL, opts...)
}
