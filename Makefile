include ./Makefile.Common

# This is the code that we want to run lint, etc.
ALL_SRC := $(shell find . -name '*.go' \
							-not -path './cmd/schemagen/*' \
							-not -path './internal/tools/*' \
							-not -path './examples/demo/app/*' \
							-not -path './model/internal/data/protogen/*' \
							-not -path './service/internal/zpages/tmplgen/*' \
							-type f | sort)

# All source code and documents. Used in spell check.
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

# ALL_MODULES includes ./* dirs (excludes . dir)
ALL_MODULES := $(shell find . -type f -name "go.mod" -exec dirname {} \; | sort | egrep  '^./' )

CMD?=
TOOLS_MOD_DIR := ./internal/tools

GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

BUILD_INFO_IMPORT_PATH=go.opentelemetry.io/collector/internal/version
VERSION=$(shell git describe --always --match "v[0-9]*" HEAD)
BUILD_INFO=-ldflags "-X $(BUILD_INFO_IMPORT_PATH).Version=$(VERSION)"

RUN_CONFIG?=examples/local/otel-config.yaml
CONTRIB_PATH=$(CURDIR)/../opentelemetry-collector-contrib
COMP_REL_PATH=service/defaultcomponents/defaults.go
MOD_NAME=go.opentelemetry.io/collector

GO_ACC=go-acc
ADDLICENSE=addlicense
MISSPELL=misspell -error
MISSPELL_CORRECTION=misspell -w

# Function to execute a command. Note the empty line before endef to make sure each command
# gets executed separately instead of concatenated with previous one.
# Accepts command to execute as first parameter.
define exec-command
$(1)

endef

.DEFAULT_GOAL := all

.PHONY: version
version:
	@echo ${VERSION}

.PHONY: all
all: checklicense checkdoc misspell goimpi golint gotest otelcol

all-modules:
	@echo $(ALL_MODULES) | tr ' ' '\n' | sort

.PHONY: gomoddownload
gomoddownload:
	@$(MAKE) for-all CMD="go mod download"

.PHONY: gotestinstall
gotestinstall:
	@$(MAKE) for-all CMD="make test GOTEST_OPT=\"-i\""

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
	$(MAKE) for-all CMD="go mod tidy -go=1.16"
	$(MAKE) for-all CMD="go mod tidy -go=1.17"

.PHONY: addlicense
addlicense:
	@ADDLICENSEOUT=`$(ADDLICENSE) -y "" -c "The OpenTelemetry Authors" $(ALL_SRC) 2>&1`; \
		if [ "$$ADDLICENSEOUT" ]; then \
			echo "$(ADDLICENSE) FAILED => add License errors:\n"; \
			echo "$$ADDLICENSEOUT\n"; \
			exit 1; \
		else \
			echo "Add License finished successfully"; \
		fi

.PHONY: checklicense
checklicense:
	@ADDLICENSEOUT=`$(ADDLICENSE) -check $(ALL_SRC) 2>&1`; \
		if [ "$$ADDLICENSEOUT" ]; then \
			echo "$(ADDLICENSE) FAILED => add License errors:\n"; \
			echo "$$ADDLICENSEOUT\n"; \
			echo "Use 'make addlicense' to fix this."; \
			exit 1; \
		else \
			echo "Check License finished successfully"; \
		fi

.PHONY: misspell
misspell:
	$(MISSPELL) $(ALL_DOC)

.PHONY: misspell-correction
misspell-correction:
	$(MISSPELL_CORRECTION) $(ALL_DOC)

.PHONY: install-tools
install-tools:
	cd $(TOOLS_MOD_DIR) && go install github.com/client9/misspell/cmd/misspell
	cd $(TOOLS_MOD_DIR) && go install github.com/golangci/golangci-lint/cmd/golangci-lint
	cd $(TOOLS_MOD_DIR) && go install github.com/google/addlicense
	cd $(TOOLS_MOD_DIR) && go install github.com/jstemmer/go-junit-report
	cd $(TOOLS_MOD_DIR) && go install github.com/mjibson/esc
	cd $(TOOLS_MOD_DIR) && go install github.com/ory/go-acc
	cd $(TOOLS_MOD_DIR) && go install github.com/pavius/impi/cmd/impi
	cd $(TOOLS_MOD_DIR) && go install github.com/tcnksm/ghr
	cd $(TOOLS_MOD_DIR) && go install go.opentelemetry.io/build-tools/semconvgen
	cd $(TOOLS_MOD_DIR) && go install go.opentelemetry.io/build-tools/checkdoc
	cd $(TOOLS_MOD_DIR) && go install golang.org/x/exp/cmd/apidiff
	cd $(TOOLS_MOD_DIR) && go install golang.org/x/tools/cmd/goimports

.PHONY: otelcol
otelcol:
	go generate ./...
	$(MAKE) build-binary-internal

.PHONY: run
run:
	GO111MODULE=on go run --race ./cmd/otelcol/... --config ${RUN_CONFIG} ${RUN_ARGS}

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
	@set -e; for dir in $(ALL_MODULES); do \
	  (echo Adding tag "$${dir:2}/$${TAG}" && \
	 	git tag -a "$${dir:2}/$${TAG}" -s -m "Version ${dir:2}/${TAG}" ); \
	done

.PHONY: push-tag
push-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Pushing tag ${TAG}"
	@git push upstream ${TAG}
	@set -e; for dir in $(ALL_MODULES); do \
	  (echo Pushing tag "$${dir:2}/$${TAG}" && \
	 	git push upstream "$${dir:2}/$${TAG}"); \
	done

.PHONY: delete-tag
delete-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Deleting tag ${TAG}"
	@git tag -d ${TAG}
	@set -e; for dir in $(ALL_MODULES); do \
	  (echo Deleting tag "$${dir:2}/$${TAG}" && \
	 	git tag -d "$${dir:2}/$${TAG}" ); \
	done

.PHONY: docker-otelcol
docker-otelcol:
	COMPONENT=otelcol $(MAKE) docker-component

# build collector binaries with different OS and Architecture
.PHONY: binaries-all-sys
binaries-all-sys: binaries-darwin_amd64 binaries-darwin_arm64 binaries-linux_amd64 binaries-linux_arm64 binaries-windows_amd64

.PHONY: binaries-darwin_amd64
binaries-darwin_amd64:
	GOOS=darwin  GOARCH=amd64 $(MAKE) build-binary-internal

.PHONY: binaries-darwin_arm64
binaries-darwin_arm64:
	GOOS=darwin  GOARCH=arm64 $(MAKE) build-binary-internal

.PHONY: binaries-linux_amd64
binaries-linux_amd64:
	GOOS=linux   GOARCH=amd64 $(MAKE) build-binary-internal

.PHONY: binaries-linux_arm64
binaries-linux_arm64:
	GOOS=linux   GOARCH=arm64 $(MAKE) build-binary-internal

.PHONY: binaries-windows_amd64
binaries-windows_amd64:
	GOOS=windows GOARCH=amd64 EXTENSION=.exe $(MAKE) build-binary-internal

.PHONY: build-binary-internal
build-binary-internal:
	GO111MODULE=on CGO_ENABLED=0 go build -trimpath -o ./bin/otelcol_$(GOOS)_$(GOARCH)$(EXTENSION) $(BUILD_INFO) ./cmd/otelcol

.PHONY: deb-rpm-package
%-package: ARCH ?= amd64
%-package:
	$(MAKE) binaries-linux_$(ARCH)
	docker build -t otelcol-fpm internal/buildscripts/packaging/fpm
	docker run --rm -v $(CURDIR):/repo -e PACKAGE=$* -e VERSION=$(VERSION) -e ARCH=$(ARCH) otelcol-fpm

.PHONY: genmdata
genmdata:
	$(MAKE) for-all CMD="go generate ./..."

DEPENDABOT_PATH=".github/dependabot.yml"
.PHONY: internal-gendependabot
internal-gendependabot:
	@echo "Add rule for \"${PACKAGE}\" in \"${DIR}\"";
	@echo "  - package-ecosystem: \"${PACKAGE}\"" >> ${DEPENDABOT_PATH};
	@echo "    directory: \"${DIR}\"" >> ${DEPENDABOT_PATH};
	@echo "    schedule:" >> ${DEPENDABOT_PATH};
	@echo "      interval: \"weekly\"" >> ${DEPENDABOT_PATH};

.PHONY: gendependabot
gendependabot:
	@echo "Recreating ${DEPENDABOT_PATH} file"
	@echo "# File generated by \"make gendependabot\"; DO NOT EDIT." > ${DEPENDABOT_PATH}
	@echo "" >> ${DEPENDABOT_PATH}
	@echo "version: 2" >> ${DEPENDABOT_PATH}
	@echo "updates:" >> ${DEPENDABOT_PATH}
	$(MAKE) internal-gendependabot DIR="/" PACKAGE="github-actions"
	$(MAKE) internal-gendependabot DIR="/" PACKAGE="docker"
	$(MAKE) internal-gendependabot DIR="/" PACKAGE="gomod"
	@set -e; for dir in $(ALL_MODULES); do \
		$(MAKE) internal-gendependabot DIR=$${dir:1} PACKAGE="gomod"; \
	done

# Definitions for ProtoBuf generation.

# The source directory for OTLP ProtoBufs.
OPENTELEMETRY_PROTO_SRC_DIR=model/internal/opentelemetry-proto

# Find all .proto files.
OPENTELEMETRY_PROTO_FILES := $(subst $(OPENTELEMETRY_PROTO_SRC_DIR)/,,$(wildcard $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/*/v1/*.proto $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/collector/*/v1/*.proto))

# Target directory to write generated files to.
PROTO_TARGET_GEN_DIR=model/internal/data/protogen

# Go package name to use for generated files.
PROTO_PACKAGE=go.opentelemetry.io/collector/$(PROTO_TARGET_GEN_DIR)

# Intermediate directory used during generation.
PROTO_INTERMEDIATE_DIR=model/internal/.patched-otlp-proto

DOCKER_PROTOBUF ?= otel/build-protobuf:0.4.1
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

# Generate semantic convention constants. Requires a clone of the opentelemetry-specification repo
gensemconv:
	@[ "${SPECPATH}" ] || ( echo ">> env var SPECPATH is not set"; exit 1 )
	@[ "${SPECTAG}" ] || ( echo ">> env var SPECTAG is not set"; exit 1 )
	@echo "Generating semantic convention constants from specification version ${SPECTAG} at ${SPECPATH}"
	semconvgen -o model/semconv/${SPECTAG} -t model/internal/semconv/template.j2 -s ${SPECTAG} -i ${SPECPATH}/semantic_conventions/resource -p conventionType=resource
	semconvgen -o model/semconv/${SPECTAG} -t model/internal/semconv/template.j2 -s ${SPECTAG} -i ${SPECPATH}/semantic_conventions/trace -p conventionType=trace

# Checks that the HEAD of the contrib repo checked out in CONTRIB_PATH compiles
# against the current version of this repo.
.PHONY: check-contrib
check-contrib:
	@echo Setting contrib at $(CONTRIB_PATH) to use this core checkout
	make -C $(CONTRIB_PATH) for-all CMD="go mod edit -replace go.opentelemetry.io/collector=$(CURDIR)"
	make -C $(CONTRIB_PATH) for-all CMD="go mod edit -replace go.opentelemetry.io/collector/model=$(CURDIR)/model"
	make -C $(CONTRIB_PATH) for-all CMD="go mod tidy -go=1.16"
	make -C $(CONTRIB_PATH) for-all CMD="go mod tidy -go=1.17"
	make -C $(CONTRIB_PATH) test
	@echo Restoring contrib to no longer use this core checkout
	make -C $(CONTRIB_PATH) for-all CMD="go mod edit -dropreplace go.opentelemetry.io/collector"

# List of directories where certificates are stored for unit tests.
CERT_DIRS := config/configgrpc/testdata \
             config/confighttp/testdata

# Generate certificates for unit tests relying on certificates.
.PHONY: certs
certs:
	$(foreach dir, $(CERT_DIRS), $(call exec-command, @internal/buildscripts/gen-certs.sh -o $(dir)))

# Generate certificates for unit tests relying on certificates without copying certs to specific test directories.
.PHONY: certs-dryrun
certs-dryrun:
	@internal/buildscripts/gen-certs.sh -d

# Verify existence of READMEs for components specified as default components in the collector.
.PHONY: checkdoc
checkdoc:
	checkdoc --project-path $(CURDIR) --component-rel-path $(COMP_REL_PATH) --module-name $(MOD_NAME)

# Construct new API state snapshots
.PHONY: apidiff-build
apidiff-build:
	@$(foreach pkg,$(ALL_PKGS),$(call exec-command,./internal/buildscripts/gen-apidiff.sh -p $(pkg)))

# If we are running in CI, change input directory
ifeq ($(CI), true)
APICOMPARE_OPTS=$(COMPARE_OPTS)
else
APICOMPARE_OPTS=-d "./internal/data/apidiff"
endif

# Compare API state snapshots
.PHONY: apidiff-compare
apidiff-compare:
	@$(foreach pkg,$(ALL_PKGS),$(call exec-command,./internal/buildscripts/compare-apidiff.sh -p $(pkg)))
