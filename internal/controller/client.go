package controller

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsConfig struct {
	CRDConnect  bool
	ClientName  string
	Credentials string
	NKey        string
	ServerURL   string
	CA          string
	Certificate string
	Key         string
	TLSFirst    bool
}

// buildOptions creates options from the config to be used in nats.Connect.
func (o *NatsConfig) buildOptions() ([]nats.Option, error) {
	opts := make([]nats.Option, 0)

	if o.TLSFirst {
		opts = append(opts, nats.TLSHandshakeFirst())
	}

	if o.ClientName != "" {
		opts = append(opts, nats.Name(o.ClientName))
	}

	if !o.CRDConnect {
		// Use JWT/NKEYS based credentials if present.
		if o.Credentials != "" {
			opts = append(opts, nats.UserCredentials(o.Credentials))
		} else if o.NKey != "" {
			opt, err := nats.NkeyOptionFromSeed(o.NKey)
			if err != nil {
				return nil, fmt.Errorf("nkey option from seed: %w", err)
			}
			opts = append(opts, opt)
		}

		if o.Certificate != "" && o.Key != "" {
			opts = append(opts, nats.ClientCert(o.Certificate, o.Key))
		}

		if o.CA != "" {
			opts = append(opts, nats.RootCAs(o.CA))
		}
	}

	return opts, nil
}

func CreateJetStreamClient(cfg *NatsConfig, pedantic bool) (jetstream.JetStream, error) {

	opts, err := cfg.buildOptions()
	if err != nil {
		return nil, fmt.Errorf("nats options: %w", err)
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

	nc, err := nats.Connect(cfg.ServerURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("new jetstream: %w", err)
	}
	return js, nil
}
