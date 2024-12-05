package controller

import (
	"errors"
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

func (o connectionOptions) empty() bool {
	return o.Account == "" &&
		o.Creds == "" &&
		o.Nkey == "" &&
		len(o.Servers) == 0 &&
		o.TLS.ClientCert == "" &&
		o.TLS.ClientKey == "" &&
		len(o.TLS.RootCAs) == 0
}

type JetStreamController interface {
	client.Client

	// ReadOnly returns true when this controller
	readOnly() bool
	Namespace() string
	WithJetStreamClient(*connectionOptions, func(js jetstream.JetStream) error) error
}

func NewJSController(k8sClient client.Client, natsConfig *NatsConfig, controllerConfig *Config) (JetStreamController, error) {

	var baseClient jetstream.JetStream
	if natsConfig.ServerURL != "" {
		var err error
		baseClient, _, err = CreateJetStreamClient(natsConfig, true)
		if err != nil {
			return nil, fmt.Errorf("create base js k8sClient: %w", err)
		}
	}

	return &jsController{
		Client:           k8sClient,
		config:           natsConfig,
		controllerConfig: controllerConfig,
		baseClient:       baseClient,
	}, nil
}

type jsController struct {
	client.Client
	config           *NatsConfig
	controllerConfig *Config
	baseClient       jetstream.JetStream
}

func (c *jsController) readOnly() bool {
	return c.controllerConfig.ReadOnly
}

func (c *jsController) Namespace() string {
	return c.controllerConfig.Namespace
}

func (c *jsController) WithJetStreamClient(opts *connectionOptions, op func(js jetstream.JetStream) error) error {
	if c.baseClient == nil {
		return errors.New("no client available")
	}

	// Use base client when no config is given
	if opts == nil || opts.empty() {
		return op(c.baseClient)
	}

	// Build single use client
	serverUrl := strings.Join(opts.Servers, ",")

	// TODO needs review:
	// Here the config from spec takes precedence over base config on a value by value basis.
	// This could lead to issues when an NKey is set in the spec config and Credentials are set in the base config.
	// In NatsConfig.buildOptions, Credentials take precedence over the Nkey,
	// which would lead to the nkey from spec to be ignored.
	cfg := &NatsConfig{
		CRDConnect:  false,
		ClientName:  c.config.ClientName,
		Credentials: or(opts.Creds, c.config.Credentials),
		NKey:        or(opts.Nkey, c.config.NKey),
		ServerURL:   or(serverUrl, c.config.ServerURL),
		CAs:         *or(&opts.TLS.RootCAs, &c.config.CAs),
		Certificate: or(opts.TLS.ClientCert, c.config.Certificate),
		Key:         or(opts.TLS.ClientKey, c.config.Key),
		TLSFirst:    false,
	}

	client, closer, err := CreateJetStreamClient(cfg, true)
	if err != nil {
		return fmt.Errorf("create jetstream client: %w", err)
	}
	defer closer.Close()

	return op(client)
}

// or returns the value if it is not the null value. Otherwise, the fallback value is returned
func or[T comparable](v T, fallback T) T {
	if v == *new(T) {
		return fallback
	}
	return v
}

// updateReadyCondition returns the conditions with an added or updated ready condition
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
