package jetstream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	jsmapi "github.com/nats-io/jsm.go/api"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1"
	typed "github.com/nats-io/nack/pkg/jetstream/generated/clientset/versioned/typed/jetstream/v1"

	k8sapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	consumerFinalizerKey = "consumerfinalizer.jetstream.nats.io"
)

func consumerEventHandlers(ctx context.Context, q workqueue.RateLimitingInterface, jif typed.JetstreamV1Interface) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			consumer, ok := obj.(*apis.Consumer)
			if !ok {
				return
			}

			if err := enqueueWork(q, consumer); err != nil {
				utilruntime.HandleError(err)
			}
		},
		UpdateFunc: func(prevObj, nextObj interface{}) {
			prev, ok := prevObj.(*apis.Consumer)
			if !ok {
				return
			}
			next, ok := nextObj.(*apis.Consumer)
			if !ok {
				return
			}

			if !shouldEnqueueConsumer(next, prev) {
				return
			}

			if err := enqueueWork(q, next); err != nil {
				utilruntime.HandleError(err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			consumer, ok := obj.(*apis.Consumer)
			if !ok {
				return
			}

			if err := enqueueWork(q, consumer); err != nil {
				utilruntime.HandleError(err)
			}
		},
	}
}

func (c *Controller) runConsumerQueue() {
	for {
		processQueueNext(c.consumerQueue, &realJsmClient{}, c.processConsumer)
	}
}

func (c *Controller) processConsumer(ns, name string, jsmc jsmClient) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to process consumer: %w", err)
		}
	}()

	consumer, err := c.consumerLister.Consumers(ns).Get(name)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	cif := c.ji.Consumers(ns)

	creds, err := getCreds(c.ctx, consumer.Spec.CredentialsSecret, c.ki.Secrets(ns))
	if err != nil {
		if _, serr := setConsumerErrored(c.ctx, consumer, cif, err); serr != nil {
			return fmt.Errorf("%s: %w", err, serr)
		}
		return err
	}

	c.normalEvent(consumer, "Connecting", "Connecting to NATS Server")
	err = jsmc.Connect(
		strings.Join(consumer.Spec.Servers, ","),
		getNATSOptions(c.natsName, creds)...,
	)
	if err != nil {
		if _, serr := setConsumerErrored(c.ctx, consumer, cif, err); serr != nil {
			return fmt.Errorf("%s: %w", err, serr)
		}
		return err
	}
	defer jsmc.Close()
	c.normalEvent(consumer, "Connected", "Connected to NATS Server")

	deleteOK := consumer.GetDeletionTimestamp() != nil
	newGeneration := consumer.Generation != consumer.Status.ObservedGeneration
	consumerOK, err := consumerExists(c.ctx, jsmc, consumer.Spec.StreamName, consumer.Spec.DurableName)
	if err != nil {
		if _, serr := setConsumerErrored(c.ctx, consumer, cif, err); serr != nil {
			return fmt.Errorf("%s: %w", err, serr)
		}
		return err
	}
	updateOK := (consumerOK && !deleteOK && newGeneration)
	createOK := (!consumerOK && !deleteOK && newGeneration)

	consumerName, streamName := consumer.Spec.DurableName, consumer.Spec.StreamName
	switch {
	case createOK:
		c.normalEvent(consumer, "Creating",
			fmt.Sprintf("Creating consumer %q on stream %q", consumerName, streamName))
		if err := createConsumer(c.ctx, jsmc, consumer); err != nil {
			if _, serr := setConsumerErrored(c.ctx, consumer, cif, err); serr != nil {
				return fmt.Errorf("%s: %w", err, serr)
			}
			return err
		}

		res, err := setConsumerFinalizer(c.ctx, consumer, cif)
		if err != nil {
			if _, serr := setConsumerErrored(c.ctx, consumer, cif, err); serr != nil {
				return fmt.Errorf("%s: %w", err, serr)
			}
			return err
		}
		consumer = res

		if _, err := setConsumerStatus(c.ctx, consumer, cif); err != nil {
			return err
		}
		c.normalEvent(consumer, "Created",
			fmt.Sprintf("Created consumer %q on stream %q", consumerName, streamName))
	case updateOK:
		c.warningEvent(consumer, "Updating",
			fmt.Sprintf("Consumer updates (%q on %q) are not allowed, recreate to update", consumerName, streamName))
	case deleteOK:
		c.normalEvent(consumer, "Deleting", fmt.Sprintf("Deleting consumer %q on stream %q", consumerName, streamName))
		if err := deleteConsumer(c.ctx, jsmc, consumer.Spec.StreamName, consumer.Spec.DurableName); err != nil {
			if _, serr := setConsumerErrored(c.ctx, consumer, cif, err); serr != nil {
				return fmt.Errorf("%s: %w", err, serr)
			}
			return err
		}

		if _, err := clearConsumerFinalizer(c.ctx, consumer, cif); err != nil {
			if _, serr := setConsumerErrored(c.ctx, consumer, cif, err); serr != nil {
				return fmt.Errorf("%s: %w", err, serr)
			}
			return err
		}
	}

	return nil
}

func shouldEnqueueConsumer(prev, next *apis.Consumer) bool {
	delChanged := prev.DeletionTimestamp != next.DeletionTimestamp
	specChanged := !equality.Semantic.DeepEqual(prev.Spec, next.Spec)

	return delChanged || specChanged
}

func consumerExists(ctx context.Context, c jsmClient, streamName, consumerName string) (ok bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to check if consumer exists: %w", err)
		}
	}()

	var apierr jsmapi.ApiError
	_, err = c.LoadConsumer(ctx, streamName, consumerName)
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func createConsumer(ctx context.Context, c jsmClient, cns *apis.Consumer) (err error) {
	defer func() {
		if err != nil {
			sn, cn := cns.Spec.StreamName, cns.Spec.DurableName
			err = fmt.Errorf("failed to create consumer %q on stream %q: %w", cn, sn, err)
		}
	}()

	_, err = c.NewConsumer(ctx, cns.Spec.StreamName, jsmapi.ConsumerConfig{
		Durable: cns.Spec.DurableName,
	})
	return err
}

func deleteConsumer(ctx context.Context, c jsmClient, stream, consumer string) (err error) {
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

func setConsumerStatus(ctx context.Context, s *apis.Consumer, i typed.ConsumerInterface) (*apis.Consumer, error) {
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

func setConsumerFinalizer(ctx context.Context, s *apis.Consumer, i typed.ConsumerInterface) (*apis.Consumer, error) {
	s.SetFinalizers(addFinalizer(s.GetFinalizers(), streamFinalizerKey))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.Update(ctx, s, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set finalizers for consumer %q: %w", s.Spec.DurableName, err)
	}

	return res, nil
}

func clearConsumerFinalizer(ctx context.Context, s *apis.Consumer, i typed.ConsumerInterface) (*apis.Consumer, error) {
	s.SetFinalizers(removeFinalizer(s.GetFinalizers(), streamFinalizerKey))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.Update(ctx, s, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to clear finalizers for consumer %q: %w", s.Spec.DurableName, err)
	}

	return res, nil
}
