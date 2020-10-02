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

	LoadStreamTemplate(ctx context.Context, name string) (jsmDeleter, error)
	NewStreamTemplate(ctx context.Context, cnf jsmapi.StreamTemplateConfig) (jsmDeleter, error)

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
}

func (c *realJsmClient) Connect(servers string, opts ...nats.Option) error {
	nc, err := nats.Connect(servers, opts...)
	if err != nil {
		return err
	}
	c.nc = nc
	return nil
}

func (c *realJsmClient) Close() {
	c.nc.Drain()
}

func (c *realJsmClient) LoadStream(ctx context.Context, name string) (jsmStream, error) {
	return jsm.LoadStream(name, jsm.WithConnection(c.nc), jsm.WithContext(ctx))
}

func (c *realJsmClient) NewStream(ctx context.Context, name string, opts []jsm.StreamOption) (jsmStream, error) {
	opts = append(opts, jsm.StreamConnection(jsm.WithConnection(c.nc), jsm.WithContext(ctx)))
	return jsm.NewStream(name, opts...)
}

func (c *realJsmClient) LoadStreamTemplate(ctx context.Context, name string) (jsmDeleter, error) {
	return jsm.LoadStreamTemplate(name, jsm.WithConnection(c.nc), jsm.WithContext(ctx))
}

func (c *realJsmClient) NewStreamTemplate(ctx context.Context, cnf jsmapi.StreamTemplateConfig) (jsmDeleter, error) {
	opts := []jsm.StreamOption{jsm.StreamConnection(jsm.WithConnection(c.nc), jsm.WithContext(ctx))}
	return jsm.NewStreamTemplate(cnf.Name, cnf.MaxStreams, *cnf.Config, opts...)
}

func (c *realJsmClient) LoadConsumer(ctx context.Context, stream, consumer string) (jsmDeleter, error) {
	return jsm.LoadConsumer(stream, consumer, jsm.WithConnection(c.nc), jsm.WithContext(ctx))
}

func (c *realJsmClient) NewConsumer(ctx context.Context, stream string, opts []jsm.ConsumerOption) (jsmDeleter, error) {
	opts = append(opts, jsm.ConsumerConnection(jsm.WithConnection(c.nc), jsm.WithContext(ctx)))
	return jsm.NewConsumer(stream, opts...)
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

	loadStreamTemplate    jsmDeleter
	loadStreamTemplateErr error
	newStreamTemplate     jsmDeleter
	newStreamTemplateErr  error

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

func (c *mockJsmClient) LoadStreamTemplate(ctx context.Context, name string) (jsmDeleter, error) {
	return c.loadStreamTemplate, c.loadStreamTemplateErr
}

func (c *mockJsmClient) NewStreamTemplate(ctx context.Context, cnf jsmapi.StreamTemplateConfig) (jsmDeleter, error) {
	return c.newStreamTemplate, c.newStreamTemplateErr
}

func (c *mockJsmClient) LoadConsumer(ctx context.Context, stream, consumer string) (jsmDeleter, error) {
	return c.loadConsumer, c.loadConsumerErr
}

func (c *mockJsmClient) NewConsumer(ctx context.Context, stream string, opts []jsm.ConsumerOption) (jsmDeleter, error) {
	return c.newConsumer, c.newConsumerErr
}
