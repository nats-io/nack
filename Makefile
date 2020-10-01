export GOFLAGS := -mod=vendor
export GO111MODULE := on

codeGenerator := vendor/k8s.io/code-generator/generate-groups.sh

jetstreamGenOut := pkg/jetstream/generated pkg/jetstream/apis/jetstream/v1beta1/zz_generated.deepcopy.go
jetstreamGenIn:= $(shell grep -l -R -F "// +" pkg/jetstream/apis | grep -v "zz_generated.deepcopy.go")
jetstreamSrc := $(shell find cmd/jetstream-controller pkg/jetstream controllers/jetstream -name "*.go")

jetstreamTag := connecteverything/jetstream-controller:1.0.0

now := $(shell date -u +%Y-%m-%dT%H:%M:%S%z)

gitTag := $(shell git describe --tags --abbrev=0 2>/dev/null)
gitBranch := $(shell git rev-parse --abbrev-ref HEAD)
gitCommit := $(shell git rev-parse --short HEAD)
repoDirty := $(shell git diff --quiet || echo "-dirty")

linkerVars := -X main.BuildTime=$(now) -X main.Version=$(gitBranch)-$(gitCommit)$(repoDirty)

vendor: go.mod go.sum
	go mod vendor

$(jetstreamGenOut): vendor $(codeGenerator) $(jetstreamGenIn) pkg/k8scodegen/file-header.txt
	GOFLAGS='' bash $(codeGenerator) all \
		github.com/nats-io/nack/pkg/jetstream/generated \
		github.com/nats-io/nack/pkg/jetstream/apis \
		"jetstream:v1beta1" \
		--go-header-file pkg/k8scodegen/file-header.txt
	touch pkg/jetstream/generated

jetstream-controller: $(sort $(jetstreamSrc) $(jetstreamGenOut)) vendor
	go build -race -o $@ \
		-ldflags "$(linkerVars)" \
		github.com/nats-io/nack/cmd/jetstream-controller

jetstream-controller.docker: $(sort $(jetstreamSrc) $(jetstreamGenOut)) vendor
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@ \
		-ldflags "-s -w $(linkerVars)" \
		-tags timetzdata \
		github.com/nats-io/nack/cmd/jetstream-controller

.PHONY: build
build: jetstream-controller

.PHONY: jetstream-controller-docker
jetstream-controller-docker:
	sudo docker build --tag $(jetstreamTag) \
		--file docker/jetstream-controller/Dockerfile .

.PHONY: test
test:
	go vet ./controllers/...
	go test -v -cover -timeout 10s ./controllers/...

.PHONY: clean
clean:
	git clean -x -d -f
