module github.com/nats-io/nack

go 1.15

require (
	github.com/nats-io/nats.go v1.10.0
	github.com/sirupsen/logrus v1.6.0
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	k8s.io/api v0.19.0 // indirect
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/code-generator v0.19.0
	k8s.io/klog v1.0.0 // indirect
	k8s.io/utils v0.0.0-20200821003339-5e75c0163111 // indirect
)

// Taken from kubernetes/sample-controller@369f5297d99f2baa95923c117ffb5a3dc32e569c
replace (
	k8s.io/api => k8s.io/api v0.0.0-20200902051604-73d7eb3bb026
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20200902131538-ba0f2f062330
	k8s.io/client-go => k8s.io/client-go v0.0.0-20200902132332-b643ec487eb7
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20200813011144-5a311e69ffcf
)
