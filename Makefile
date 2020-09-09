export GOFLAGS := -mod=vendor
export GO111MODULE := on

codeGenerator := vendor/k8s.io/code-generator/generate-groups.sh

jetstreamControllerGenOut := pkg/jetstreamcontroller/generated \
pkg/jetstreamcontroller/apis/jetstreamcontroller/v1alpha1/zz_generated.deepcopy.go
jetstreamControllerGenIn:= $(shell grep -l -R -F "// +" pkg/jetstreamcontroller/apis | grep -v "zz_generated.deepcopy.go")
jetstreamControllerSrc := $(shell find cmd/jetstream-controller pkg/jetstreamcontroller -name "*.go")

leafConfigControllerGenOut := pkg/leafconfigcontroller/generated \
pkg/leafconfigcontroller/apis/leafconfigcontroller/v1alpha1/zz_generated.deepcopy.go
leafConfigControllerGenIn:= $(shell grep -l -R -F "// +" pkg/leafconfigcontroller/apis | grep -v "zz_generated.deepcopy.go")
leafConfigControllerSrc := $(shell find cmd/leaf-config-controller pkg/leafconfigcontroller -name "*.go")

vendor: go.mod go.sum
	go mod vendor

$(jetstreamControllerGenOut): $(codeGenerator) $(jetstreamControllerGenIn) pkg/k8scodegen/file-header.txt
	GOFLAGS='' bash $< all \
		github.com/nats-io/nack/pkg/jetstreamcontroller/generated \
		github.com/nats-io/nack/pkg/jetstreamcontroller/apis \
		"jetstreamcontroller:v1alpha1" \
		--go-header-file "pkg/k8scodegen/file-header.txt"

jetstream-controller: $(jetstreamControllerSrc) vendor
	go build -o $@ github.com/nats-io/nack/cmd/jetstream-controller

$(leafConfigControllerGenOut): $(codeGenerator) $(leafConfigControllerGenIn) pkg/k8scodegen/file-header.txt
	GOFLAGS='' bash $< all \
		github.com/nats-io/nack/pkg/leafconfigcontroller/generated \
		github.com/nats-io/nack/pkg/leafconfigcontroller/apis \
		"leafconfigcontroller:v1alpha1" \
		--go-header-file "pkg/k8scodegen/file-header.txt"

leaf-config-controller: $(leafConfigControllerSrc) vendor
	go build -o $@ github.com/nats-io/nack/cmd/leaf-config-controller

.PHONY: build
build: jetstream-controller leaf-config-controller

.PHONY: clean
clean:
	git clean -x -d -f
