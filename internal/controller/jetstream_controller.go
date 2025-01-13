package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	js "github.com/nats-io/nack/controllers/jetstream"
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/nats-io/nats.go/jetstream"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	WithJetStreamClient(opts api.ConnectionOpts, op func(js jetstream.JetStream) error) error
}

func NewJSController(k8sClient client.Client, natsConfig *NatsConfig, controllerConfig *Config) (JetStreamController, error) {
	return &jsController{
		Client:           k8sClient,
		config:           natsConfig,
		controllerConfig: controllerConfig,
		cacheDir:         controllerConfig.CacheDir,
	}, nil
}

type jsController struct {
	client.Client
	config           *NatsConfig
	controllerConfig *Config
	cacheDir         string
}

func (c *jsController) ReadOnly() bool {
	return c.controllerConfig.ReadOnly
}

func (c *jsController) ValidNamespace(namespace string) bool {
	ns := c.controllerConfig.Namespace
	return ns == "" || ns == namespace
}

func (c *jsController) WithJetStreamClient(opts api.ConnectionOpts, op func(js jetstream.JetStream) error) error {
	// Build single use client
	// TODO(future-feature): Use client-pool instead of single use client
	cfg, err := c.natsConfigFromOpts(opts)
	if err != nil {
		return err
	}

	jsClient, closer, err := CreateJetStreamClient(cfg, true)
	if err != nil {
		return fmt.Errorf("create jetstream client: %w", err)
	}
	defer closer.Close()

	return op(jsClient)
}

// Setup default options, override from account resource if configured
func (c *jsController) natsConfigFromOpts(opts api.ConnectionOpts) (*NatsConfig, error) {
	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()

	natsConfig := c.setupNatsConfig(opts)

	if opts.Account == "" {
		return natsConfig, nil
	}

	account := &api.Account{}
	err := c.Get(ctx,
		types.NamespacedName{
			Name:      opts.Account,
			Namespace: c.controllerConfig.Namespace,
		},
		account,
	)
	if err != nil {
		return nil, err
	}

	natsConfig.ServerURL = strings.Join(account.Spec.Servers, ",")

	if account.Spec.TLS != nil && account.Spec.TLS.Secret != nil {
		tlsSecret := &v1.Secret{}
		err := c.Get(ctx,
			types.NamespacedName{
				Name:      account.Spec.TLS.Secret.Name,
				Namespace: c.controllerConfig.Namespace,
			},
			tlsSecret,
		)
		if err != nil {
			return nil, err
		}

		accDir := filepath.Join(c.cacheDir, c.controllerConfig.Namespace, opts.Account)
		if err := os.MkdirAll(accDir, 0o755); err != nil {
			return nil, err
		}

		for k, v := range tlsSecret.Data {
			filePath := ""
			switch k {
			case account.Spec.TLS.ClientCert:
				filePath = filepath.Join(accDir, account.Spec.TLS.ClientCert)
				natsConfig.Certificate = filePath
			case account.Spec.TLS.ClientKey:
				filePath = filepath.Join(accDir, account.Spec.TLS.ClientKey)
				natsConfig.Key = filePath
			case account.Spec.TLS.RootCAs:
				filePath = filepath.Join(accDir, account.Spec.TLS.RootCAs)
				natsConfig.CAs = append(natsConfig.CAs, filePath)
			default:
				return nil, fmt.Errorf("key in TLS secret does not match any of the expected values")
			}
			if err := os.WriteFile(filePath, v, 0o600); err != nil {
				return nil, err
			}
		}
	} else if account.Spec.TLS != nil {
		natsConfig.Certificate = account.Spec.TLS.ClientCert
		natsConfig.Key = account.Spec.TLS.ClientKey
		natsConfig.CAs = []string{account.Spec.TLS.RootCAs}
	}

	if account.Spec.Creds != nil && account.Spec.Creds.Secret != nil {
		credsSecret := &v1.Secret{}
		err := c.Get(ctx,
			types.NamespacedName{
				Name:      account.Spec.Creds.Secret.Name,
				Namespace: c.controllerConfig.Namespace,
			},
			credsSecret,
		)
		if err != nil {
			return nil, err
		}

		accDir := filepath.Join(c.cacheDir, c.controllerConfig.Namespace, opts.Account)
		if err := os.MkdirAll(accDir, 0o755); err != nil {
			return nil, err
		}

		for k, v := range credsSecret.Data {
			filePath := ""
			switch k {
			case account.Spec.Creds.File:
				filePath = filepath.Join(accDir, account.Spec.Creds.File)
				natsConfig.Credentials = filePath
			default:
				return nil, fmt.Errorf("key in Creds secret does not match any of the expected values")
			}
			if err := os.WriteFile(filePath, v, 0o600); err != nil {
				return nil, err
			}
		}
	} else if account.Spec.Creds.File != "" {
		natsConfig.Credentials = account.Spec.Creds.File
	}

	return natsConfig, nil
}

// Default to global config, but always accept override from provided opts
func (c *jsController) setupNatsConfig(opts api.ConnectionOpts) *NatsConfig {
	servers := c.config.ServerURL
	if len(opts.Servers) > 0 {
		servers = strings.Join(opts.Servers, ",")
	}

	// Currently, if the global TLSFirst is set, a false value in the CRD will not override
	// due to that being the bool zero value. A true value in the CRD can override a global false.
	tlsFirst := c.config.TLSFirst
	if opts.TLSFirst {
		tlsFirst = opts.TLSFirst
	}

	creds := c.config.Credentials
	if opts.Creds != "" {
		creds = opts.Creds
	}

	rootCAs := c.config.CAs
	if len(opts.TLS.RootCAs) > 0 {
		rootCAs = opts.TLS.RootCAs
	}

	var clientCert, clientKey string
	if c.config.Certificate != "" && c.config.Key != "" {
		clientCert = c.config.Certificate
		clientKey = c.config.Key
	}
	if opts.TLS.ClientCert != "" && opts.TLS.ClientKey != "" {
		clientCert = opts.TLS.ClientCert
		clientKey = opts.TLS.ClientKey
	}

	// Takes opts values if present
	cfg := &NatsConfig{
		ClientName:  c.config.ClientName,
		ServerURL:   servers,
		TLSFirst:    tlsFirst,
		Credentials: creds,
		CAs:         rootCAs,
		Certificate: clientCert,
		Key:         clientKey,
	}

	return cfg
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

// jsonString returns the given string wrapped in " and converted to []byte.
// Helper for mapping spec config to jetStream config using UnmarshalJSON.
func jsonString(v string) []byte {
	return []byte("\"" + v + "\"")
}
