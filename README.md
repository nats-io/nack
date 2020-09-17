<img width="800" alt="nack-large" src="https://user-images.githubusercontent.com/26195/92535603-71ad9a80-f1ec-11ea-8959-cdc22b31b84a.png">

NATS Controllers for Kubernetes (NACK)

## Jetstream Controller

### Normal usage

For normal usage, you can install with Helm.

```
# Install
helm install myjsc ./helm/jetstream-controller

# Uninstall
helm uninstall myjsc
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
