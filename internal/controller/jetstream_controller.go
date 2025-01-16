package controller

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
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
	WithJetStreamClient(opts api.ConnectionOpts, ns string, op func(js jetstream.JetStream) error) error

	RequeueInterval() time.Duration
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

func (c *jsController) RequeueInterval() time.Duration {
	// Stagger the requeue slightly
	interval := c.controllerConfig.RequeueInterval

	// Allow up to a 10% variance
	intervalRange := float64(interval.Nanoseconds()) * 0.1

	randomFactor := (rand.Float64() * 2) - 1.0

	return time.Duration(float64(interval.Nanoseconds()) + (intervalRange * randomFactor))
}

func (c *jsController) ReadOnly() bool {
	return c.controllerConfig.ReadOnly
}

func (c *jsController) ValidNamespace(namespace string) bool {
	ns := c.controllerConfig.Namespace
	return ns == "" || ns == namespace
}

func (c *jsController) WithJetStreamClient(opts api.ConnectionOpts, ns string, op func(js jetstream.JetStream) error) error {
	// Build single use client
	// TODO(future-feature): Use client-pool instead of single use client
	cfg, err := c.natsConfigFromOpts(opts, ns)
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

// Setup default options, override from account resource and CRD options if configured
func (c *jsController) natsConfigFromOpts(opts api.ConnectionOpts, ns string) (*NatsConfig, error) {
	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()

	natsConfig := &NatsConfig{
		ClientName:  c.config.ClientName,
		Credentials: c.config.Credentials,
		NKey:        c.config.NKey,
		ServerURL:   c.config.ServerURL,
		CAs:         c.config.CAs,
		TLSFirst:    c.config.TLSFirst,
	}

	if c.config.Certificate != "" && c.config.Key != "" {
		natsConfig.Certificate = c.config.Certificate
		natsConfig.Key = c.config.Key
	}

	if opts.Account == "" {
		return applyConnOpts(*natsConfig, opts), nil
	}

	// Apply Account options first, over global.
	// Apply remaining CRD options last

	account := &api.Account{}
	err := c.Get(ctx,
		types.NamespacedName{
			Name:      opts.Account,
			Namespace: ns,
		},
		account,
	)
	if err != nil {
		return nil, err
	}

	if len(account.Spec.Servers) > 0 {
		natsConfig.ServerURL = strings.Join(account.Spec.Servers, ",")
	}

	if account.Spec.TLS != nil && account.Spec.TLS.Secret != nil {
		tlsSecret := &v1.Secret{}
		err := c.Get(ctx,
			types.NamespacedName{
				Name:      account.Spec.TLS.Secret.Name,
				Namespace: ns,
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
				Namespace: ns,
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

	return applyConnOpts(*natsConfig, opts), nil
}

func applyConnOpts(baseConfig NatsConfig, opts api.ConnectionOpts) *NatsConfig {
	natsConfig := baseConfig
	if len(opts.Servers) > 0 {
		natsConfig.ServerURL = strings.Join(opts.Servers, ",")
	}

	// Currently, if the global TLSFirst is set, a false value in the CRD will not override
	// due to that being the bool zero value. A true value in the CRD can override a global false.
	if opts.TLSFirst {
		natsConfig.TLSFirst = opts.TLSFirst
	}

	if opts.Creds != "" {
		natsConfig.Credentials = opts.Creds
	}

	if len(opts.TLS.RootCAs) > 0 {
		natsConfig.CAs = opts.TLS.RootCAs
	}

	if opts.TLS.ClientCert != "" && opts.TLS.ClientKey != "" {
		natsConfig.Certificate = opts.TLS.ClientCert
		natsConfig.Key = opts.TLS.ClientKey
	}

	return &natsConfig
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

func compareConfigState(actual any, desired any) string {
	return cmp.Diff(actual, desired)
}
