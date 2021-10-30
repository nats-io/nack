<img width="800" alt="nack-large" src="https://user-images.githubusercontent.com/26195/92535603-71ad9a80-f1ec-11ea-8959-cdc22b31b84a.png">

[![License][License-Image]][License-Url]
[![Version](https://d25lcipzij17d.cloudfront.net/badge.svg?id=go&type=5&v=0.6.0)](https://github.com/nats-io/nack/releases/tag/v0.6.0)

[License-Url]: https://www.apache.org/licenses/LICENSE-2.0
[License-Image]: https://img.shields.io/badge/License-Apache2-blue.svg

[NATS](https://nats.io) Controllers for Kubernetes (NACK)

## JetStream Controller

The JetStream controllers allows you to manage [NATS JetStream](https://github.com/nats-io/jetstream) [Streams](https://github.com/nats-io/jetstream#streams-1) and [Consumers](https://github.com/nats-io/jetstream#consumers-1) via K8S CRDs.

### Getting started

First install the JetStream CRDs:

```sh
$ kubectl apply -f https://raw.githubusercontent.com/nats-io/nack/v0.6.0/deploy/crds.yml
```

Now install with Helm:

```
helm repo add nats https://nats-io.github.io/k8s/helm/charts/
helm install nats nats/nats --set=nats.jetstream.enabled=true
helm install nack nats/nack
```

#### Creating Streams and Consumers

Let's create a a stream and a couple of consumers:

```yaml
---
apiVersion: jetstream.nats.io/v1beta2
kind: Stream
metadata:
  name: mystream
spec:
  name: mystream
  subjects: ["orders.*"]
  storage: memory
  maxAge: 1h
---
apiVersion: jetstream.nats.io/v1beta2
kind: Consumer
metadata:
  name: my-push-consumer
spec:
  streamName: mystream
  durableName: my-push-consumer
  deliverSubject: my-push-consumer.orders
  deliverPolicy: last
  ackPolicy: none
  replayPolicy: instant
---
apiVersion: jetstream.nats.io/v1beta2
kind: Consumer
metadata:
  name: my-pull-consumer
spec:
  streamName: mystream
  durableName: my-pull-consumer
  deliverPolicy: all
  filterSubject: orders.received
  maxDeliver: 20
  ackPolicy: explicit
```

```sh
# Create a stream.
$ kubectl apply -f https://raw.githubusercontent.com/nats-io/nack/main/deploy/examples/stream.yml

# Check if it was successfully created.
$ kubectl get streams
NAME       STATE     STREAM NAME   SUBJECTS
mystream   Created   mystream      [orders.*]

# Create a push-based consumer
$ kubectl apply -f https://raw.githubusercontent.com/nats-io/nack/main/deploy/examples/consumer_push.yml

# Create a pull based consumer
$ kubectl apply -f https://raw.githubusercontent.com/nats-io/nack/main/deploy/examples/consumer_pull.yml

# Check if they were successfully created.
$ kubectl get consumers
NAME               STATE     STREAM     CONSUMER           ACK POLICY
my-pull-consumer   Created   mystream   my-pull-consumer   explicit
my-push-consumer   Created   mystream   my-push-consumer   none

# If you end up in an Errored state, run kubectl describe for more info.
#     kubectl describe streams mystream
#     kubectl describe consumers my-pull-consumer
```

Now we're ready to use Streams and Consumers. Let's start off with writing some
data into `mystream`.

```sh
# Run nats-box that includes the NATS management utilities, and exec into it.
$ kubectl apply -f https://nats-io.github.io/k8s/tools/nats-box.yml
$ kubectl exec -it nats-box -- /bin/sh -l

# Publish a couple of messages from nats-box
nats-box:~$ nats context save jetstream -s nats://nats:4222
nats-box:~$ nats context select jetstream

nats-box:~$ nats pub orders.received "order 1"
nats-box:~$ nats pub orders.received "order 2"
```

First, we'll read the data using a pull-based consumer.

From the above `my-pull-consumer` Consumer CRD, we have set the filterSubject
of `orders.received`. You can double check with the following command:

```sh
$ kubectl get consumer my-pull-consumer -o jsonpath={.spec.filterSubject}
orders.received
```

So that's the subject my-pull-consumer will pull messages from.

```sh
# Pull first message.
nats-box:~$ nats consumer next mystream my-pull-consumer
--- subject: orders.received / delivered: 1 / stream seq: 1 / consumer seq: 1

order 1

Acknowledged message

# Pull next message.
nats-box:~$ nats consumer next mystream my-pull-consumer
--- subject: orders.received / delivered: 1 / stream seq: 2 / consumer seq: 2

order 2

Acknowledged message
```

Next, let's read data using a push-based consumer.

From the above `my-push-consumer` Consumer CRD, we have set the deliverSubject
of `my-push-consumer.orders`, as you can confirm with the following command:

```sh
$ kubectl get consumer my-push-consumer -o jsonpath={.spec.deliverSubject}
my-push-consumer.orders
```

So pushed messages will arrive on that subject. This time all messages arrive automatically.

```sh
nats-box:~$ nats sub my-push-consumer.orders
17:57:24 Subscribing on my-push-consumer.orders
[#1] Received JetStream message: consumer: mystream > my-push-consumer / subject: orders.received /
delivered: 1 / consumer seq: 1 / stream seq: 1 / ack: false
order 1

[#2] Received JetStream message: consumer: mystream > my-push-consumer / subject: orders.received /
delivered: 1 / consumer seq: 2 / stream seq: 2 / ack: false
order 2
```

### Local Development

```sh
# First, build the jetstream controller.
make jetstream-controller

# Next, run the controller like this
./jetstream-controller -kubeconfig ~/.kube/config -s nats://localhost:4222

# Pro tip: jetstream-controller uses klog just like kubectl or kube-apiserver.
# This means you can change the verbosity of logs with the -v flag.
#
# For example, this prints raw HTTP requests and responses.
#     ./jetstream-controller -v=10

# You'll probably want to start a local Jetstream-enabled NATS server, unless
# you use a public one.
nats-server -DV -js
```

Build Docker image
```sh
make jetstream-controller-docker ver=1.2.3
```

## NATS Server Config Reloader

This is a sidecar that you can use to automatically reload your NATS Server
configuration file.

### Installing with Helm

For more information see the
[Chart repo](https://github.com/nats-io/k8s/tree/master/helm/charts/nats).

```
helm repo add nats https://nats-io.github.io/k8s/helm/charts/
helm install my-nats nats/nats
```

### Configuring

```yaml
reloader:
  enabled: true
  image: natsio/nats-server-config-reloader:0.6.0
  pullPolicy: IfNotPresent
```

### Local Development

```sh
# First, build the config reloader.
make nats-server-config-reloader

# Next, run the reloader like this
./nats-server-config-reloader
```

Build Docker image
```sh
make nats-server-config-reloader-docker ver=1.2.3
```

## NATS Boot Config

### Installing with Helm

For more information see the
[Chart repo](https://github.com/nats-io/k8s/tree/master/helm/charts/nats).

```
helm repo add nats https://nats-io.github.io/k8s/helm/charts/
helm install my-nats nats/nats
```

### Configuring

```yaml
bootconfig:
  image: natsio/nats-boot-config:0.5.2
  pullPolicy: IfNotPresent
```

### Local Development

```sh
# First, build the project.
make nats-boot-config

# Next, run the project like this
./nats-boot-config
```

Build Docker image
```sh
make nats-boot-config-docker ver=1.2.3
```
