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

	"github.com/nats-io/jsm.go"
	jsmapi "github.com/nats-io/jsm.go/api"
	"github.com/nats-io/nats.go"

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

			if err := enqueueStreamWork(q, stream); err != nil {
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

			if err := validateStreamUpdate(prev, next); errors.Is(err, errNothingToUpdate) {
				return
			} else if err != nil {
				sif := jif.Streams(next.Namespace)
				if _, serr := setStreamErrored(ctx, next, sif, err); serr != nil {
					err = fmt.Errorf("%s: %w", err, serr)
				}

				utilruntime.HandleError(err)
				return
			}

			if err := enqueueStreamWork(q, next); err != nil {
				utilruntime.HandleError(err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			stream, ok := obj.(*apis.Stream)
			if !ok {
				return
			}

			if err := enqueueStreamWork(q, stream); err != nil {
				utilruntime.HandleError(err)
			}
		},
	}
}

func enqueueStreamWork(q workqueue.RateLimitingInterface, stream *apis.Stream) (err error) {
	key, err := cache.MetaNamespaceKeyFunc(stream)
	if err != nil {
		return fmt.Errorf("failed to queue stream work: %w", err)
	}

	q.Add(key)
	return nil
}

func validateStreamUpdate(prev, next *apis.Stream) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to validate update: %w", err)
		}
	}()

	if prev.DeletionTimestamp != next.DeletionTimestamp {
		return nil
	}

	if prev.Spec.Name != next.Spec.Name {
		return fmt.Errorf("updating stream name is not allowed, please recreate")
	}
	if prev.Spec.Storage != next.Spec.Storage {
		return fmt.Errorf("updating stream storage is not allowed, please recreate")
	}

	if equality.Semantic.DeepEqual(prev.Spec, next.Spec) {
		return errNothingToUpdate
	}

	return nil
}

func (c *Controller) streamWorker() {
	for {
		item, shutdown := c.streamQueue.Get()
		if shutdown {
			return
		}

		ns, name, err := splitNamespaceName(item)
		if err != nil {
			// Probably junk, clean it up.
			utilruntime.HandleError(err)
			c.streamQueue.Forget(item)
			c.streamQueue.Done(item)
			continue
		}

		err = c.processStream(ns, name)
		if err == nil {
			// Item processed successfully, don't requeue.
			c.streamQueue.Forget(item)
			c.streamQueue.Done(item)
			continue
		}

		utilruntime.HandleError(err)

		if c.streamQueue.NumRequeues(item) < 10 {
			// Failed to process item, try again.
			c.streamQueue.AddRateLimited(item)
			c.streamQueue.Done(item)
			continue
		}

		// If we haven't been able to recover by this point, then just stop.
		// The user should have enough info in kubectl describe to debug.
		c.streamQueue.Forget(item)
		c.streamQueue.Done(item)
	}
}

func getNATSOptions(connName string) []nats.Option {
	return []nats.Option{
		nats.Name(connName),
		nats.Option(func(o *nats.Options) error {
			o.Pedantic = true
			return nil

		}),
	}
}

func (c *Controller) processStream(ns, name string) (err error) {
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

	c.normalEvent(stream, "Processing", fmt.Sprintf("Processing stream %q", stream.Spec.Name))

	nc, err := nats.Connect(
		strings.Join(stream.Spec.Servers, ","),
		getNATSOptions(c.natsName)...,
	)
	if err != nil {
		return err
	}
	defer nc.Close()

	sif := c.ji.Streams(stream.Namespace)

	deleteOK := stream.GetDeletionTimestamp() != nil
	newGeneration := stream.Generation != stream.Status.ObservedGeneration
	streamExists, err := natsStreamExists(c.ctx, stream.Spec.Name, nc)
	if err != nil {
		if _, serr := setStreamErrored(c.ctx, stream, sif, err); serr != nil {
			return fmt.Errorf("%s: %w", err, serr)
		}
		return err
	}
	updateOK := (streamExists && !deleteOK && newGeneration)
	createOK := (!streamExists && !deleteOK && newGeneration)

	switch {
	case updateOK:
		c.normalEvent(stream, "Updating", fmt.Sprintf("Updating stream %q", stream.Spec.Name))
		if err := updateStream(c.ctx, stream, nc); err != nil {
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

		_, err = setStreamSynced(c.ctx, stream, sif)
		return err
	case createOK:
		c.normalEvent(stream, "Creating", fmt.Sprintf("Creating stream %q", stream.Spec.Name))
		if err := createStream(c.ctx, stream, nc); err != nil {
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

		_, err = setStreamSynced(c.ctx, stream, sif)
		return err
	case deleteOK:
		c.normalEvent(stream, "Deleting", fmt.Sprintf("Deleting stream %q", stream.Spec.Name))
		if err := deleteStream(c.ctx, stream, nc, sif); err != nil {
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

func deleteStream(ctx context.Context, s *apis.Stream, nc *nats.Conn, sif typed.StreamInterface) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to delete stream %q: %w", s.Spec.Name, err)
		}
	}()

	var apierr jsmapi.ApiError
	js, err := jsm.LoadStream(s.Spec.Name, jsm.WithConnection(nc), jsm.WithContext(ctx))
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		return nil
	} else if err != nil {
		return err
	}

	return js.Delete()
}

func updateStream(ctx context.Context, s *apis.Stream, nc *nats.Conn) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to update stream %q: %w", s.Spec.Name, err)
		}
	}()

	js, err := jsm.LoadStream(s.Spec.Name, jsm.WithConnection(nc), jsm.WithContext(ctx))
	if err != nil {
		return err
	}

	c := js.Configuration()

	maxDur, err := time.ParseDuration(s.Spec.MaxAge)
	if err != nil {
		return err
	}
	c.MaxAge = maxDur

	return js.UpdateConfiguration(c)
}

func createStream(ctx context.Context, s *apis.Stream, nc *nats.Conn) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to create stream %q: %w", s.Spec.Name, err)
		}
	}()

	maxDur, err := time.ParseDuration(s.Spec.MaxAge)
	if err != nil {
		return err
	}

	storage, err := getStorageType(s.Spec.Storage)
	if err != nil {
		return err
	}

	opts := []jsm.StreamOption{
		jsm.StreamConnection(jsm.WithConnection(nc), jsm.WithContext(ctx)),
		jsm.Subjects(s.Spec.Subjects...),
		jsm.MaxAge(maxDur),
	}
	if storage == jsmapi.MemoryStorage {
		opts = append(opts, jsm.MemoryStorage())
	} else if storage == jsmapi.FileStorage {
		opts = append(opts, jsm.FileStorage())
	}

	_, err = jsm.NewStream(s.Spec.Name, opts...)
	return err
}

func setStreamErrored(ctx context.Context, s *apis.Stream, sif typed.StreamInterface, err error) (*apis.Stream, error) {
	if err == nil {
		return s, nil
	}

	sc := s.DeepCopy()
	sc.Status.Conditions = append(sc.Status.Conditions, apis.StreamCondition{
		Type:               "Ready",
		Status:             k8sapi.ConditionFalse,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Errored",
		Message:            err.Error(),
	})
	sc.Status.Conditions = pruneConditions(sc.Status.Conditions)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := sif.UpdateStatus(ctx, sc, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set stream errored status: %w", err)
	}

	return res, nil
}

func setStreamSynced(ctx context.Context, s *apis.Stream, i typed.StreamInterface) (*apis.Stream, error) {
	sc := s.DeepCopy()

	sc.Status.ObservedGeneration = s.Generation
	sc.Status.Conditions = append(sc.Status.Conditions, apis.StreamCondition{
		Type:               "Ready",
		Status:             k8sapi.ConditionTrue,
		LastTransitionTime: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:             "Synced",
		Message:            "Stream is synced with spec",
	})
	sc.Status.Conditions = pruneConditions(sc.Status.Conditions)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := i.UpdateStatus(ctx, sc, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set %q stream synced status: %w", s.Spec.Name, err)
	}

	return res, nil
}

func pruneConditions(cs []apis.StreamCondition) []apis.StreamCondition {
	const maxCond = 10
	if len(cs) < maxCond {
		return cs
	}

	cs = cs[len(cs)-maxCond:]
	return cs
}

func setStreamFinalizer(ctx context.Context, s *apis.Stream, sif typed.StreamInterface) (*apis.Stream, error) {
	fs := s.GetFinalizers()
	if hasFinalizerKey(fs, streamFinalizerKey) {
		return s, nil
	}
	fs = append(fs, streamFinalizerKey)
	s.SetFinalizers(fs)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := sif.Update(ctx, s, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to set %q stream finalizers: %w", s.GetName(), err)
	}

	return res, nil
}

func clearStreamFinalizer(ctx context.Context, s *apis.Stream, sif typed.StreamInterface) (*apis.Stream, error) {
	if s.GetDeletionTimestamp() == nil {
		// Already deleted.
		return s, nil
	}

	fs := s.GetFinalizers()
	if !hasFinalizerKey(fs, streamFinalizerKey) {
		return s, nil
	}
	var filtered []string
	for _, f := range fs {
		if f == streamFinalizerKey {
			continue
		}
		filtered = append(filtered, f)
	}
	s.SetFinalizers(filtered)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := sif.Update(ctx, s, k8smeta.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to clear %q stream finalizers: %w", s.GetName(), err)
	}

	return res, nil
}

func natsStreamExists(ctx context.Context, name string, nc *nats.Conn) (ok bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to check if stream exists: %w", err)
		}
	}()

	_, err = jsm.LoadStream(name, jsm.WithConnection(nc), jsm.WithContext(ctx))
	if err == nil {
		return true, nil
	}

	apierr, ok := err.(jsmapi.ApiError)
	if !ok {
		return false, err
	}

	if apierr.NotFoundError() {
		return false, nil
	}
	return false, apierr
}
