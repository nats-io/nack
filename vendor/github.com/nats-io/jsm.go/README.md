## Overview

This is a library for managing and interacting with JetStream.

This library provides API access to all the JetStream related abilities of the `nats` CLI utility.

**NOTE** For general access to JetStream no special libraries are needed, the standard language specific NATS client can be used. These are optional helpers.

## Stability

This package is under development, while JetStream is in Preview we make no promises about the API stability of this package.

## NATS Connection

Streams, Consumers and Templates need their own NATS connection to manage their communication with the network these are set using `jsm.StreamConnection()` and `jsm.ConsumerConnection()` which allows you to set a pre configured NATS connection, a timeout and a context using `jsm.RequestOptions`.  These are also accepted by many functions allowing you to override timeouts etc 

## Streams
### Creating Streams

Before anything you have to create a stream, the basic pattern is:

```go
nc, _ := nats.Connect(servers)
stream, _ := jsm.NewStream("ORDERS", jsm.Subjects("ORDERS.*"), jsm.StreamConnection(jsm.WithConnection(nc)), jsm.MaxAge(24*365*time.Hour), jsm.FileStorage())
```

This can get quite verbose so you might have a template configuration of your own choosing to create many similar Streams.

```go
template, _ := jsm.NewStreamConfiguration(jsm.DefaultStream, jsm.StreamConnection(jsm.WithConnection(nc)), jsm.MaxAge(24 * 365 * time.Hour), jsm.FileStorage())

orders, _ := jsm.NewStreamFromDefault("ORDERS", template, jsm.StreamConnection(jsm.WithConnection(nc)), jsm.Subjects("ORDERS.*"))
archive, _ := jsm.NewStreamFromDefault("ARCHIVE", template, jsm.StreamConnection(jsm.WithConnection(nc)), jsm.Subjects("ARCHIVE"), jsm.MaxAge(5*template.MaxAge))
```

The `jsm.NewStream` uses `jsm.DefaultStream` as starting defaults.  We also have `jsm.DefaultWorkQueue` to help you with a sane starting point.

You can even copy Stream configurations this way (not content, just configuration), this creates `STAGING` using `ORDERS` config with a different set of subjects:

```go
orders, err := jsm.NewStream("ORDERS", jsm.Subjects("ORDERS.*"), jsm.StreamConnection(jsm.WithConnection(nc)), jsm.MaxAge(24*365*time.Hour), jsm.FileStorage())
staging, err := jsm.NewStreamFromDefault("STAGING", orders.Configuration(), jsm.StreamConnection(jsm.WithConnection(nc)), jsm.Subjects("STAGINGORDERS.*"))
```

### Loading references to existing streams

Once a Stream exist you can load it later:

```go
orders, err := jsm.LoadStream("ORDERS", jsm.WithConnection(nc))
```

This will fail if the stream does not exist, create and load can be combined:

```go
orders, err := jsm.LoadOrNewFromDefault("ORDERS", template, jsm.Subjects("ORDERS.*"), jsm.WithConnection(nc))
```

This will create the Stream if it doesn't exist, else load the existing one - though no effort is made to ensure the loaded one matches the desired configuration in that case.

### Associated Consumers

With a stream handle you can get lists of known Consumers using `stream.ConsumerNames()`, or create new Consumers within the stream using `stream.NewConsumer` and `stream.NewConsumerFromDefault`. Consumers can also be loaded using `stream.LoadConsumer` and you can combine load and create using `stream.LoadOrNewConsumer` and `stream.LoadOrNewConsumerFromDefault`.

These methods just proxy to the Consumer specific ones which will be discussed below. When creating new Consumer instances this way the connection information from the Stream is passed into the Consumer.

### Other actions

There are a number of other functions allowing you to purge messages, read individual messages, get statistics and access the configuration. Review the godoc for details.

## Consumers

### Creating

Above you saw that once you have a handle to a stream you can create and load consumers, you can access the consumer directly though, lets create one:

```go
consumer, err := jsm.NewConsumer("ORDERS", "NEW", jsm.ConsumerConnection(jsm.WithConnection(nc)), jsm.FilterSubject("ORDERS.received"), jsm.SampleFrequency("100"))
```

Like with Streams we have `NewConsumerFromDefault`, `LoadOrNewConsumer` and `LoadOrNewConsumerFromDefault` and we supply 2 default default configurations to help you `DefaultConsumer` and `SampledDefaultConsumer`.

When using `LoadOrNewConsumer` and `LoadOrNewConsumerFromDefault` if a durable name is given then that has to match the name supplied.

Many options exist to set starting points, durability and more - everything that you will find in the `jsm` utility, review the godoc for full details.

### Consuming

Push-based Consumers are accessed using the normal NATS subscribe dynamics, we have a few helpers:

```go
sub, err := consumer.Subscribe(func(m *nats.Msg) {
   // handle the message
})
```

We have all the usual `Subscribe`, `ChanSubscribe`, `ChanQueueSubscribe`, `SubscribeSync`, `QueueSubscribe`, `QueueSubscribeSync` and `QueueSubscribeSyncWithChan`, these just proxy to the standard go library functions of the same name but it helps you with the topic names etc.

For Pull-based Consumers we have:

```go
// 1 message
msg, err := consumer.NextMsg()

// 10 messages
msgs, err := consumer.NextMsgs(10)
```

When consuming these messages they have metadata attached that you can parse:

```go
msg, _ := consumer.NextMsg(jsm.WithTimeout(60*time.Second))
meta, _ := jsm.ParseJSMsgMetadata(msg)
```

At this point you have access to `meta.Stream`, `meta.Consumer` for the names and `meta.StreamSequence`, `meta.ConsumerSequence` to determine which exact message and `meta.Delivered` for how many times it was redelivered.

### Other Actions

There are a number of other functions to help you determine if its Pull or Push, is it Durable, Sampled and to access the full configuration.
