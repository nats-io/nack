export GOFLAGS := -mod=vendor
export GO111MODULE := on

codeGenerator := vendor/k8s.io/code-generator/generate-groups.sh

jetstreamGenOut := pkg/jetstream/generated pkg/jetstream/apis/jetstream/v1/zz_generated.deepcopy.go
jetstreamGenIn:= $(shell grep -l -R -F "// +" pkg/jetstream/apis | grep -v "zz_generated.deepcopy.go")
jetstreamSrc := $(shell find cmd/jetstream-controller pkg/jetstream -name "*.go")

vendor: go.mod go.sum
	go mod vendor

$(jetstreamGenOut): $(codeGenerator) $(jetstreamGenIn) pkg/k8scodegen/file-header.txt
	GOFLAGS='' bash $< all \
		github.com/nats-io/nack/pkg/jetstream/generated \
		github.com/nats-io/nack/pkg/jetstream/apis \
		"jetstream:v1" \
		--go-header-file $(lastword $^)

jetstream-controller: $(jetstreamSrc) $(jetstreamGenOut) vendor
	go build -race -o $@ github.com/nats-io/nack/cmd/jetstream-controller

.PHONY: build
build: jetstream-controller

.PHONY: clean
clean:
	git clean -x -d -f
