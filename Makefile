include ./Makefile.Common

# This is the code that we want to run checklicense, staticcheck, etc.
ALL_SRC := $(shell find . -name '*.go' \
							-not -path './cmd/issuegenerator/*' \
							-not -path './cmd/mdatagen/*' \
							-not -path './internal/tools/*' \
							-not -path './examples/demo/app/*' \
							-not -path './internal/data/opentelemetry-proto-gen/*' \
							-type f | sort)

# ALL_PKGS is the list of all packages where ALL_SRC files reside.
ALL_PKGS := $(shell go list $(sort $(dir $(ALL_SRC))))

# All source code and documents. Used in spell check.
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

# ALL_MODULES includes ./* dirs (excludes . dir)
ALL_MODULES := $(shell find . -type f -name "go.mod" -exec dirname {} \; | sort | egrep  '^./' )

CMD?=
TOOLS_MOD_DIR := ./internal/tools

GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
# BUILD_TYPE should be one of (dev, release).
BUILD_TYPE?=release

GIT_SHA=$(shell git rev-parse --short HEAD)
BUILD_INFO_IMPORT_PATH=go.opentelemetry.io/collector/internal/version
BUILD_X1=-X $(BUILD_INFO_IMPORT_PATH).GitHash=$(GIT_SHA)
ifdef VERSION
BUILD_X2=-X $(BUILD_INFO_IMPORT_PATH).Version=$(VERSION)
endif
BUILD_X3=-X $(BUILD_INFO_IMPORT_PATH).BuildType=$(BUILD_TYPE)
BUILD_INFO=-ldflags "${BUILD_X1} ${BUILD_X2} ${BUILD_X3}"

RUN_CONFIG?=examples/local/otel-config.yaml

CONTRIB_PATH=$(CURDIR)/../opentelemetry-collector-contrib

# Function to execute a command. Note the empty line before endef to make sure each command
# gets executed separately instead of concatenated with previous one.
# Accepts command to execute as first parameter.
define exec-command
$(1)

endef

.DEFAULT_GOAL := all

.PHONY: all
all: gochecklicense goimpi golint gomisspell gotest otelcol

all-modules:
	@echo $(ALL_MODULES) | tr ' ' '\n' | sort

.PHONY: testbed-loadtest
testbed-loadtest: otelcol
	cd ./testbed/tests && ./runtests.sh

.PHONY: testbed-correctness
testbed-correctness: otelcol
	cd ./testbed/correctness/traces && ./runtests.sh

.PHONY: testbed-list-loadtest
testbed-list-loadtest:
	RUN_TESTBED=1 $(GOTEST) -v ./testbed/tests --test.list '.*'| grep "^Test"

.PHONY: testbed-list-correctness
testbed-list-correctness:
	RUN_TESTBED=1 $(GOTEST) -v ./testbed/correctness --test.list '.*'| grep "^Test"

.PHONY: gotest
gotest:
	@$(MAKE) for-all CMD="make test"

.PHONY: gobenchmark
gobenchmark:
	@$(MAKE) for-all CMD="make benchmark"

.PHONY: gotest-with-cover
gotest-with-cover:
	@echo pre-compiling tests
	@time $(GOTEST) -i ./...
	$(GO_ACC) ./...
	go tool cover -html=coverage.txt -o coverage.html

.PHONY: goaddlicense
goaddlicense:
	@$(MAKE) for-all CMD="make addlicense"

.PHONY: gochecklicense
gochecklicense:
	@$(MAKE) for-all CMD="make checklicense"

.PHONY: gomisspell
gomisspell:
	@$(MAKE) for-all CMD="make misspell"

.PHONY: gomisspell-correction
gomisspell-correction:
	@$(MAKE) for-all CMD="make misspell-correction"

.PHONY: golint
golint:
	@$(MAKE) for-all CMD="make lint"

.PHONY: goimpi
goimpi:
	@$(MAKE) for-all CMD="make impi"

.PHONY: gofmt
gofmt:
	@$(MAKE) for-all CMD="make fmt"

.PHONY: gotidy
gotidy:
	$(MAKE) for-all CMD="rm -fr go.sum"
	$(MAKE) for-all CMD="go mod tidy"

.PHONY: install-tools
install-tools:
	cd $(TOOLS_MOD_DIR) && go install github.com/client9/misspell/cmd/misspell
	cd $(TOOLS_MOD_DIR) && go install github.com/golangci/golangci-lint/cmd/golangci-lint
	cd $(TOOLS_MOD_DIR) && go install github.com/google/addlicense
	cd $(TOOLS_MOD_DIR) && go install github.com/jstemmer/go-junit-report
	cd $(TOOLS_MOD_DIR) && go install github.com/mjibson/esc
	cd $(TOOLS_MOD_DIR) && go install github.com/ory/go-acc
	cd $(TOOLS_MOD_DIR) && go install github.com/pavius/impi/cmd/impi
	cd $(TOOLS_MOD_DIR) && go install github.com/securego/gosec/v2/cmd/gosec
	cd $(TOOLS_MOD_DIR) && go install github.com/tcnksm/ghr
	cd $(TOOLS_MOD_DIR) && go install golang.org/x/tools/cmd/goimports
	cd $(TOOLS_MOD_DIR) && go install honnef.co/go/tools/cmd/staticcheck
	cd cmd/mdatagen && go install ./

.PHONY: otelcol
otelcol:
	go generate ./...
	GO111MODULE=on CGO_ENABLED=0 go build -o ./bin/otelcol_$(GOOS)_$(GOARCH)$(EXTENSION) $(BUILD_INFO) ./cmd/otelcol

.PHONY: run
run:
	GO111MODULE=on go run --race ./cmd/otelcol/... --config ${RUN_CONFIG}

.PHONY: docker-component # Not intended to be used directly
docker-component: check-component
	GOOS=linux $(MAKE) $(COMPONENT)
	cp ./bin/$(COMPONENT)_linux_amd64 ./cmd/$(COMPONENT)/$(COMPONENT)
	docker build -t $(COMPONENT) ./cmd/$(COMPONENT)/
	rm ./cmd/$(COMPONENT)/$(COMPONENT)

.PHONY: for-all
for-all:
	@echo "running $${CMD} in root"
	@$${CMD}
	@set -e; for dir in $(ALL_MODULES); do \
	  (cd "$${dir}" && \
	  	echo "running $${CMD} in $${dir}" && \
	 	$${CMD} ); \
	done

.PHONY: check-component
check-component:
ifndef COMPONENT
	$(error COMPONENT variable was not defined)
endif

.PHONY: add-tag
add-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Adding tag ${TAG}"
	@git tag -a ${TAG} -s -m "Version ${TAG}"

.PHONY: delete-tag
delete-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Deleting tag ${TAG}"
	@git tag -d ${TAG}

.PHONY: docker-otelcol
docker-otelcol:
	COMPONENT=otelcol $(MAKE) docker-component

.PHONY: binaries
binaries: otelcol

.PHONY: binaries-all-sys
binaries-all-sys: binaries-darwin_amd64 binaries-linux_amd64 binaries-linux_arm64 binaries-windows_amd64

.PHONY: binaries-darwin_amd64
binaries-darwin_amd64:
	GOOS=darwin  GOARCH=amd64 $(MAKE) binaries

.PHONY: binaries-linux_amd64
binaries-linux_amd64:
	GOOS=linux   GOARCH=amd64 $(MAKE) binaries

.PHONY: binaries-linux_arm64
binaries-linux_arm64:
	GOOS=linux   GOARCH=arm64 $(MAKE) binaries

.PHONY: binaries-windows_amd64
binaries-windows_amd64:
	GOOS=windows GOARCH=amd64 EXTENSION=.exe $(MAKE) binaries

.PHONY: deb-rpm-package
%-package: ARCH ?= amd64
%-package:
	$(MAKE) binaries-linux_$(ARCH)
	docker build -t otelcol-fpm internal/buildscripts/packaging/fpm
	docker run --rm -v $(CURDIR):/repo -e PACKAGE=$* -e VERSION=$(VERSION) -e ARCH=$(ARCH) otelcol-fpm

.PHONY: genmdata
genmdata:
	$(MAKE) for-all CMD="go generate ./..."

# Definitions for ProtoBuf generation.

# The source directory for OTLP ProtoBufs.
OPENTELEMETRY_PROTO_SRC_DIR=internal/data/opentelemetry-proto

# Find all .proto files.
OPENTELEMETRY_PROTO_FILES := $(subst $(OPENTELEMETRY_PROTO_SRC_DIR)/,,$(wildcard $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/*/v1/*.proto $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/collector/*/v1/*.proto))

# Target directory to write generated files to.
PROTO_TARGET_GEN_DIR=internal/data/opentelemetry-proto-gen

# Go package name to use for generated files.
PROTO_PACKAGE=go.opentelemetry.io/collector/$(PROTO_TARGET_GEN_DIR)

# Intermediate directory used during generation.
PROTO_INTERMEDIATE_DIR=internal/data/.patched-otlp-proto

DOCKER_PROTOBUF ?= otel/build-protobuf:0.1.0
PROTOC := docker run --rm -u ${shell id -u} -v${PWD}:${PWD} -w${PWD}/$(PROTO_INTERMEDIATE_DIR) ${DOCKER_PROTOBUF} --proto_path=${PWD}
PROTO_INCLUDES := -I/usr/include/github.com/gogo/protobuf -I./

# Generate OTLP Protobuf Go files. This will place generated files in PROTO_TARGET_GEN_DIR.
genproto:
	git submodule update --init
	# Call a sub-make to ensure OPENTELEMETRY_PROTO_FILES is populated after the submodule
	# files are present.
	$(MAKE) genproto_sub
	$(MAKE) fmt

genproto_sub:
	@echo Generating code for the following files:
	@$(foreach file,$(OPENTELEMETRY_PROTO_FILES),$(call exec-command,echo $(file)))

	@echo Delete intermediate directory.
	@rm -rf $(PROTO_INTERMEDIATE_DIR)

	@echo Copy .proto file to intermediate directory.
	mkdir -p $(PROTO_INTERMEDIATE_DIR)/opentelemetry
	cp -R $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/* $(PROTO_INTERMEDIATE_DIR)/opentelemetry

	# Patch proto files. See proto_patch.sed for patching rules.
	@echo Modify them in the intermediate directory.
	$(foreach file,$(OPENTELEMETRY_PROTO_FILES),$(call exec-command,sed -f proto_patch.sed $(OPENTELEMETRY_PROTO_SRC_DIR)/$(file) > $(PROTO_INTERMEDIATE_DIR)/$(file)))

	@echo Generate Go code from .proto files in intermediate directory.
	$(foreach file,$(OPENTELEMETRY_PROTO_FILES),$(call exec-command,$(PROTOC) $(PROTO_INCLUDES) --gogofaster_out=plugins=grpc:./ $(file)))

	@echo Generate gRPC gateway code.
	$(PROTOC) $(PROTO_INCLUDES) --grpc-gateway_out=logtostderr=true,grpc_api_configuration=opentelemetry/proto/collector/trace/v1/trace_service_http.yaml:./ opentelemetry/proto/collector/trace/v1/trace_service.proto
	$(PROTOC) $(PROTO_INCLUDES) --grpc-gateway_out=logtostderr=true,grpc_api_configuration=opentelemetry/proto/collector/metrics/v1/metrics_service_http.yaml:./ opentelemetry/proto/collector/metrics/v1/metrics_service.proto
	$(PROTOC) $(PROTO_INCLUDES) --grpc-gateway_out=logtostderr=true,grpc_api_configuration=opentelemetry/proto/collector/logs/v1/logs_service_http.yaml:./ opentelemetry/proto/collector/logs/v1/logs_service.proto

	@echo Move generated code to target directory.
	mkdir -p $(PROTO_TARGET_GEN_DIR)
	cp -R $(PROTO_INTERMEDIATE_DIR)/$(PROTO_PACKAGE)/* $(PROTO_TARGET_GEN_DIR)/
	rm -rf $(PROTO_INTERMEDIATE_DIR)/go.opentelemetry.io

	@rm -rf $(OPENTELEMETRY_PROTO_SRC_DIR)/*
	@rm -rf $(OPENTELEMETRY_PROTO_SRC_DIR)/.* > /dev/null 2>&1 || true

# Generate structs, functions and tests for pdata package. Must be used after any changes
# to proto and after running `make genproto`
genpdata:
	go run cmd/pdatagen/main.go
	$(MAKE) fmt

# Checks that the HEAD of the contrib repo checked out in CONTRIB_PATH compiles
# against the current version of this repo.
.PHONY: check-contrib
check-contrib:
	@echo Setting contrib at $(CONTRIB_PATH) to use this core checkout
	make -C $(CONTRIB_PATH) for-all CMD="go mod edit -replace go.opentelemetry.io/collector=$(CURDIR)"
	make -C $(CONTRIB_PATH) test
	@echo Restoring contrib to no longer use this core checkout
	make -C $(CONTRIB_PATH) for-all CMD="go mod edit -dropreplace go.opentelemetry.io/collector"

# List of directories where certificates are stored for unit tests.
CERT_DIRS := config/configgrpc/testdata \
             config/confighttp/testdata \
             receiver/jaegerreceiver/testdata \
             exporter/jaegerexporter/testdata

# Generate certificates for unit tests relying on certificates.
.PHONY: certs
certs:
	$(foreach dir, $(CERT_DIRS), $(call exec-command, @internal/buildscripts/gen-certs.sh -o $(dir)))

# Generate certificates for unit tests relying on certificates without copying certs to specific test directories.
.PHONY: certs-dryrun
certs-dryrun:
	@internal/buildscripts/gen-certs.sh -d
