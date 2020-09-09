<img width="800" alt="nack-large" src="https://user-images.githubusercontent.com/26195/92535603-71ad9a80-f1ec-11ea-8959-cdc22b31b84a.png">

NATS Controllers for Kubernetes (NACK)

## Local Development

### JetStream Controller

```sh
$ kubectl apply -f deploy/jetstream-crds.yaml 
$ kubectl apply -f deploy/example-stream.yaml

# Start NATS Server with JetStream enabled
$ nats-server -DV -js

# Start JetStream Controller
make jetstream-controller
KUBERNETES_CONFIG_FILE=~/.kube/config ./jetstream-controller

# Start leaf config Controller
make leaf-config-controller
KUBERNETES_CONFIG_FILE=~/.kube/config ./leaf-config-controller

# Build all controllers
make build
```
