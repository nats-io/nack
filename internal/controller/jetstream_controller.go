package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/nats-io/jsm.go"
	js "github.com/nats-io/nack/controllers/jetstream"
	api "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	"github.com/nats-io/nats.go/jetstream"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	JSConsumerNotFoundErr uint16 = 10014
	JSStreamNotFoundErr   uint16 = 10059
)

var semVerRe = regexp.MustCompile(`\Av?([0-9]+)\.?([0-9]+)?\.?([0-9]+)?`)

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

	// WithJSMClient provides a jsm.go client to the given operation.
	WithJSMClient(opts api.ConnectionOpts, ns string, op func(jsm *jsm.Manager) error) error

	RequeueInterval() time.Duration
}

func NewJSController(k8sClient client.Client, natsConfig *NatsConfig, controllerConfig *Config) (JetStreamController, error) {
	return &jsController{
		Client:           k8sClient,
		config:           natsConfig,
		controllerConfig: controllerConfig,
		cacheDir:         controllerConfig.CacheDir,
		connPool:         newConnPool(time.Second * 15),
	}, nil
}

type jsController struct {
	client.Client
	config           *NatsConfig
	controllerConfig *Config
	cacheDir         string
	cacheLock        sync.Mutex
	connPool         *connectionPool
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

func (c *jsController) WithJSMClient(opts api.ConnectionOpts, ns string, op func(js *jsm.Manager) error) error {
	cfg, err := c.natsConfigFromOpts(opts, ns)
	if err != nil {
		return err
	}

	conn, err := c.connPool.Get(cfg, true)
	if err != nil {
		return err
	}

	jsmClient, err := CreateJSMClient(conn, true, cfg.JsDomain)
	if err != nil {
		return fmt.Errorf("create jsm client: %w", err)
	}
	defer conn.Close()

	return op(jsmClient)
}

func (c *jsController) WithJetStreamClient(opts api.ConnectionOpts, ns string, op func(js jetstream.JetStream) error) error {
	cfg, err := c.natsConfigFromOpts(opts, ns)
	if err != nil {
		return err
	}

	conn, err := c.connPool.Get(cfg, true)
	if err != nil {
		return err
	}

	jsClient, err := CreateJetStreamClient(conn, true, cfg.JsDomain)
	if err != nil {
		return fmt.Errorf("create jetstream client: %w", err)
	}
	defer conn.Close()

	return op(jsClient)
}

// Setup default options, override from account resource and CRD options if configured
func (c *jsController) natsConfigFromOpts(opts api.ConnectionOpts, ns string) (*NatsConfig, error) {
	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()

	natsConfig := &NatsConfig{}
	natsConfig.Overlay(c.config)

	if opts.Account == "" {
		natsConfig.Overlay(natsConfigFromOpts(opts))
		return natsConfig, nil
	}

	// Apply Account options first, over global.
	// Apply remaining CRD options last

	accountOverlay := &NatsConfig{}

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
		accountOverlay.ServerURL = strings.Join(account.Spec.Servers, ",")
	}

	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

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

		var certData, keyData []byte
		var certPath, keyPath string

		for k, v := range tlsSecret.Data {
			switch k {
			case account.Spec.TLS.ClientCert:
				certPath = filepath.Join(accDir, k)
				certData = v
			case account.Spec.TLS.ClientKey:
				keyPath = filepath.Join(accDir, k)
				keyData = v
			case account.Spec.TLS.RootCAs:
				rootCAPath := filepath.Join(accDir, k)
				accountOverlay.CAs = append(accountOverlay.CAs, rootCAPath)

				if _, err := os.Stat(rootCAPath); err == nil {
					caBytes, err := os.ReadFile(rootCAPath)
					// Skip file write if data is unchanged
					if err == nil && bytes.Equal(caBytes, v) {
						continue
					}
				}

				if err := os.WriteFile(rootCAPath, v, 0o644); err != nil {
					return nil, err
				}
			}
		}

		if certData != nil && keyData != nil {
			accountOverlay.Certificate = certPath
			accountOverlay.Key = keyPath

			writeCert := true
			if _, err := os.Stat(certPath); err == nil {
				fileBytes, err := os.ReadFile(certPath)
				// Skip disk write if data is unchanged
				if err == nil && bytes.Equal(fileBytes, certData) {
					writeCert = false
				}
			}

			if writeCert {
				if err := os.WriteFile(certPath, certData, 0o644); err != nil {
					return nil, err
				}
			}

			writeKey := true
			if _, err := os.Stat(keyPath); err == nil {
				fileBytes, err := os.ReadFile(keyPath)
				// Skip disk write if data is unchanged
				if err == nil && bytes.Equal(fileBytes, keyData) {
					writeKey = false
				}
			}

			if writeKey {
				if err := os.WriteFile(keyPath, keyData, 0o600); err != nil {
					return nil, err
				}
			}
		}
	} else if account.Spec.TLS != nil {
		if account.Spec.TLS.ClientCert != "" && account.Spec.TLS.ClientKey != "" {
			accountOverlay.Certificate = account.Spec.TLS.ClientCert
			accountOverlay.Key = account.Spec.TLS.ClientKey
		}
		accountOverlay.CAs = []string{account.Spec.TLS.RootCAs}
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

		if credsBytes, ok := credsSecret.Data[account.Spec.Creds.File]; ok {
			filePath := filepath.Join(accDir, account.Spec.Creds.File)
			accountOverlay.Credentials = filePath

			writeCreds := true
			if _, err := os.Stat(filePath); err == nil {
				fileBytes, err := os.ReadFile(filePath)
				// Skip disk write if data is unchanged
				if err == nil && bytes.Equal(fileBytes, credsBytes) {
					writeCreds = false
				}
			}

			if writeCreds {
				if err := os.WriteFile(filePath, credsBytes, 0o600); err != nil {
					return nil, err
				}
			}
		}
	} else if account.Spec.Creds != nil {
		accountOverlay.Credentials = account.Spec.Creds.File
	}

	if account.Spec.User != nil {
		userSecret := &v1.Secret{}
		err := c.Get(ctx,
			types.NamespacedName{
				Name:      account.Spec.User.Secret.Name,
				Namespace: ns,
			},
			userSecret,
		)
		if err != nil {
			return nil, err
		}

		userName := userSecret.Data[account.Spec.User.User]
		userPassword := userSecret.Data[account.Spec.User.Password]

		if userName != nil && userPassword != nil {
			accountOverlay.User = string(userName)
			accountOverlay.Password = string(userPassword)
		}
	}

	if account.Spec.Token != nil {
		tokenSecret := &v1.Secret{}
		err := c.Get(ctx,
			types.NamespacedName{
				Name:      account.Spec.Token.Secret.Name,
				Namespace: ns,
			},
			tokenSecret,
		)
		if err != nil {
			return nil, err
		}

		if token := tokenSecret.Data[account.Spec.Token.Token]; token != nil {
			accountOverlay.Token = string(token)
		}
	}

	// Overlay Account Config
	natsConfig.Overlay(accountOverlay)

	// Overlay Spec Config
	natsConfig.Overlay(natsConfigFromOpts(opts))

	return natsConfig, nil
}

func natsConfigFromOpts(opts api.ConnectionOpts) *NatsConfig {
	natsConfig := &NatsConfig{}

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

	if opts.Nkey != "" {
		natsConfig.NKey = opts.Nkey
	}

	if len(opts.TLS.RootCAs) > 0 {
		natsConfig.CAs = opts.TLS.RootCAs
	}

	if opts.TLS.ClientCert != "" && opts.TLS.ClientKey != "" {
		natsConfig.Certificate = opts.TLS.ClientCert
		natsConfig.Key = opts.TLS.ClientKey
	}

	if opts.JsDomain != "" {
		natsConfig.JsDomain = opts.JsDomain
	}

	return natsConfig
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
	return cmp.Diff(desired, actual)
}

func versionComponents(version string) (major, minor, patch int, err error) {
	m := semVerRe.FindStringSubmatch(version)
	if m == nil {
		return 0, 0, 0, errors.New("invalid semver")
	}
	major, err = strconv.Atoi(m[1])
	if err != nil {
		return -1, -1, -1, err
	}
	minor, err = strconv.Atoi(m[2])
	if err != nil {
		return -1, -1, -1, err
	}
	patch, err = strconv.Atoi(m[3])
	if err != nil {
		return -1, -1, -1, err
	}
	return major, minor, patch, err
}
