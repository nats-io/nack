export GO111MODULE := on

SHELL=/usr/bin/env bash

ENVTEST_K8S_VERSION = 1.29.0

now := $(shell date -u +%Y-%m-%dT%H:%M:%S%z)
gitBranch := $(shell git rev-parse --abbrev-ref HEAD)
gitCommit := $(shell git rev-parse --short HEAD)
repoDirty := $(shell git diff --quiet || echo "-dirty")

VERSION ?= version-not-set
linkerVars := -X main.BuildTime=$(now) -X main.GitInfo=$(gitBranch)-$(gitCommit)$(repoDirty) -X main.Version=$(VERSION)
drepo ?= natsio

jetstreamSrc := $(shell find cmd/jetstream-controller pkg/jetstream controllers/jetstream -name "*.go") pkg/jetstream/apis/jetstream/v1beta2/zz_generated.deepcopy.go

configReloaderSrc := $(shell find cmd/nats-server-config-reloader/ pkg/natsreloader/ -name "*.go")

bootConfigSrc := $(shell find cmd/nats-boot-config/ pkg/bootconfig/ -name "*.go")

# You might override this so as to use a more recent version, to update old
# generated imports, and so migrate away from old import paths and get back to
# a consistent import tree.
codeGeneratorDir ?=

default:
	# Try these (read Makefile for more recipes):
	#   make jetstream-controller
	#   make nats-server-config-reloader
	#   make nats-boot-config

generate: fetch-modules pkg/k8scodegen/file-header.txt
	rm -rf pkg/jetstream/generated
	D="$(codeGeneratorDir)"; : "$${D:=`go list -m -f '{{.Dir}}' k8s.io/code-generator`}"; \
	source "$$D/kube_codegen.sh" ; \
	kube::codegen::gen_helpers \
	  --boilerplate pkg/k8scodegen/file-header.txt \
	  pkg/jetstream/apis; \
	kube::codegen::gen_client \
		--with-watch \
		--with-applyconfig \
		--boilerplate pkg/k8scodegen/file-header.txt \
		--output-dir pkg/jetstream/generated \
		--output-pkg github.com/nats-io/nack/pkg/jetstream/generated \
		--one-input-api jetstream/v1beta2 \
		pkg/jetstream/apis

jetstream-controller: $(jetstreamSrc)
	go build -race -o $@ \
		-ldflags "$(linkerVars)" \
		github.com/nats-io/nack/cmd/jetstream-controller

jetstream-controller.docker: $(jetstreamSrc)
	CGO_ENABLED=0 GOOS=linux go build -o $@ \
		-ldflags "-s -w $(linkerVars)" \
		-tags timetzdata \
		github.com/nats-io/nack/cmd/jetstream-controller

.PHONY: jetstream-controller-docker
jetstream-controller-docker:
ifneq ($(ver),)
	docker build --tag $(drepo)/jetstream-controller:$(ver) \
		--build-arg VERSION=$(ver) \
		--file docker/jetstream-controller/Dockerfile .
else
	# Missing version, try this.
	# make jetstream-controller-docker ver=1.2.3
	exit 1
endif

.PHONY: jetstream-controller-dockerx
jetstream-controller-dockerx:
ifneq ($(ver),)
	# Ensure 'docker buildx ls' shows correct platforms.
	docker buildx build \
		--tag $(drepo)/jetstream-controller:$(ver) --tag $(drepo)/jetstream-controller:latest \
		--build-arg VERSION=$(ver) \
		--platform linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64/v8 \
		--file docker/jetstream-controller/Dockerfile \
		--push .
else
	# Missing version, try this.
	# make jetstream-controller-dockerx ver=1.2.3
	exit 1
endif

nats-server-config-reloader: $(configReloaderSrc)
	go build -race -o $@ \
		-ldflags "$(linkerVars)" \
		github.com/nats-io/nack/cmd/nats-server-config-reloader

nats-server-config-reloader.docker: $(configReloaderSrc)
	CGO_ENABLED=0 GOOS=linux go build -o $@ \
		-ldflags "-s -w $(linkerVars)" \
		-tags timetzdata \
		github.com/nats-io/nack/cmd/nats-server-config-reloader

.PHONY: nats-server-config-reloader-docker
nats-server-config-reloader-docker:
ifneq ($(ver),)
	docker build --tag $(drepo)/nats-server-config-reloader:$(ver) \
		--build-arg VERSION=$(ver) \
		--file docker/nats-server-config-reloader/Dockerfile .
else
	# Missing version, try this.
	# make nats-server-config-reloader-docker ver=1.2.3
	exit 1
endif

.PHONY: nats-server-config-reloader-dockerx
nats-server-config-reloader-dockerx:
ifneq ($(ver),)
	# Ensure 'docker buildx ls' shows correct platforms.
	docker buildx build \
		--tag $(drepo)/nats-server-config-reloader:$(ver) --tag $(drepo)/nats-server-config-reloader:latest \
		--build-arg VERSION=$(ver) \
		--platform linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64/v8 \
		--file docker/nats-server-config-reloader/Dockerfile \
		--push .
else
	# Missing version, try this.
	# make nats-server-config-reloader-dockerx ver=1.2.3
	exit 1
endif

nats-boot-config: $(bootConfigSrc)
	go build -race -o $@ \
		-ldflags "$(linkerVars)" \
		github.com/nats-io/nack/cmd/nats-boot-config

nats-boot-config.docker: $(bootConfigSrc)
	CGO_ENABLED=0 GOOS=linux go build -o $@ \
		-ldflags "-s -w $(linkerVars)" \
		-tags timetzdata \
		github.com/nats-io/nack/cmd/nats-boot-config

.PHONY: nats-boot-config-docker
nats-boot-config-docker:
ifneq ($(ver),)
	docker build --tag $(drepo)/nats-boot-config:$(ver) \
		--build-arg VERSION=$(ver) \
		--file docker/nats-boot-config/Dockerfile .
else
	# Missing version, try this.
	# make nats-boot-config-docker ver=1.2.3
	exit 1
endif

.PHONY: nats-boot-config-dockerx
nats-boot-config-dockerx:
ifneq ($(ver),)
	# Ensure 'docker buildx ls' shows correct platforms.
	docker buildx build \
		--tag $(drepo)/nats-boot-config:$(ver) --tag $(drepo)/nats-boot-config:latest \
		--build-arg VERSION=$(ver) \
		--platform linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64/v8 \
		--file docker/nats-boot-config/Dockerfile \
		--push .
else
	# Missing version, try this.
	# make nats-boot-config-dockerx ver=1.2.3
	exit 1
endif

.PHONY: fetch-modules
# This will error if we have removed some code to be regenerated, so we instead silence it and force success
fetch-modules:
	go list -f '{{with .Module}}{{end}}' all >/dev/null 2>&1 || true

.PHONY: build
build: jetstream-controller nats-server-config-reloader nats-boot-config

# Setup envtest tools based on a operator-sdk project makefile
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
}
endef

ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
ENVTEST_VERSION ?= release-0.17

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))


.PHONY: test
test: envtest
	go vet ./controllers/... ./pkg/natsreloader/... ./internal/controller/...
	go test -race -cover -count=1 -timeout 10s ./controllers/... ./pkg/natsreloader/... ./internal/controller/...

.PHONY: clean
clean:
	rm -f jetstream-controller jetstream-controller.docker \
		nats-server-config-reloader nats-server-config-reloader.docker \
		nats-boot-config nats-boot-config.docker

tools/minikube:
	mkdir -p $(dir $@)
	curl -L -o $@ https://storage.googleapis.com/minikube/releases/v1.25.1/minikube-linux-amd64
	chmod u+x $@

tools/kubeconfig.yaml:
	mkdir -p $(dir $@)
	touch $@
	chmod 600 $@

.PHONY: minikube-start
minikube-start: kver ?= 1.22.2
minikube-start: tools/kubeconfig.yaml tools/minikube
	KUBECONFIG=$(word 1,$^) $(word 2,$^) start --vm-driver=docker --kubernetes-version=v$(kver) \
		--extra-config=apiserver.service-account-signing-key-file=/var/lib/minikube/certs/sa.key \
		--extra-config=apiserver.service-account-key-file=/var/lib/minikube/certs/sa.pub \
		--extra-config=apiserver.service-account-issuer=api \
		--extra-config=apiserver.service-account-api-audiences=api

.PHONY: minikube-stop
minikube-stop: tools/kubeconfig.yaml tools/minikube
	KUBECONFIG=$(word 1,$^) $(word 2,$^) stop

.PHONY: minikube-delete
minikube-delete: tools/kubeconfig.yaml tools/minikube
	KUBECONFIG=$(word 1,$^) $(word 2,$^) delete

.PHONY: minikube-status
minikube-status: tools/kubeconfig.yaml tools/minikube
	KUBECONFIG=$(word 1,$^) $(word 2,$^) status
