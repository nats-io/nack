package jetstream

import (
	"context"

	"github.com/nats-io/jsm.go"
	jsmapi "github.com/nats-io/jsm.go/api"
	"github.com/nats-io/nats.go"
)

type jsmClient interface {
	Connect(servers string, opts ...nats.Option) error
	Close()

	LoadStream(ctx context.Context, name string) (jsmStream, error)
	NewStream(ctx context.Context, name string, opts []jsm.StreamOption) (jsmStream, error)

	LoadConsumer(ctx context.Context, stream, consumer string) (jsmDeleter, error)
	NewConsumer(ctx context.Context, stream string, opts []jsm.ConsumerOption) (jsmDeleter, error)
}

type jsmStream interface {
	UpdateConfiguration(cnf jsmapi.StreamConfig, opts ...jsm.StreamOption) error
	Delete() error
}

type jsmDeleter interface {
	Delete() error
}

type realJsmClient struct {
	nc *nats.Conn
	jm *jsm.Manager
}

func (c *realJsmClient) Connect(servers string, opts ...nats.Option) error {
	nc, err := nats.Connect(servers, opts...)
	if err != nil {
		return err
	}
	c.nc = nc

	m, err := jsm.New(nc)
	if err != nil {
		return err
	}
	c.jm = m

	return nil
}

func (c *realJsmClient) Close() {
	_ = c.nc.Drain()
}

func (c *realJsmClient) LoadStream(_ context.Context, name string) (jsmStream, error) {
	return c.jm.LoadStream(name)
}

func (c *realJsmClient) NewStream(_ context.Context, name string, opts []jsm.StreamOption) (jsmStream, error) {
	return c.jm.NewStream(name, opts...)
}

func (c *realJsmClient) LoadConsumer(_ context.Context, stream, consumer string) (jsmDeleter, error) {
	return c.jm.LoadConsumer(stream, consumer)
}

func (c *realJsmClient) NewConsumer(_ context.Context, stream string, opts []jsm.ConsumerOption) (jsmDeleter, error) {
	return c.jm.NewConsumer(stream, opts...)
}

type mockStream struct {
	deleteErr error
}

func (m *mockStream) UpdateConfiguration(cnf jsmapi.StreamConfig, opts ...jsm.StreamOption) error {
	return nil
}

func (m *mockStream) Delete() error {
	return m.deleteErr
}

type mockDeleter struct {
	deleteErr error
}

func (m *mockDeleter) Delete() error {
	return m.deleteErr
}

type mockJsmClient struct {
	connectErr error

	loadStream    jsmStream
	loadStreamErr error
	newStream     jsmStream
	newStreamErr  error

	loadConsumer    jsmDeleter
	loadConsumerErr error
	newConsumer     jsmDeleter
	newConsumerErr  error
}

func (c *mockJsmClient) Connect(servers string, opts ...nats.Option) error {
	return c.connectErr
}

func (c *mockJsmClient) Close() {}

func (c *mockJsmClient) LoadStream(ctx context.Context, name string) (jsmStream, error) {
	return c.loadStream, c.loadStreamErr
}

func (c *mockJsmClient) NewStream(ctx context.Context, name string, opt []jsm.StreamOption) (jsmStream, error) {
	return c.newStream, c.newStreamErr
}

func (c *mockJsmClient) LoadConsumer(ctx context.Context, stream, consumer string) (jsmDeleter, error) {
	return c.loadConsumer, c.loadConsumerErr
}

func (c *mockJsmClient) NewConsumer(ctx context.Context, stream string, opts []jsm.ConsumerOption) (jsmDeleter, error) {
	return c.newConsumer, c.newConsumerErr
}
