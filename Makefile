export GOFLAGS := -mod=vendor
export GO111MODULE := on

codeGenerator := vendor/k8s.io/code-generator/generate-groups.sh

jetstreamGenOut := pkg/jetstream/generated pkg/jetstream/apis/jetstream/v1/zz_generated.deepcopy.go
jetstreamGenIn:= $(shell grep -l -R -F "// +" pkg/jetstream/apis | grep -v "zz_generated.deepcopy.go")
jetstreamSrc := $(shell find cmd/jetstream-controller pkg/jetstream controllers/jetstream -name "*.go")

now := $(shell date -u +%Y-%m-%dT%H:%M:%S%z)

gitTag := $(shell git describe --tags --abbrev=0 2>/dev/null)
gitBranch := $(shell git branch --show-current)
gitCommit := $(shell git rev-parse --short HEAD)
repoDirty := $(shell git diff --quiet || echo "-dirty")

vendor: go.mod go.sum
	go mod vendor

$(jetstreamGenOut): vendor $(codeGenerator) $(jetstreamGenIn) pkg/k8scodegen/file-header.txt
	GOFLAGS='' bash $(codeGenerator) all \
		github.com/nats-io/nack/pkg/jetstream/generated \
		github.com/nats-io/nack/pkg/jetstream/apis \
		"jetstream:v1" \
		--go-header-file pkg/k8scodegen/file-header.txt
	touch pkg/jetstream/generated

jetstream-controller: $(sort $(jetstreamSrc) $(jetstreamGenOut)) vendor
	go build -race -o $@ \
		-ldflags "-X main.BuildTime=$(now) -X main.Version=$(gitBranch)-$(gitCommit)$(repoDirty)" \
		github.com/nats-io/nack/cmd/jetstream-controller

.PHONY: build
build: jetstream-controller

.PHONY: test
test:
	go vet ./controllers/...
	go test -v -cover ./controllers/...

.PHONY: clean
clean:
	git clean -x -d -f
