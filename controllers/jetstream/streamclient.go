package jetstream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/jsm.go"
	jsmapi "github.com/nats-io/jsm.go/api"
	apis "github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1"
	"github.com/nats-io/nats.go"
)

type streamClient interface {
	Connect(servers string, opts ...nats.Option) error
	Close()

	Exists(ctx context.Context, name string) (bool, error)
	Delete(ctx context.Context, name string) error
	Update(ctx context.Context, s *apis.Stream) error
	Create(ctx context.Context, s *apis.Stream) error
}

type realStreamClient struct {
	nc *nats.Conn
}

func (c *realStreamClient) Connect(servers string, opts ...nats.Option) error {
	nc, err := nats.Connect(servers, opts...)
	if err != nil {
		return err
	}
	c.nc = nc
	return nil
}

func (c *realStreamClient) Close() {
	c.nc.Close()
}

func (c *realStreamClient) Exists(ctx context.Context, name string) (ok bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to check if stream exists: %w", err)
		}
	}()

	_, err = jsm.LoadStream(name, jsm.WithConnection(c.nc), jsm.WithContext(ctx))
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

func (c *realStreamClient) Delete(ctx context.Context, name string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to delete stream %q: %w", name, err)
		}
	}()

	var apierr jsmapi.ApiError
	js, err := jsm.LoadStream(name, jsm.WithConnection(c.nc), jsm.WithContext(ctx))
	if errors.As(err, &apierr) && apierr.NotFoundError() {
		return nil
	} else if err != nil {
		return err
	}

	return js.Delete()
}

func (c *realStreamClient) Update(ctx context.Context, s *apis.Stream) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to update stream %q: %w", s.Spec.Name, err)
		}
	}()

	js, err := jsm.LoadStream(s.Spec.Name, jsm.WithConnection(c.nc), jsm.WithContext(ctx))
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

func (c *realStreamClient) Create(ctx context.Context, s *apis.Stream) (err error) {
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
		jsm.StreamConnection(jsm.WithConnection(c.nc), jsm.WithContext(ctx)),
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

type mockStreamClient struct {
	connectErr error
	existsOK   bool
	existsErr  error
	deleteErr  error
	updateErr  error
	createErr  error
}

func (c *mockStreamClient) Connect(servers string, opts ...nats.Option) error {
	return c.connectErr
}

func (c *mockStreamClient) Close() {}

func (c *mockStreamClient) Exists(ctx context.Context, name string) (bool, error) {
	return c.existsOK, c.existsErr
}

func (c *mockStreamClient) Delete(ctx context.Context, name string) error {
	return c.deleteErr
}

func (c *mockStreamClient) Update(ctx context.Context, s *apis.Stream) error {
	return c.updateErr
}

func (c *mockStreamClient) Create(ctx context.Context, s *apis.Stream) error {
	return c.createErr
}
