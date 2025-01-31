package controller

import (
	"fmt"
	js "github.com/nats-io/nack/controllers/jetstream"
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/nats-io/nats.go/jetstream"
	v1 "k8s.io/api/core/v1"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type connectionOptions struct {
	Account string   `json:"account"`
	Creds   string   `json:"creds"`
	Nkey    string   `json:"nkey"`
	Servers []string `json:"servers"`
	TLS     api.TLS  `json:"tls"`
}

type JetStreamController interface {
	client.Client

	// ReadOnly returns true when no changes should be made by the controller.
	ReadOnly() bool

	// ValidNamespace ok if the controllers namespace restriction allows the given namespace.
	ValidNamespace(namespace string) bool

	// WithJetStreamClient provides a jetStream client to the given operation.
	// The client uses the controllers connection configuration merged with opts.
	//
	// The given opts values take precedence over the controllers base configuration.
	//
	// Returns the error of the operation or errors during client setup.
	WithJetStreamClient(opts *connectionOptions, op func(js jetstream.JetStream) error) error
}

func NewJSController(k8sClient client.Client, natsConfig *NatsConfig, controllerConfig *Config) (JetStreamController, error) {

	return &jsController{
		Client:           k8sClient,
		config:           natsConfig,
		controllerConfig: controllerConfig,
	}, nil
}

type jsController struct {
	client.Client
	config           *NatsConfig
	controllerConfig *Config
}

func (c *jsController) ReadOnly() bool {
	return c.controllerConfig.ReadOnly
}

func (c *jsController) ValidNamespace(namespace string) bool {
	ns := c.controllerConfig.Namespace
	return ns == "" || ns == namespace
}

func (c *jsController) WithJetStreamClient(opts *connectionOptions, op func(js jetstream.JetStream) error) error {

	// Build single use client
	// TODO(future-feature): Use client-pool instead of single use client
	cfg := c.buildNatsConfig(opts)

	jsClient, closer, err := CreateJetStreamClient(cfg, true)
	if err != nil {
		return fmt.Errorf("create jetstream client: %w", err)
	}
	defer closer.Close()

	return op(jsClient)
}

// buildNatsConfig uses given opts to override the base NatsConfig.
func (c *jsController) buildNatsConfig(opts *connectionOptions) *NatsConfig {

	serverUrls := strings.Join(opts.Servers, ",")

	// Takes opts values if present
	cfg := &NatsConfig{
		CRDConnect: false,
		ClientName: c.config.ClientName,
		ServerURL:  or(serverUrls, c.config.ServerURL),
		TLSFirst:   c.config.TLSFirst, // TODO(future-feature): expose TLSFirst in the spec config
	}

	// Note: The opts.Account value coming from the resource spec is currently not considered.
	// creds/nkey are associated with an account, the account field might be redundant.
	// See https://github.com/nats-io/nack/pull/211#pullrequestreview-2511111670

	// Authentication either from opts or base config
	if opts.Creds != "" || opts.Nkey != "" {
		cfg.Credentials = opts.Creds
		cfg.NKey = opts.Nkey
	} else {
		cfg.Credentials = c.config.Credentials
		cfg.NKey = c.config.NKey
	}

	// CAs from opts or base config
	if len(opts.TLS.RootCAs) > 0 {
		cfg.CAs = opts.TLS.RootCAs
	} else {
		cfg.CAs = c.config.CAs
	}

	// Client Cert and Key either from opts or base config
	if opts.TLS.ClientCert != "" && opts.TLS.ClientKey != "" {
		cfg.Certificate = opts.TLS.ClientCert
		cfg.Key = opts.TLS.ClientKey
	} else {
		cfg.Certificate = c.config.Certificate
		cfg.Key = c.config.Key
	}

	return cfg
}

// or returns the value if it is not the null value. Otherwise, the fallback value is returned
func or[T comparable](v T, fallback T) T {
	if v == *new(T) {
		return fallback
	}
	return v
}

// updateReadyCondition returns the given conditions with an added or updated ready condition.
func updateReadyCondition(conditions []api.Condition, status v1.ConditionStatus, reason string, message string) []api.Condition {

	var currentStatus v1.ConditionStatus
	var lastTransitionTime string
	for _, condition := range conditions {
		if condition.Type == readyCondType {
			currentStatus = condition.Status
			lastTransitionTime = condition.LastTransitionTime
			break
		}
	}

	// Set transition time to now, when no previous ready condition or the status changed
	if lastTransitionTime == "" || currentStatus != status {
		lastTransitionTime = time.Now().UTC().Format(time.RFC3339Nano)
	}

	newCondition := api.Condition{
		Type:               readyCondType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: lastTransitionTime,
	}
	if conditions == nil {
		return []api.Condition{newCondition}
	} else {
		return js.UpsertCondition(conditions, newCondition)
	}
}

// asJsonString returns the given string wrapped in " and converted to []byte.
// Helper for mapping spec config to jetStream config using UnmarshalJSON.
func asJsonString(v string) []byte {
	return []byte("\"" + v + "\"")
}
