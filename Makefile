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

.PHONY: build
build: jetstream-controller

.PHONY: test
test:
	go vet ./controllers/...
	go test -race -cover -count=1 -timeout 10s ./controllers/...

.PHONY: clean
clean:
	rm -f jetstream-controller jetstream-controller.docker
