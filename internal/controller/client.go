package controller

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsConfig struct {
	ClientName  string   `json:"name,omitempty"`
	ServerURL   string   `json:"url,omitempty"`
	Certificate string   `json:"tls_cert,omitempty"`
	Key         string   `json:"tls_key,omitempty"`
	TLSFirst    bool     `json:"tls_first,omitempty"`
	CAs         []string `json:"tls_ca,omitempty"`
	Credentials string   `json:"credential,omitempty"`
	NKey        string   `json:"nkey,omitempty"`
	Token       string   `json:"token,omitempty"`
	User        string   `json:"username,omitempty"`
	Password    string   `json:"password,omitempty"`
	JsDomain    string   `json:"js_domain,omitempty"`
}

func (o *NatsConfig) Copy() *NatsConfig {
	if o == nil {
		return nil
	}

	cp := *o
	return &cp
}

func (o *NatsConfig) Hash() (string, error) {
	b, err := json.Marshal(o)
	if err != nil {
		return "", fmt.Errorf("error marshaling config to json: %v", err)
	}

	if o.NKey != "" {
		fb, err := os.ReadFile(o.NKey)
		if err != nil {
			return "", fmt.Errorf("error opening nkey file %s: %v", o.NKey, err)
		}
		b = append(b, fb...)
	}

	if o.Credentials != "" {
		fb, err := os.ReadFile(o.Credentials)
		if err != nil {
			return "", fmt.Errorf("error opening creds file %s: %v", o.Credentials, err)
		}
		b = append(b, fb...)
	}

	if len(o.CAs) > 0 {
		for _, cert := range o.CAs {
			fb, err := os.ReadFile(cert)
			if err != nil {
				return "", fmt.Errorf("error opening ca file %s: %v", cert, err)
			}
			b = append(b, fb...)
		}
	}

	if o.Certificate != "" {
		fb, err := os.ReadFile(o.Certificate)
		if err != nil {
			return "", fmt.Errorf("error opening cert file %s: %v", o.Certificate, err)
		}
		b = append(b, fb...)
	}

	if o.Key != "" {
		fb, err := os.ReadFile(o.Key)
		if err != nil {
			return "", fmt.Errorf("error opening key file %s: %v", o.Key, err)
		}
		b = append(b, fb...)
	}

	hash := sha256.New()
	hash.Write(b)
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (o *NatsConfig) Overlay(overlay *NatsConfig) {
	if overlay.ClientName != "" {
		o.ClientName = overlay.ClientName
	}

	if overlay.ServerURL != "" {
		o.ServerURL = overlay.ServerURL
	}

	if overlay.Certificate != "" && overlay.Key != "" {
		o.Certificate = overlay.Certificate
		o.Key = overlay.Key
	}

	if len(overlay.CAs) > 0 {
		o.CAs = overlay.CAs
	}

	if overlay.TLSFirst {
		o.TLSFirst = overlay.TLSFirst
	}

	if !overlay.HasAuth() {
		return
	}

	o.UnsetAuth()

	if overlay.Credentials != "" {
		o.Credentials = overlay.Credentials
	} else if overlay.NKey != "" {
		o.NKey = overlay.NKey
	} else if overlay.Token != "" {
		o.Token = overlay.Token
	} else if overlay.User != "" && overlay.Password != "" {
		o.User = overlay.User
		o.Password = overlay.Password
	}
}

func (o *NatsConfig) HasAuth() bool {
	return o.Credentials != "" || o.NKey != "" || o.Token != "" || (o.User != "" && o.Password != "")
}

func (o *NatsConfig) UnsetAuth() {
	o.Credentials = ""
	o.NKey = ""
	o.User = ""
	o.Password = ""
	o.Token = ""
}

// buildOptions creates options from the config to be used in nats.Connect.
func (o *NatsConfig) buildOptions() ([]nats.Option, error) {
	opts := make([]nats.Option, 0)

	if o.ClientName != "" {
		opts = append(opts, nats.Name(o.ClientName))
	}

	if o.ServerURL == "" {
		return nil, fmt.Errorf("server url is required")
	}

	if o.Certificate != "" && o.Key != "" {
		opts = append(opts, nats.ClientCert(o.Certificate, o.Key))
	}

	if o.TLSFirst {
		opts = append(opts, nats.TLSHandshakeFirst())
	}

	if len(o.CAs) > 0 {
		opts = append(opts, nats.RootCAs(o.CAs...))
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

	if o.Token != "" {
		opts = append(opts, nats.Token(o.Token))
	}

	if o.User != "" && o.Password != "" {
		opts = append(opts, nats.UserInfo(o.User, o.Password))
	}

	return opts, nil
}

type Closable interface {
	Close()
}

func CreateJSMClient(conn *pooledConnection, pedantic bool, domain string) (*jsm.Manager, error) {
	if !conn.nc.IsConnected() {
		return nil, errors.New("not connected")
	}

	major, minor, _, err := versionComponents(conn.nc.ConnectedServerVersion())
	if err != nil {
		return nil, fmt.Errorf("parse server version: %w", err)
	}

	// JetStream pedantic mode unsupported prior to NATS Server 2.11
	if pedantic && major < 2 || (major == 2 && minor < 11) {
		pedantic = false
	}

	jsmOpts := make([]jsm.Option, 0)
	if pedantic {
		jsmOpts = append(jsmOpts, jsm.WithPedanticRequests())
	}
	if domain != "" {
		jsmOpts = append(jsmOpts, jsm.WithDomain(domain))
	}

	jsmClient, err := jsm.New(conn.nc, jsmOpts...)
	if err != nil {
		return nil, fmt.Errorf("new jsm client: %w", err)
	}

	return jsmClient, nil
}

// CreateJetStreamClient creates new Jetstream client with a connection based on the given NatsConfig.
// Returns a jetstream.Jetstream client and the Closable of the underlying connection.
// Close should be called when the client is no longer used.
func CreateJetStreamClient(conn *pooledConnection, pedantic bool, domain string) (jetstream.JetStream, error) {
	var (
		err error
		js  jetstream.JetStream
	)

	if domain != "" {
		js, err = jetstream.NewWithDomain(conn.nc, domain)
	} else {
		js, err = jetstream.New(conn.nc)
	}

	if err != nil {
		return nil, fmt.Errorf("new jetstream: %w", err)
	}
	return js, nil
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
