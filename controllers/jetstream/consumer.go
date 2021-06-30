package jetstream

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/jsm.go"
	jsmapi "github.com/nats-io/jsm.go/api"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta1"
	typed "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/typed/jetstream/v1beta1"
	"github.com/nats-io/nats.go"

	k8sapi "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	consumerFinalizerKey = "consumerfinalizer.jetstream.nats.io"
)

func (c *Controller) runConsumerQueue() {
	for {
		processQueueNext(c.cnsQueue, &realJsmClient{jm: c.jm}, c.processConsumer)
	}
}

func (c *Controller) processConsumer(ns, name string, jsmc jsmClient) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to process consumer: %w", err)
		}
	}()

	cns, err := c.cnsLister.Consumers(ns).Get(name)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	spec := cns.Spec
	ifc := c.ji.Consumers(ns)

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
		if len(servers) != 0 {
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
			opts = append(opts, nats.MaxReconnects(-1))

			natsServers := strings.Join(servers, ",")
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

		res, err := setConsumerFinalizer(c.ctx, cns, ifc)
		if err != nil {
			return err
		}
		cns = res

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

		if _, err := clearConsumerFinalizer(c.ctx, cns, ifc); err != nil {
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

func setConsumerFinalizer(ctx context.Context, s *apis.Consumer, i typed.ConsumerInterface) (*apis.Consumer, error) {
	s.SetFinalizers(addFinalizer(s.GetFinalizers(), consumerFinalizerKey))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.Update(ctx, s, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set finalizers for consumer %q: %w", s.Spec.DurableName, err)
	}

	return res, nil
}

func clearConsumerFinalizer(ctx context.Context, s *apis.Consumer, i typed.ConsumerInterface) (*apis.Consumer, error) {
	s.SetFinalizers(removeFinalizer(s.GetFinalizers(), consumerFinalizerKey))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.Update(ctx, s, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to clear finalizers for consumer %q: %w", s.Spec.DurableName, err)
	}

	return res, nil
}
