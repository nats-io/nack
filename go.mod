module github.com/nats-io/nack

go 1.16

require (
	github.com/fsnotify/fsnotify v1.5.0
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/nats-io/jsm.go v0.0.25
	github.com/nats-io/nats.go v1.11.1-0.20210623165838-4b75fc59ae30
	github.com/sirupsen/logrus v1.8.1
	k8s.io/api v0.19.1
	k8s.io/apimachinery v0.19.1
	k8s.io/client-go v0.19.1
	k8s.io/code-generator v0.19.1
	k8s.io/klog/v2 v2.9.0
)

// Look for updates here:
// https://github.com/kubernetes/sample-controller/blob/master/go.mod
replace (
	k8s.io/api => k8s.io/api v0.0.0-20210817200411-f6e49805ed5a
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20210817200207-02cfb5391634
	k8s.io/client-go => k8s.io/client-go v0.0.0-20210817200704-2961e1de2c13
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20210817200016-7edd0050705a
)
