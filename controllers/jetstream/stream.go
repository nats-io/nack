// Copyright 2020 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	streamFinalizerKey = "streamfinalizer.jetstream.nats.io"
)

func streamEventHandlers(ctx context.Context, q workqueue.RateLimitingInterface, jif typed.JetstreamV1Interface) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			stream, ok := obj.(*apis.Stream)
			if !ok {
				return
			}

			if err := enqueueWork(q, stream); err != nil {
				utilruntime.HandleError(err)
			}
		},
		UpdateFunc: func(prevObj, nextObj interface{}) {
			prev, ok := prevObj.(*apis.Stream)
			if !ok {
				return
			}
			next, ok := nextObj.(*apis.Stream)
			if !ok {
				return
			}

			if !shouldEnqueueStream(next, prev) {
				return
			}

			if err := enqueueWork(q, next); err != nil {
				utilruntime.HandleError(err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			stream, ok := obj.(*apis.Stream)
			if !ok {
				return
			}

			if err := enqueueWork(q, stream); err != nil {
				utilruntime.HandleError(err)
			}
		},
	}
}

func (c *Controller) runStreamQueue() {
	for {
		processQueueNext(c.streamQueue, &realJsmClient{}, c.processStream)
	}
}

func (c *Controller) processStream(ns, name string, jsmc jsmClient) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to process stream: %w", err)
		}
	}()

	stream, err := c.streamLister.Streams(ns).Get(name)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	sif := c.ji.Streams(stream.Namespace)

	creds, err := getCreds(c.ctx, stream.Spec.CredentialsSecret, c.ki.Secrets(ns))
	if err != nil {
		if _, serr := setStreamErrored(c.ctx, stream, sif, err); serr != nil {
			return fmt.Errorf("%s: %w", err, serr)
		}
		return err
	}

	c.normalEvent(stream, "Connecting", "Connecting to NATS Server")
	err = jsmc.Connect(
		strings.Join(stream.Spec.Servers, ","),
		getNATSOptions(c.natsName, creds)...,
	)
	if err != nil {
		if _, serr := setStreamErrored(c.ctx, stream, sif, err); serr != nil {
			return fmt.Errorf("%s: %w", err, serr)
		}
		return err
	}
	defer jsmc.Close()
	c.normalEvent(stream, "Connected", "Connected to NATS Server")

	deleteOK := stream.GetDeletionTimestamp() != nil
	newGeneration := stream.Generation != stream.Status.ObservedGeneration
	streamOK, err := streamExists(c.ctx, jsmc, stream.Spec.Name)
	if err != nil {
		if _, serr := setStreamErrored(c.ctx, stream, sif, err); serr != nil {
			return fmt.Errorf("%s: %w", err, serr)
		}
		return err
	}
	updateOK := (streamOK && !deleteOK && newGeneration)
	createOK := (!streamOK && !deleteOK && newGeneration)

	switch {
	case createOK:
		c.normalEvent(stream, "Creating", fmt.Sprintf("Creating stream %q", stream.Spec.Name))
		if err := createStream(c.ctx, jsmc, stream); err != nil {
			if _, serr := setStreamErrored(c.ctx, stream, sif, err); serr != nil {
				return fmt.Errorf("%s: %w", err, serr)
			}
			return err
		}

		res, err := setStreamFinalizer(c.ctx, stream, sif)
		if err != nil {
			if _, serr := setStreamErrored(c.ctx, stream, sif, err); serr != nil {
				return fmt.Errorf("%s: %w", err, serr)
			}
			return err
		}
		stream = res

		if _, err := setStreamStatus(c.ctx, stream, sif); err != nil {
			return err
		}
		c.normalEvent(stream, "Created", fmt.Sprintf("Created stream %q", stream.Spec.Name))
		return nil
	case updateOK:
		c.normalEvent(stream, "Updating", fmt.Sprintf("Updating stream %q", stream.Spec.Name))
		if err := updateStream(c.ctx, jsmc, stream); err != nil {
			if _, serr := setStreamErrored(c.ctx, stream, sif, err); serr != nil {
				return fmt.Errorf("%s: %w", err, serr)
			}
			return err
		}

		res, err := setStreamFinalizer(c.ctx, stream, sif)
		if err != nil {
			if _, serr := setStreamErrored(c.ctx, stream, sif, err); serr != nil {
				return fmt.Errorf("%s: %w", err, serr)
			}
			return err
		}
		stream = res

		if _, err := setStreamStatus(c.ctx, stream, sif); err != nil {
			return err
		}
		c.normalEvent(stream, "Updated", fmt.Sprintf("Updated stream %q", stream.Spec.Name))
		return nil
	case deleteOK:
		c.normalEvent(stream, "Deleting", fmt.Sprintf("Deleting stream %q", stream.Spec.Name))
		if err := deleteStream(c.ctx, jsmc, stream.Spec.Name); err != nil {
			if _, serr := setStreamErrored(c.ctx, stream, sif, err); serr != nil {
				return fmt.Errorf("%s: %w", err, serr)
			}
			return err
		}

		if _, err := clearStreamFinalizer(c.ctx, stream, sif); err != nil {
			if _, serr := setStreamErrored(c.ctx, stream, sif, err); serr != nil {
				return fmt.Errorf("%s: %w", err, serr)
			}
			return err
		}

		return nil
	}

	// default: Nothing to do.
	return nil
}

func shouldEnqueueStream(prev, next *apis.Stream) bool {
	delChanged := prev.DeletionTimestamp != next.DeletionTimestamp
	specChanged := !equality.Semantic.DeepEqual(prev.Spec, next.Spec)

	return delChanged || specChanged
}

func createStream(ctx context.Context, c jsmClient, s *apis.Stream) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create stream %q: %w", s.Spec.Name, err)
		}
	}()

	maxAge, err := time.ParseDuration(s.Spec.MaxAge)
	if err != nil {
		return err
	}

	storage, err := getStorageType(s.Spec.Storage)
	if err != nil {
		return err
	}

	_, err = c.NewStream(ctx, jsmapi.StreamConfig{
		Name:     s.Spec.Name,
		Storage:  storage,
		Subjects: s.Spec.Subjects,
		MaxAge:   maxAge,
	})
	return err
}

func updateStream(ctx context.Context, c jsmClient, s *apis.Stream) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to update stream %q: %w", s.Spec.Name, err)
		}
	}()

	js, err := c.LoadStream(ctx, s.Spec.Name)
	if err != nil {
		return err
	}

	config := js.Configuration()

	maxDur, err := time.ParseDuration(s.Spec.MaxAge)
	if err != nil {
		return err
	}
	config.MaxAge = maxDur

	return js.UpdateConfiguration(config)
}

func deleteStream(ctx context.Context, c jsmClient, name string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to delete stream %q: %w", name, err)
		}
	}()

	var apierr jsmapi.ApiError
	str, err := c.LoadStream(ctx, name)
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		return nil
	} else if err != nil {
		return err
	}

	return str.Delete()
}

func streamExists(ctx context.Context, c jsmClient, name string) (ok bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to check if stream exists: %w", err)
		}
	}()

	var apierr jsmapi.ApiError
	_, err = c.LoadStream(ctx, name)
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func setStreamErrored(ctx context.Context, s *apis.Stream, sif typed.StreamInterface, err error) (*apis.Stream, error) {
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
		return nil, fmt.Errorf("failed to set stream errored status: %w", err)
	}

	return res, nil
}

func setStreamStatus(ctx context.Context, s *apis.Stream, i typed.StreamInterface) (*apis.Stream, error) {
	sc := s.DeepCopy()

	sc.Status.ObservedGeneration = s.Generation
	sc.Status.Conditions = upsertCondition(sc.Status.Conditions, apis.Condition{
		Type:               readyCondType,
		Status:             k8sapi.ConditionTrue,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Created",
		Message:            "Stream successfully created",
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.UpdateStatus(ctx, sc, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set stream %q status: %w", s.Spec.Name, err)
	}

	return res, nil
}

func setStreamFinalizer(ctx context.Context, o *apis.Stream, i typed.StreamInterface) (*apis.Stream, error) {
	o.SetFinalizers(addFinalizer(o.GetFinalizers(), streamFinalizerKey))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.Update(ctx, o, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set %q stream finalizers: %w", o.GetName(), err)
	}

	return res, nil
}

func clearStreamFinalizer(ctx context.Context, o *apis.Stream, i typed.StreamInterface) (*apis.Stream, error) {
	o.SetFinalizers(removeFinalizer(o.GetFinalizers(), streamFinalizerKey))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.Update(ctx, o, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to clear %q stream finalizers: %w", o.GetName(), err)
	}

	return res, nil
}
