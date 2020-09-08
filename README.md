<img width="1817" alt="nack-large" src="https://user-images.githubusercontent.com/26195/92535603-71ad9a80-f1ec-11ea-8959-cdc22b31b84a.png">

NATS Controllers for Kubernetes (NACK)

## Local Development

### JetStream Controller

```sh
# Start NATS Server with JetStream enabled
$ nats-server -DV -js

# Start JetStream Controller
KUBERNETES_CONFIG_FILE=~/.kube/config go run cmd/jetstream-controller/main.go
```
