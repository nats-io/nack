package controller

import (
	"errors"
	"fmt"
	js "github.com/nats-io/nack/controllers/jetstream"
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/nats-io/nats.go/jetstream"
	v1 "k8s.io/api/core/v1"

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
	// TODO Handle Account, Nkey, Servers and TLS from spec.
	// TODO Get specific client when there is connection config in the spec.

	if c.baseClient == nil {
		return errors.New("no client available")
	}

	if opts == nil || opts.empty() {
		return op(c.baseClient)
	}

	// TODO Build single use client
	return errors.New("Not Implemented")
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
