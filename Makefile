include ./Makefile.Common

# This is the code that we want to run lint, etc.
ALL_SRC := $(shell find . -name '*.go' \
							-not -path './internal/tools/*' \
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

ADDLICENSE=addlicense
GOCOVMERGE=gocovmerge
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
all: checklicense checkdoc misspell goimpi golint gotest

all-modules:
	@echo $(ALL_MODULES) | tr ' ' '\n' | sort

.PHONY: gomoddownload
gomoddownload:
	@$(MAKE) for-all CMD="go mod download"

.PHONY: gotest
gotest:
	@$(MAKE) for-all CMD="make test test-unstable"

.PHONY: gobenchmark
gobenchmark:
	@$(MAKE) for-all CMD="make benchmark"

.PHONY: gotest-with-cover
gotest-with-cover:
	@$(MAKE) for-all CMD="make test-with-cover"
	$(GOCOVMERGE) $$(find . -name coverage.out) > coverage.txt

.PHONY: goporto
goporto:
	@$(MAKE) for-all CMD="make porto"

.PHONY: golint
golint:
	@$(MAKE) for-all CMD="make lint lint-unstable"

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
	cd $(TOOLS_MOD_DIR) && go install github.com/ory/go-acc
	cd $(TOOLS_MOD_DIR) && go install github.com/pavius/impi/cmd/impi
	cd $(TOOLS_MOD_DIR) && go install github.com/tcnksm/ghr
	cd $(TOOLS_MOD_DIR) && go install github.com/wadey/gocovmerge
	cd $(TOOLS_MOD_DIR) && go install go.opentelemetry.io/build-tools/checkdoc
	cd $(TOOLS_MOD_DIR) && go install go.opentelemetry.io/build-tools/semconvgen
	cd $(TOOLS_MOD_DIR) && go install golang.org/x/exp/cmd/apidiff
	cd $(TOOLS_MOD_DIR) && go install golang.org/x/tools/cmd/goimports
	cd $(TOOLS_MOD_DIR) && go install github.com/jcchavezs/porto/cmd/porto
	cd $(TOOLS_MOD_DIR) && go install go.opentelemetry.io/build-tools/multimod

.PHONY: run
run: otelcorecol
	./bin/otelcorecol_$(GOOS)_$(GOARCH) --config ${RUN_CONFIG} ${RUN_ARGS}

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

# Build the Collector executable.
.PHONY: otelcorecol
otelcorecol:
	pushd cmd/otelcorecol && GO111MODULE=on CGO_ENABLED=0 go build -trimpath -o ../../bin/otelcorecol_$(GOOS)_$(GOARCH) \
		$(BUILD_INFO) -tags $(GO_BUILD_TAGS) ./cmd/otelcorecol && popd

.PHONY: genotelcorecol
genotelcorecol:
	pushd cmd/builder/ && go run ./ --skip-compilation --config ../otelcorecol/builder-config.yaml --output-path ../otelcorecol && popd

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

# This target should run on /bin/bash since the syntax DIR=$${dir:1} is not supported by /bin/sh.
.PHONY: gendependabot
gendependabot: $(eval SHELL:=/bin/bash)
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

# The SHA matching the current version of the proto to use
OPENTELEMETRY_PROTO_VERSION=v0.12.0

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

# Cleanup temporary directory
genproto-cleanup:
	rm -Rf ${OPENTELEMETRY_PROTO_SRC_DIR}

# Generate OTLP Protobuf Go files. This will place generated files in PROTO_TARGET_GEN_DIR.
genproto: genproto-cleanup
	mkdir -p ${OPENTELEMETRY_PROTO_SRC_DIR}
	curl -sSL https://api.github.com/repos/open-telemetry/opentelemetry-proto/tarball/${OPENTELEMETRY_PROTO_VERSION} | tar xz --strip 1 -C ${OPENTELEMETRY_PROTO_SRC_DIR}
	# Call a sub-make to ensure OPENTELEMETRY_PROTO_FILES is populated
	$(MAKE) genproto_sub
	$(MAKE) fmt
	$(MAKE) genproto-cleanup

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
	go run model/internal/cmd/pdatagen/main.go
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

.PHONY: multimod-verify
multimod-verify: install-tools
	@echo "Validating versions.yaml"
	multimod verify

.PHONY: multimod-prerelease
multimod-prerelease: install-tools
	multimod prerelease -v ./versions.yaml -m collector-base
