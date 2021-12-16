package jetstream

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/jsm.go"
	jsmapi "github.com/nats-io/jsm.go/api"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta2"
	typed "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/typed/jetstream/v1beta2"
	"github.com/nats-io/nats.go"

	k8sapi "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) runConsumerQueue() {
	for {
		processQueueNext(c.cnsQueue, &realJsmClient{jm: c.jm}, c.processConsumer)
	}
}

func (c *Controller) processConsumer(ns, name string, jsmc jsmClient) (err error) {
	cns, err := c.cnsLister.Consumers(ns).Get(name)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	return c.processConsumerObject(cns, jsmc)
}

func (c *Controller) processConsumerObject(cns *apis.Consumer, jsmc jsmClient) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to process consumer: %w", err)
		}
	}()

	ns := cns.Namespace
	spec := cns.Spec
	ifc := c.ji.Consumers(ns)

	var (
		remoteClientCert string
		remoteClientKey  string
		remoteRootCA     string
		accServers       []string
	)
	if spec.Account != "" && c.opts.CRDConnect {
		// Lookup the account.
		acc, err := c.accLister.Accounts(ns).Get(spec.Account)
		if err != nil {
			return err
		}

		// Lookup the TLS secrets
		if acc.Spec.TLS != nil && acc.Spec.TLS.Secret != nil {
			secretName := acc.Spec.TLS.Secret.Name
			secret, err := c.ki.Secrets(ns).Get(c.ctx, secretName, k8smeta.GetOptions{})
			if err != nil {
				return err
			}

			// Write this to the cacheDir
			accDir := filepath.Join(c.cacheDir, ns, spec.Account)
			if err := os.MkdirAll(accDir, 0755); err != nil {
				return err
			}

			remoteClientCert = filepath.Join(accDir, acc.Spec.TLS.ClientCert)
			remoteClientKey = filepath.Join(accDir, acc.Spec.TLS.ClientKey)
			remoteRootCA = filepath.Join(accDir, acc.Spec.TLS.RootCAs)
			accServers = acc.Spec.Servers

			for k, v := range secret.Data {
				if err := os.WriteFile(filepath.Join(accDir, k), v, 0644); err != nil {
					return err
				}
			}
		}
	}

	defer func() {
		if err == nil {
			return
		}

		if _, serr := setConsumerErrored(c.ctx, cns, ifc, err); serr != nil {
			err = fmt.Errorf("%s: %w", err, serr)
		}
	}()

	type operator func(ctx context.Context, c jsmClient, spec apis.ConsumerSpec) (err error)

	natsClientUtil := func(op operator) error {
		servers := spec.Servers
		if c.opts.CRDConnect {
			// Create a new client
			opts := make([]nats.Option, 0)
			opts = append(opts, nats.Name(fmt.Sprintf("%s-con-%s-%d", c.opts.NATSClientName, spec.DurableName, cns.Generation)))
			// Use JWT/NKEYS based credentials if present.
			if spec.Creds != "" {
				opts = append(opts, nats.UserCredentials(spec.Creds))
			} else if spec.Nkey != "" {
				opt, err := nats.NkeyOptionFromSeed(spec.Nkey)
				if err != nil {
					return err
				}
				opts = append(opts, opt)
			}
			if spec.TLS.ClientCert != "" && spec.TLS.ClientKey != "" {
				opts = append(opts, nats.ClientCert(spec.TLS.ClientCert, spec.TLS.ClientKey))
			}

			// Use fetched secrets for the account and server if defined.
			if remoteClientCert != "" && remoteClientKey != "" {
				opts = append(opts, nats.ClientCert(remoteClientCert, remoteClientKey))
			}
			if remoteRootCA != "" {
				opts = append(opts, nats.RootCAs(remoteRootCA))
			}

			if len(spec.TLS.RootCAs) > 0 {
				opts = append(opts, nats.RootCAs(spec.TLS.RootCAs...))
			}

			opts = append(opts, nats.MaxReconnects(-1))

			natsServers := strings.Join(append(servers, accServers...), ",")
			newNc, err := nats.Connect(natsServers, opts...)
			if err != nil {
				return fmt.Errorf("failed to connect to leaf nats(%s): %w", natsServers, err)
			}

			c.normalEvent(cns, "Connecting", "Connecting to new nats-servers")
			newJm, err := jsm.New(newNc)
			if err != nil {
				return err
			}
			newJsmc := &realJsmClient{nc: newNc, jm: newJm}

			if err := op(c.ctx, newJsmc, spec); err != nil {
				return err
			}
			newJsmc.Close()
		} else {
			if err := op(c.ctx, jsmc, spec); err != nil {
				return err
			}
		}
		return nil
	}

	deleteOK := cns.GetDeletionTimestamp() != nil
	newGeneration := cns.Generation != cns.Status.ObservedGeneration
	consumerOK := true
	err = natsClientUtil(consumerExists)
	var apierr jsmapi.ApiError
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		consumerOK = false
	} else if err != nil {
		return err
	}
	updateOK := (consumerOK && !deleteOK && newGeneration)
	createOK := (!consumerOK && !deleteOK && newGeneration)

	switch {
	case createOK:
		c.normalEvent(cns, "Creating",
			fmt.Sprintf("Creating consumer %q on stream %q", spec.DurableName, spec.StreamName))
		if err := natsClientUtil(createConsumer); err != nil {
			return err
		}

		if _, err := setConsumerOK(c.ctx, cns, ifc); err != nil {
			return err
		}
		c.normalEvent(cns, "Created",
			fmt.Sprintf("Created consumer %q on stream %q", spec.DurableName, spec.StreamName))
	case updateOK:
		c.warningEvent(cns, "Updating",
			fmt.Sprintf("Consumer updates (%q on %q) are not allowed, recreate to update", spec.DurableName, spec.StreamName))
	case deleteOK:
		c.normalEvent(cns, "Deleting", fmt.Sprintf("Deleting consumer %q on stream %q", spec.DurableName, spec.StreamName))
		if err := natsClientUtil(deleteConsumer); err != nil {
			return err
		}
	default:
		c.warningEvent(cns, "Noop", fmt.Sprintf("Nothing done for consumer %q", spec.DurableName))
	}

	return nil
}

func consumerExists(ctx context.Context, c jsmClient, spec apis.ConsumerSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to check if consumer exists: %w", err)
		}
	}()

	_, err = c.LoadConsumer(ctx, spec.StreamName, spec.DurableName)
	return err
}

func createConsumer(ctx context.Context, c jsmClient, spec apis.ConsumerSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create consumer %q on stream %q: %w", spec.DurableName, spec.StreamName, err)
		}
	}()

	opts := []jsm.ConsumerOption{
		jsm.DurableName(spec.DurableName),
		jsm.DeliverySubject(spec.DeliverSubject),
		jsm.FilterStreamBySubject(spec.FilterSubject),
		jsm.RateLimitBitsPerSecond(uint64(spec.RateLimitBps)),
		jsm.MaxAckPending(uint(spec.MaxAckPending)),
	}

	if spec.MaxDeliver != 0 {
		opts = append(opts, jsm.MaxDeliveryAttempts(spec.MaxDeliver))
	}

	switch spec.DeliverPolicy {
	case "all":
		opts = append(opts, jsm.DeliverAllAvailable())
	case "last":
		opts = append(opts, jsm.StartWithLastReceived())
	case "new":
		opts = append(opts, jsm.StartWithNextReceived())
	case "byStartSequence":
		opts = append(opts, jsm.StartAtSequence(uint64(spec.OptStartSeq)))
	case "byStartTime":
		t, err := time.Parse(spec.OptStartTime, time.RFC3339)
		if err != nil {
			return err
		}
		opts = append(opts, jsm.StartAtTime(t))
	}

	switch spec.AckPolicy {
	case "none":
		opts = append(opts, jsm.AcknowledgeNone())
	case "all":
		opts = append(opts, jsm.AcknowledgeAll())
	case "explicit":
		opts = append(opts, jsm.AcknowledgeExplicit())
	}

	if spec.AckWait != "" {
		d, err := time.ParseDuration(spec.AckWait)
		if err != nil {
			return err
		}
		opts = append(opts, jsm.AckWait(d))
	}

	switch spec.ReplayPolicy {
	case "instant":
		opts = append(opts, jsm.ReplayInstantly())
	case "original":
		opts = append(opts, jsm.ReplayAsReceived())
	}

	if spec.SampleFreq != "" {
		n, err := strconv.Atoi(spec.SampleFreq)
		if err != nil {
			return err
		}
		opts = append(opts, jsm.SamplePercent(n))
	}

	if spec.DeliverGroup != "" {
		opts = append(opts, func(o *jsmapi.ConsumerConfig) error {
			o.DeliverGroup = spec.DeliverGroup
			return nil
		})
	}

	if spec.Description != "" {
		opts = append(opts, func(o *jsmapi.ConsumerConfig) error {
			o.Description = spec.Description
			return nil
		})
	}

	if spec.FlowControl {
		opts = append(opts, func(o *jsmapi.ConsumerConfig) error {
			o.FlowControl = spec.FlowControl
			return nil
		})
	}

	if spec.HeartbeatInterval != "" {
		d, err := time.ParseDuration(spec.HeartbeatInterval)
		if err != nil {
			return err
		}
		opts = append(opts, func(o *jsmapi.ConsumerConfig) error {
			o.Heartbeat = d
			return nil
		})
	}

	_, err = c.NewConsumer(ctx, spec.StreamName, opts)
	return err
}

func deleteConsumer(ctx context.Context, c jsmClient, spec apis.ConsumerSpec) (err error) {
	stream, consumer := spec.StreamName, spec.DurableName
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to delete consumer %q on stream %q: %w", consumer, stream, err)
		}
	}()

	var apierr jsmapi.ApiError
	cn, err := c.LoadConsumer(ctx, stream, consumer)
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		return nil
	} else if err != nil {
		return err
	}

	return cn.Delete()
}

func setConsumerOK(ctx context.Context, s *apis.Consumer, i typed.ConsumerInterface) (*apis.Consumer, error) {
	sc := s.DeepCopy()

	sc.Status.ObservedGeneration = s.Generation
	sc.Status.Conditions = upsertCondition(sc.Status.Conditions, apis.Condition{
		Type:               readyCondType,
		Status:             k8sapi.ConditionTrue,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Created",
		Message:            "Consumer successfully created",
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.UpdateStatus(ctx, sc, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set consumer %q status: %w", s.Spec.DurableName, err)
	}

	return res, nil
}

func setConsumerErrored(ctx context.Context, s *apis.Consumer, sif typed.ConsumerInterface, err error) (*apis.Consumer, error) {
	if err == nil {
		return s, nil
	}

	sc := s.DeepCopy()
	sc.Status.Conditions = upsertCondition(sc.Status.Conditions, apis.Condition{
		Type:               readyCondType,
		Status:             k8sapi.ConditionFalse,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Errored",
		Message:            err.Error(),
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := sif.UpdateStatus(ctx, sc, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set consumer errored status: %w", err)
	}

	return res, nil
}
