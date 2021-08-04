export GOFLAGS := -mod=vendor
export GO111MODULE := on

now := $(shell date -u +%Y-%m-%dT%H:%M:%S%z)
gitBranch := $(shell git rev-parse --abbrev-ref HEAD)
gitCommit := $(shell git rev-parse --short HEAD)
repoDirty := $(shell git diff --quiet || echo "-dirty")

VERSION ?= version-not-set
linkerVars := -X main.BuildTime=$(now) -X main.GitInfo=$(gitBranch)-$(gitCommit)$(repoDirty) -X main.Version=$(VERSION)

jetstreamGenIn:= $(shell grep -l -R -F "// +k8s:" pkg/jetstream/apis)
jetstreamSrc := $(shell find cmd/jetstream-controller pkg/jetstream controllers/jetstream -name "*.go")

configReloaderSrc := $(shell find cmd/nats-server-config-reloader/ pkg/natsreloader/ -name "*.go")

bootConfigSrc := $(shell find cmd/nats-boot-config/ pkg/bootconfig/ -name "*.go")

default:
	# Try these (read Makefile for more recipes):
	#   make jetstream-controller
	#   make nats-server-config-reloader
	#   make nats-boot-config

vendor: go.mod go.sum
	go mod vendor
	touch $@

pkg/jetstream/generated pkg/jetstream/apis/jetstream/v1beta1/zz_generated.deepcopy.go: vendor $(jetstreamGenIn) pkg/k8scodegen/file-header.txt
	rm -rf pkg/jetstream/generated
	GOFLAGS='' bash vendor/k8s.io/code-generator/generate-groups.sh all \
		github.com/nats-io/nack/pkg/jetstream/generated \
		github.com/nats-io/nack/pkg/jetstream/apis \
		"jetstream:v1beta1" \
		--output-base . \
		--go-header-file pkg/k8scodegen/file-header.txt
	mv github.com/nats-io/nack/pkg/jetstream/generated pkg/jetstream/generated
	mv github.com/nats-io/nack/pkg/jetstream/apis/jetstream/v1beta1/zz_generated.deepcopy.go pkg/jetstream/apis/jetstream/v1beta1/zz_generated.deepcopy.go
	rm -rf github.com

jetstream-controller: $(jetstreamSrc) vendor
	go build -race -o $@ \
		-ldflags "$(linkerVars)" \
		github.com/nats-io/nack/cmd/jetstream-controller

jetstream-controller.docker: $(jetstreamSrc) vendor
	CGO_ENABLED=0 GOOS=linux go build -o $@ \
		-ldflags "-s -w $(linkerVars)" \
		-tags timetzdata \
		github.com/nats-io/nack/cmd/jetstream-controller

.PHONY: jetstream-controller-docker
jetstream-controller-docker:
ifneq ($(ver),)
	docker build --tag natsio/jetstream-controller:$(ver) \
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
		--tag natsio/jetstream-controller:$(ver) --tag natsio/jetstream-controller:latest \
		--build-arg VERSION=$(ver) \
		--platform linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64/v8 \
		--file docker/jetstream-controller/Dockerfile \
		--push .
else
	# Missing version, try this.
	# make jetstream-controller-dockerx ver=1.2.3
	exit 1
endif

nats-server-config-reloader: $(configReloaderSrc) vendor
	go build -race -o $@ \
		-ldflags "$(linkerVars)" \
		github.com/nats-io/nack/cmd/nats-server-config-reloader

nats-server-config-reloader.docker: $(configReloaderSrc) vendor
	CGO_ENABLED=0 GOOS=linux go build -o $@ \
		-ldflags "-s -w $(linkerVars)" \
		-tags timetzdata \
		github.com/nats-io/nack/cmd/nats-server-config-reloader

.PHONY: nats-server-config-reloader-docker
nats-server-config-reloader-docker:
ifneq ($(ver),)
	docker build --tag natsio/nats-server-config-reloader:$(ver) \
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
		--tag natsio/nats-server-config-reloader:$(ver) --tag natsio/nats-server-config-reloader:latest \
		--build-arg VERSION=$(ver) \
		--platform linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64/v8 \
		--file docker/nats-server-config-reloader/Dockerfile \
		--push .
else
	# Missing version, try this.
	# make nats-server-config-reloader-dockerx ver=1.2.3
	exit 1
endif

nats-boot-config: $(bootConfigSrc) vendor
	go build -race -o $@ \
		-ldflags "$(linkerVars)" \
		github.com/nats-io/nack/cmd/nats-boot-config

nats-boot-config.docker: $(bootConfigSrc) vendor
	CGO_ENABLED=0 GOOS=linux go build -o $@ \
		-ldflags "-s -w $(linkerVars)" \
		-tags timetzdata \
		github.com/nats-io/nack/cmd/nats-boot-config

.PHONY: nats-boot-config-docker
nats-boot-config-docker:
ifneq ($(ver),)
	docker build --tag natsio/nats-boot-config:$(ver) \
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
		--tag natsio/nats-boot-config:$(ver) --tag natsio/nats-boot-config:latest \
		--build-arg VERSION=$(ver) \
		--platform linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64/v8 \
		--file docker/nats-boot-config/Dockerfile \
		--push .
else
	# Missing version, try this.
	# make nats-boot-config-dockerx ver=1.2.3
	exit 1
endif

.PHONY: build
build: jetstream-controller nats-server-config-reloader nats-boot-config

.PHONY: test
test:
	go vet ./controllers/... ./pkg/natsreloader/...
	go test -race -cover -count=1 -timeout 10s ./controllers/... ./pkg/natsreloader/...

.PHONY: clean
clean:
	rm -f jetstream-controller jetstream-controller.docker \
		nats-server-config-reloader nats-server-config-reloader.docker \
		nats-boot-config nats-boot-config.docker
