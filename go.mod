module github.com/nats-io/nack

go 1.15

require (
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-logr/logr v0.2.1 // indirect
	github.com/google/go-cmp v0.5.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/nats-io/jsm.go v0.0.20
	github.com/nats-io/jwt v1.0.1 // indirect
	github.com/nats-io/nats.go v1.10.1-0.20201111151633-9e1f4a0d80d8
	github.com/sirupsen/logrus v1.6.0 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/net v0.0.0-20200904194848-62affa334b73 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/sys v0.0.0-20200918174421-af09f7315aff // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	k8s.io/api v0.19.2
	k8s.io/apiextensions-apiserver v0.19.1 // indirect
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/code-generator v0.19.1
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.3.0
	k8s.io/kube-openapi v0.0.0-20200831175022-64514a1d5d59 // indirect
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800 // indirect
)

// Taken from kubernetes/sample-controller@369f5297d99f2baa95923c117ffb5a3dc32e569c
replace (
	k8s.io/api => k8s.io/api v0.0.0-20200902051604-73d7eb3bb026
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20200902131538-ba0f2f062330
	k8s.io/client-go => k8s.io/client-go v0.0.0-20200902132332-b643ec487eb7
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20200813011144-5a311e69ffcf
)
