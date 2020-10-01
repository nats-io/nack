<img width="800" alt="nack-large" src="https://user-images.githubusercontent.com/26195/92535603-71ad9a80-f1ec-11ea-8959-cdc22b31b84a.png">

NATS Controllers for Kubernetes (NACK)

## Jetstream Controller

### Usage Walkthrough

First, we'll need to install the Jetstream Controller with Helm.

```sh
# First, install with Helm.
$ helm install myjsc ./helm/jetstream-controller

# This is how you uninstall, if you need to.
# helm uninstall myjsc
```

Now we can create some Streams and Consumers.

```sh
# Create a stream.
$ kubectl apply -f examples/stream.yaml
# Check if it was successfully created.
$ kubectl get streams
NAME       STATE     STREAM NAME   SUBJECTS
mystream   Created   mystream      [orders.*]

# Create two consumers, one push-based, and the other pull-based.
$ kubectl apply -f examples/consumer_push.yaml -f examples/consumer_pull.yaml
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
$ nats pub orders.received "order 1"
$ nats pub orders.received "order 2"
```

First, we'll read the data using a pull-based consumer. In `consumer_pull.yaml`
we set `filterSubject: orders.received`, so that's the subject
`my-pull-consumer` will pull messages from.

```sh
# Pull first message.
$ nats consumer next mystream my-pull-consumer
--- subject: orders.received / delivered: 1 / stream seq: 1 / consumer seq: 1

order 1

Acknowledged message

# Pull next message.
$ nats consumer next mystream my-pull-consumer
--- subject: orders.received / delivered: 1 / stream seq: 2 / consumer seq: 2

order 2

Acknowledged message
```

Next, let's read data using a push-based consumer. In `consumer_push.yaml` we
set `deliverSubject: my-push-consumer.orders`, so pushed messages will arrive
on that subject. This time all messages arrive automatically.

```
$ nats sub my-push-consumer.orders
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
./jetstream-controller -kubeconfig ~/.kube/config
# or this.
KUBECONFIG=~/.kube/config ./jetstream-controller

# Pro tip: jetstream-controller uses klog just like kubectl or kube-apiserver.
# This means you can change the verbosity of logs with the -v flag.
#
# For example, this prints raw HTTP requests and responses.
#     ./jetstream-controller -v=10


# You can install the YAML like this
kubectl apply -f helm/jetstream-controller/crds/stream.yaml
# And uninstall like this.
kubectl delete -f helm/jetstream-controller/crds/stream.yaml


# You'll probably want to start a local Jetstream-enabled NATS server, unless
# you use a public one.
nats-server -DV -js


# Finally, create an example stream.
kubectl apply -f examples/stream.yaml
```
