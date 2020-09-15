<img width="800" alt="nack-large" src="https://user-images.githubusercontent.com/26195/92535603-71ad9a80-f1ec-11ea-8959-cdc22b31b84a.png">

NATS Controllers for Kubernetes (NACK)

## Local Development

### JetStream Controller

```sh
# Start nightly NATS Server with JetStream enabled.
$ nats-server -DV -js

# Start JetStream Controller
$ make jetstream-controller
$ ./jetstream-controller -kubeconfig ~/.kube/config

# Create an example stream.
$ kubectl apply -f deploy/example-stream.yaml

# Install with Helm
$ helm install myjsc ./helm/jetstream-controller/
# Uninstall with Helm
$ helm uninstall myjsc ./helm/jetstream-controller/
```
