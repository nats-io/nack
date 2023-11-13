package jetstream

import (
	"context"

	"github.com/nats-io/jsm.go"
	jsmapi "github.com/nats-io/jsm.go/api"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type jsmClient interface {
	Connect(servers string, opts ...nats.Option) error
	Close()

	LoadStream(ctx context.Context, name string) (jsmStream, error)
	NewStream(ctx context.Context, name string, opts []jsm.StreamOption) (jsmStream, error)

	LoadConsumer(ctx context.Context, stream, consumer string) (jsmConsumer, error)
	NewConsumer(ctx context.Context, stream string, opts []jsm.ConsumerOption) (jsmConsumer, error)
}

type jsmStream interface {
	UpdateConfiguration(cnf jsmapi.StreamConfig, opts ...jsm.StreamOption) error
	Delete() error
}

type jsmConsumer interface {
	UpdateConfiguration(opts ...jsm.ConsumerOption) error
	Delete() error
}

type realJsmClient struct {
	pooledNc *pooledNatsConn
	jm       *jsm.Manager
}

func (c *realJsmClient) Connect(servers string, opts ...nats.Option) error {
	connPool := newNatsConnPool(logrus.New(), &natsContextDefaults{URL: servers}, opts)
	pooledNc, err := connPool.Get(&natsContext{})
	if err != nil {
		return err
	}
	c.pooledNc = pooledNc

	m, err := jsm.New(pooledNc.nc)
	if err != nil {
		return err
	}
	c.jm = m

	return nil
}

func (c *realJsmClient) Close() {
	c.pooledNc.ReturnToPool()
}

func (c *realJsmClient) LoadStream(_ context.Context, name string) (jsmStream, error) {
	return c.jm.LoadStream(name)
}

func (c *realJsmClient) NewStream(_ context.Context, name string, opts []jsm.StreamOption) (jsmStream, error) {
	return c.jm.NewStream(name, opts...)
}

func (c *realJsmClient) LoadConsumer(_ context.Context, stream, consumer string) (jsmConsumer, error) {
	return c.jm.LoadConsumer(stream, consumer)
}

func (c *realJsmClient) NewConsumer(_ context.Context, stream string, opts []jsm.ConsumerOption) (jsmConsumer, error) {
	return c.jm.NewConsumer(stream, opts...)
}
