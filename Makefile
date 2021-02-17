export GOFLAGS := -mod=vendor
export GO111MODULE := on

now := $(shell date -u +%Y-%m-%dT%H:%M:%S%z)
gitBranch := $(shell git rev-parse --abbrev-ref HEAD)
gitCommit := $(shell git rev-parse --short HEAD)
repoDirty := $(shell git diff --quiet || echo "-dirty")

linkerVars := -X main.BuildTime=$(now) -X main.GitInfo=$(gitBranch)-$(gitCommit)$(repoDirty)

jetstreamGenOut := $(shell grep -l -R "DO NOT EDIT" pkg/jetstream/)
jetstreamGenIn:= $(shell grep -l -R -F "// +k8s:" pkg/jetstream/apis)
jetstreamSrc := $(shell find cmd/jetstream-controller pkg/jetstream controllers/jetstream -name "*.go")

configReloaderSrc := $(shell find cmd/nats-server-config-reloader/ pkg/natsreloader/ -name "*.go")

vendor: go.mod go.sum
	go mod vendor
	touch $@

$(jetstreamGenOut): vendor $(jetstreamGenIn) pkg/k8scodegen/file-header.txt
	GOFLAGS='' bash vendor/k8s.io/code-generator/generate-groups.sh all \
		github.com/nats-io/nack/pkg/jetstream/generated \
		github.com/nats-io/nack/pkg/jetstream/apis \
		"jetstream:v1beta1" \
		--go-header-file pkg/k8scodegen/file-header.txt

jetstream-controller: $(jetstreamSrc) vendor
	go build -race -o $@ \
		-ldflags "$(linkerVars)" \
		github.com/nats-io/nack/cmd/jetstream-controller

jetstream-controller.docker: $(jetstreamSrc) vendor
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@ \
		-ldflags "-s -w $(linkerVars)" \
		-tags timetzdata \
		github.com/nats-io/nack/cmd/jetstream-controller

.PHONY: jetstream-controller-docker
jetstream-controller-docker:
ifneq ($(jetstreamVersion),)
	docker build --tag natsio/jetstream-controller:$(jetstreamVersion) \
		--file docker/jetstream-controller/Dockerfile .
else
	# Missing jetstreamVersion, try again.
	# make jetstream-controller-docker jetstreamVersion=1.2.3
	exit 1
endif

nats-server-config-reloader: $(configReloaderSrc) vendor
	go build -race -o $@ \
		-ldflags "$(linkerVars)" \
		github.com/nats-io/nack/cmd/nats-server-config-reloader

nats-server-config-reloader.docker: $(configReloaderSrc) vendor
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@ \
		-ldflags "-s -w $(linkerVars)" \
		-tags timetzdata \
		github.com/nats-io/nack/cmd/nats-server-config-reloader

.PHONY: nats-server-config-reloader-docker
nats-server-config-reloader-docker:
ifneq ($(configReloaderVersion),)
	docker build --tag natsio/nats-server-config-reloader:$(configReloaderVersion) \
		--file docker/nats-server-config-reloader/Dockerfile .
else
	# Missing configReloaderVersion, try again.
	# make nats-server-config-reloader-docker configReloaderVersion=1.2.3
	exit 1
endif

.PHONY: build
build: jetstream-controller nats-server-config-reloader

.PHONY: test
test:
	go vet ./controllers/... ./pkg/natsreloader/...
	go test -race -cover -count=1 -timeout 10s ./controllers/... ./pkg/natsreloader/...

.PHONY: clean
clean:
	rm -f jetstream-controller jetstream-controller.docker \
		nats-server-config-reloader nats-server-config-reloader.docker
