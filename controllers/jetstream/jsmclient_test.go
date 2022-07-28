package jetstream

import (
	"context"

	"github.com/nats-io/jsm.go"
	jsmapi "github.com/nats-io/jsm.go/api"
	"github.com/nats-io/nats.go"
)

type mockStream struct {
	deleteErr error
}

func (m *mockStream) UpdateConfiguration(cnf jsmapi.StreamConfig, opts ...jsm.StreamOption) error {
	return nil
}

func (m *mockStream) Delete() error {
	return m.deleteErr
}

type mockConsumer struct {
	deleteErr error
}

func (m *mockConsumer) UpdateConfiguration(opts ...jsm.ConsumerOption) error {
	return nil
}

func (m *mockConsumer) Delete() error {
	return m.deleteErr
}

type mockJsmClient struct {
	connectErr error

	loadStream    jsmStream
	loadStreamErr error
	newStream     jsmStream
	newStreamErr  error

	loadConsumer    jsmConsumer
	loadConsumerErr error
	newConsumer     jsmConsumer
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

func (c *mockJsmClient) LoadConsumer(ctx context.Context, stream, consumer string) (jsmConsumer, error) {
	return c.loadConsumer, c.loadConsumerErr
}

func (c *mockJsmClient) NewConsumer(ctx context.Context, stream string, opts []jsm.ConsumerOption) (jsmConsumer, error) {
	return c.newConsumer, c.newConsumerErr
}
