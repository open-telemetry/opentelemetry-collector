include ./Makefile.Common

# This is the code that we want to run lint, etc.
ALL_SRC := $(shell find . -name '*.go' \
							-not -path './internal/tools/*' \
							-not -path '*/third_party/*' \
							-not -path './pdata/internal/data/protogen/*' \
							-not -path './service/internal/zpages/tmplgen/*' \
							-type f | sort)

# All source code and documents. Used in spell check.
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

# ALL_MODULES includes ./* dirs (excludes . dir)
ALL_MODULES := $(shell find . -type f -name "go.mod" -exec dirname {} \; | sort | grep -E '^./' )

CMD?=

RUN_CONFIG?=examples/local/otel-config.yaml
CONTRIB_PATH=$(CURDIR)/../opentelemetry-collector-contrib
COMP_REL_PATH=cmd/otelcorecol/components.go
MOD_NAME=go.opentelemetry.io/collector

# Function to execute a command. Note the empty line before endef to make sure each command
# gets executed separately instead of concatenated with previous one.
# Accepts command to execute as first parameter.
define exec-command
$(1)

endef

.DEFAULT_GOAL := all

.PHONY: all
all: checklicense checkdoc misspell goimpi goporto multimod-verify golint gotest

all-modules:
	@echo $(ALL_MODULES) | tr ' ' '\n' | sort

.PHONY: gomoddownload
gomoddownload:
	@$(MAKE) for-all-target TARGET="moddownload"

.PHONY: gotest
gotest:
	@$(MAKE) for-all-target TARGET="test"

.PHONY: gobenchmark
gobenchmark:
	@$(MAKE) for-all-target TARGET="benchmark"
	cat `find . -name benchmark.txt` > benchmarks.txt

.PHONY: gotest-with-cover
gotest-with-cover:
	@$(MAKE) for-all-target TARGET="test-with-cover"
	$(GOCMD) tool covdata textfmt -i=./coverage/unit -o ./coverage.txt

.PHONY: gotestifylint-fix
gotestifylint-fix:
	$(MAKE) for-all-target TARGET="testifylint-fix"

.PHONY: goporto
goporto: $(PORTO)
	$(PORTO) -w --include-internal --skip-dirs "^cmd/mdatagen/third_party$$" ./

.PHONY: for-all
for-all:
	@echo "running $${CMD} in root"
	@$${CMD}
	@set -e; for dir in $(GOMODULES); do \
	  (cd "$${dir}" && \
	  	echo "running $${CMD} in $${dir}" && \
	 	$${CMD} ); \
	done

.PHONY: golint
golint:
	@$(MAKE) for-all-target TARGET="lint"

.PHONY: goimpi
goimpi:
	@$(MAKE) for-all-target TARGET="impi"

.PHONY: gofmt
gofmt:
	@$(MAKE) for-all-target TARGET="fmt"

.PHONY: gotidy
gotidy:
	@$(MAKE) for-all-target TARGET="tidy"

.PHONY: gogenerate
gogenerate:
	cd cmd/mdatagen && $(GOCMD) install .
	@$(MAKE) for-all-target TARGET="generate"
	$(MAKE) fmt

.PHONY: addlicense
addlicense: $(ADDLICENSE)
	@ADDLICENSEOUT=`$(ADDLICENSE) -s=only -y "" -c "The OpenTelemetry Authors" $(ALL_SRC) 2>&1`; \
		if [ "$$ADDLICENSEOUT" ]; then \
			echo "$(ADDLICENSE) FAILED => add License errors:\n"; \
			echo "$$ADDLICENSEOUT\n"; \
			exit 1; \
		else \
			echo "Add License finished successfully"; \
		fi

.PHONY: checklicense
checklicense: $(ADDLICENSE)
	@licRes=$$(for f in $$(find . -type f \( -iname '*.go' -o -iname '*.sh' \) ! -path '**/third_party/*') ; do \
	           awk '/Copyright The OpenTelemetry Authors|generated|GENERATED/ && NR<=3 { found=1; next } END { if (!found) print FILENAME }' $$f; \
			   awk '/SPDX-License-Identifier: Apache-2.0|generated|GENERATED/ && NR<=4 { found=1; next } END { if (!found) print FILENAME }' $$f; \
	   done); \
	   if [ -n "$${licRes}" ]; then \
	           echo "license header checking failed:"; echo "$${licRes}"; \
	           exit 1; \
	   fi

.PHONY: misspell
misspell: $(MISSPELL)
	$(MISSPELL) -error $(ALL_DOC)

.PHONY: misspell-correction
misspell-correction: $(MISSPELL)
	$(MISSPELL) -w $(ALL_DOC)

.PHONY: run
run: otelcorecol
	./bin/otelcorecol_$(GOOS)_$(GOARCH) --config ${RUN_CONFIG} ${RUN_ARGS}

# Append root module to all modules
GOMODULES = $(ALL_MODULES) $(PWD)

# Define a delegation target for each module
.PHONY: $(GOMODULES)
$(GOMODULES):
	@echo "Running target '$(TARGET)' in module '$@'"
	$(MAKE) -C $@ $(TARGET)

# Triggers each module's delegation target
.PHONY: for-all-target
for-all-target: $(GOMODULES)

.PHONY: check-component
check-component:
ifndef COMPONENT
	$(error COMPONENT variable was not defined)
endif

# Build the Collector executable.
.PHONY: otelcorecol
otelcorecol:
	pushd cmd/otelcorecol && CGO_ENABLED=0 $(GOCMD) build -trimpath -o ../../bin/otelcorecol_$(GOOS)_$(GOARCH) \
		-tags $(GO_BUILD_TAGS) ./cmd/otelcorecol && popd

.PHONY: genotelcorecol
genotelcorecol: install-tools
	pushd cmd/builder/ && $(GOCMD) run ./ --skip-compilation --config ../otelcorecol/builder-config.yaml --output-path ../otelcorecol && popd
	$(MAKE) -C cmd/otelcorecol fmt

.PHONY: ocb
ocb:
	$(MAKE) -C cmd/builder config
	$(MAKE) -C cmd/builder ocb

# Definitions for ProtoBuf generation.

# The source directory for OTLP ProtoBufs.
OPENTELEMETRY_PROTO_SRC_DIR=pdata/internal/opentelemetry-proto

# The branch matching the current version of the proto to use
OPENTELEMETRY_PROTO_VERSION=v1.3.1

# Find all .proto files.
OPENTELEMETRY_PROTO_FILES := $(subst $(OPENTELEMETRY_PROTO_SRC_DIR)/,,$(wildcard $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/*/v1/*.proto $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/collector/*/v1/*.proto $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/*/v1experimental/*.proto $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/collector/*/v1experimental/*.proto))

# Target directory to write generated files to.
PROTO_TARGET_GEN_DIR=pdata/internal/data/protogen

# Go package name to use for generated files.
PROTO_PACKAGE=go.opentelemetry.io/collector/$(PROTO_TARGET_GEN_DIR)

# Intermediate directory used during generation.
PROTO_INTERMEDIATE_DIR=pdata/internal/.patched-otlp-proto

DOCKER_PROTOBUF ?= otel/build-protobuf:0.23.0
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

	# HACK: Workaround for istio 1.15 / envoy 1.23.1 mistakenly emitting deprecated field.
	# reserved 1000 -> repeated ScopeLogs deprecated_scope_logs = 1000;
	sed 's/reserved 1000;/repeated ScopeLogs deprecated_scope_logs = 1000;/g' $(PROTO_INTERMEDIATE_DIR)/opentelemetry/proto/logs/v1/logs.proto 1<> $(PROTO_INTERMEDIATE_DIR)/opentelemetry/proto/logs/v1/logs.proto
	# reserved 1000 -> repeated ScopeMetrics deprecated_scope_metrics = 1000;
	sed 's/reserved 1000;/repeated ScopeMetrics deprecated_scope_metrics = 1000;/g' $(PROTO_INTERMEDIATE_DIR)/opentelemetry/proto/metrics/v1/metrics.proto 1<> $(PROTO_INTERMEDIATE_DIR)/opentelemetry/proto/metrics/v1/metrics.proto
	# reserved 1000 -> repeated ScopeSpans deprecated_scope_spans = 1000;
	sed 's/reserved 1000;/repeated ScopeSpans deprecated_scope_spans = 1000;/g' $(PROTO_INTERMEDIATE_DIR)/opentelemetry/proto/trace/v1/trace.proto 1<> $(PROTO_INTERMEDIATE_DIR)/opentelemetry/proto/trace/v1/trace.proto


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
	$(GOCMD) run pdata/internal/cmd/pdatagen/main.go
	$(MAKE) fmt

# Generate semantic convention constants. Requires a clone of the opentelemetry-specification repo
gensemconv: $(SEMCONVGEN) $(SEMCONVKIT)
	@[ "${SPECPATH}" ] || ( echo ">> env var SPECPATH is not set"; exit 1 )
	@[ "${SPECTAG}" ] || ( echo ">> env var SPECTAG is not set"; exit 1 )
	@echo "Generating semantic convention constants from specification version ${SPECTAG} at ${SPECPATH}"
	$(SEMCONVGEN) -o semconv/${SPECTAG} -t semconv/template.j2 -s ${SPECTAG} -i ${SPECPATH}/model/. --only=resource -p conventionType=resource -f generated_resource.go
	$(SEMCONVGEN) -o semconv/${SPECTAG} -t semconv/template.j2 -s ${SPECTAG} -i ${SPECPATH}/model/. --only=event -p conventionType=event -f generated_event.go
	$(SEMCONVGEN) -o semconv/${SPECTAG} -t semconv/template.j2 -s ${SPECTAG} -i ${SPECPATH}/model/. --only=span -p conventionType=trace -f generated_trace.go
	$(SEMCONVGEN) -o semconv/${SPECTAG} -t semconv/template.j2 -s ${SPECTAG} -i ${SPECPATH}/model/. --only=attribute_group -p conventionType=attribute_group -f generated_attribute_group.go
	$(SEMCONVKIT) -output "semconv/$(SPECTAG)" -tag "$(SPECTAG)"

# Checks that the HEAD of the contrib repo checked out in CONTRIB_PATH compiles
# against the current version of this repo.
.PHONY: check-contrib
check-contrib:
	@echo Setting contrib at $(CONTRIB_PATH) to use this core checkout
	@$(MAKE) -C $(CONTRIB_PATH) for-all CMD="$(GOCMD) mod edit \
		-replace go.opentelemetry.io/collector=$(CURDIR) \
		-replace go.opentelemetry.io/collector/client=$(CURDIR)/client \
		-replace go.opentelemetry.io/collector/cmd/mdatagen=$(CURDIR)/cmd/mdatagen \
		-replace go.opentelemetry.io/collector/component=$(CURDIR)/component  \
		-replace go.opentelemetry.io/collector/component/componentstatus=$(CURDIR)/component/componentstatus  \
		-replace go.opentelemetry.io/collector/config/configauth=$(CURDIR)/config/configauth  \
		-replace go.opentelemetry.io/collector/config/configcompression=$(CURDIR)/config/configcompression  \
		-replace go.opentelemetry.io/collector/config/configgrpc=$(CURDIR)/config/configgrpc  \
		-replace go.opentelemetry.io/collector/config/confighttp=$(CURDIR)/config/confighttp  \
		-replace go.opentelemetry.io/collector/config/confignet=$(CURDIR)/config/confignet  \
		-replace go.opentelemetry.io/collector/config/configopaque=$(CURDIR)/config/configopaque  \
		-replace go.opentelemetry.io/collector/config/configretry=$(CURDIR)/config/configretry  \
		-replace go.opentelemetry.io/collector/config/configtelemetry=$(CURDIR)/config/configtelemetry  \
		-replace go.opentelemetry.io/collector/config/configtls=$(CURDIR)/config/configtls  \
		-replace go.opentelemetry.io/collector/config/internal=$(CURDIR)/config/internal  \
		-replace go.opentelemetry.io/collector/confmap=$(CURDIR)/confmap  \
		-replace go.opentelemetry.io/collector/confmap/converter/expandconverter=$(CURDIR)/confmap/converter/expandconverter  \
		-replace go.opentelemetry.io/collector/confmap/provider/envprovider=$(CURDIR)/confmap/provider/envprovider  \
		-replace go.opentelemetry.io/collector/confmap/provider/fileprovider=$(CURDIR)/confmap/provider/fileprovider  \
		-replace go.opentelemetry.io/collector/confmap/provider/httpprovider=$(CURDIR)/confmap/provider/httpprovider  \
		-replace go.opentelemetry.io/collector/confmap/provider/httpsprovider=$(CURDIR)/confmap/provider/httpsprovider  \
		-replace go.opentelemetry.io/collector/confmap/provider/yamlprovider=$(CURDIR)/confmap/provider/yamlprovider  \
		-replace go.opentelemetry.io/collector/connector=$(CURDIR)/connector  \
		-replace go.opentelemetry.io/collector/connector/connectortest=$(CURDIR)/connector/connectortest  \
		-replace go.opentelemetry.io/collector/connector/connectorprofiles=$(CURDIR)/connector/connectorprofiles  \
		-replace go.opentelemetry.io/collector/connector/forwardconnector=$(CURDIR)/connector/forwardconnector  \
		-replace go.opentelemetry.io/collector/consumer=$(CURDIR)/consumer  \
		-replace go.opentelemetry.io/collector/consumer/consumererror=$(CURDIR)/consumer/consumererror  \
		-replace go.opentelemetry.io/collector/consumer/consumererror/consumererrorprofiles=$(CURDIR)/consumer/consumererror/consumererrorprofiles  \
		-replace go.opentelemetry.io/collector/consumer/consumerprofiles=$(CURDIR)/consumer/consumerprofiles  \
		-replace go.opentelemetry.io/collector/consumer/consumertest=$(CURDIR)/consumer/consumertest  \
		-replace go.opentelemetry.io/collector/exporter=$(CURDIR)/exporter  \
		-replace go.opentelemetry.io/collector/exporter/debugexporter=$(CURDIR)/exporter/debugexporter  \
		-replace go.opentelemetry.io/collector/exporter/exporterprofiles=$(CURDIR)/exporter/exporterprofiles  \
		-replace go.opentelemetry.io/collector/exporter/exportertest=$(CURDIR)/exporter/exportertest  \
		-replace go.opentelemetry.io/collector/exporter/exporterhelper/exporterhelperprofiles=$(CURDIR)/exporter/exporterhelper/exporterhelperprofiles  \
		-replace go.opentelemetry.io/collector/exporter/nopexporter=$(CURDIR)/exporter/nopexporter  \
		-replace go.opentelemetry.io/collector/exporter/otlpexporter=$(CURDIR)/exporter/otlpexporter  \
		-replace go.opentelemetry.io/collector/exporter/otlphttpexporter=$(CURDIR)/exporter/otlphttpexporter  \
		-replace go.opentelemetry.io/collector/extension=$(CURDIR)/extension  \
		-replace go.opentelemetry.io/collector/extension/auth=$(CURDIR)/extension/auth  \
		-replace go.opentelemetry.io/collector/extension/experimental/storage=$(CURDIR)/extension/experimental/storage  \
		-replace go.opentelemetry.io/collector/extension/extensioncapabilities=$(CURDIR)/extension/extensioncapabilities  \
		-replace go.opentelemetry.io/collector/extension/memorylimiterextension=$(CURDIR)/extension/memorylimiterextension  \
		-replace go.opentelemetry.io/collector/extension/zpagesextension=$(CURDIR)/extension/zpagesextension  \
		-replace go.opentelemetry.io/collector/featuregate=$(CURDIR)/featuregate  \
		-replace go.opentelemetry.io/collector/filter=$(CURDIR)/filter  \
		-replace go.opentelemetry.io/collector/internal/memorylimiter=$(CURDIR)/internal/memorylimiter  \
		-replace go.opentelemetry.io/collector/otelcol=$(CURDIR)/otelcol  \
		-replace go.opentelemetry.io/collector/otelcol/otelcoltest=$(CURDIR)/otelcol/otelcoltest  \
		-replace go.opentelemetry.io/collector/pdata=$(CURDIR)/pdata  \
		-replace go.opentelemetry.io/collector/pdata/testdata=$(CURDIR)/pdata/testdata  \
		-replace go.opentelemetry.io/collector/pdata/pprofile=$(CURDIR)/pdata/pprofile  \
		-replace go.opentelemetry.io/collector/pipeline=$(CURDIR)/pipeline  \
		-replace go.opentelemetry.io/collector/pipeline/pipelineprofiles=$(CURDIR)/pipeline/pipelineprofiles  \
		-replace go.opentelemetry.io/collector/processor=$(CURDIR)/processor  \
		-replace go.opentelemetry.io/collector/processor/processortest=$(CURDIR)/processor/processortest  \
		-replace go.opentelemetry.io/collector/processor/batchprocessor=$(CURDIR)/processor/batchprocessor  \
		-replace go.opentelemetry.io/collector/processor/memorylimiterprocessor=$(CURDIR)/processor/memorylimiterprocessor  \
		-replace go.opentelemetry.io/collector/processor/processorprofiles=$(CURDIR)/processor/processorprofiles  \
		-replace go.opentelemetry.io/collector/receiver=$(CURDIR)/receiver  \
		-replace go.opentelemetry.io/collector/receiver/nopreceiver=$(CURDIR)/receiver/nopreceiver  \
		-replace go.opentelemetry.io/collector/receiver/otlpreceiver=$(CURDIR)/receiver/otlpreceiver  \
		-replace go.opentelemetry.io/collector/receiver/receiverprofiles=$(CURDIR)/receiver/receiverprofiles  \
		-replace go.opentelemetry.io/collector/semconv=$(CURDIR)/semconv  \
		-replace go.opentelemetry.io/collector/service=$(CURDIR)/service"
	@$(MAKE) -C $(CONTRIB_PATH) gotidy
	@$(MAKE) -C $(CONTRIB_PATH) gotest
	@if [ -z "$(SKIP_RESTORE_CONTRIB)" ]; then \
		$(MAKE) restore-contrib; \
	fi

# Restores contrib to its original state after running check-contrib.
.PHONY: restore-contrib
restore-contrib:
	@echo Restoring contrib at $(CONTRIB_PATH) to its original state
	@$(MAKE) -C $(CONTRIB_PATH) for-all CMD="$(GOCMD) mod edit \
		-dropreplace go.opentelemetry.io/collector \
		-dropreplace go.opentelemetry.io/collector/client \
		-dropreplace go.opentelemetry.io/collector/cmd/mdatagen \
		-dropreplace go.opentelemetry.io/collector/component \
		-dropreplace go.opentelemetry.io/collector/component/componentstatus \
		-dropreplace go.opentelemetry.io/collector/config/configauth  \
		-dropreplace go.opentelemetry.io/collector/config/configcompression  \
		-dropreplace go.opentelemetry.io/collector/config/configgrpc  \
		-dropreplace go.opentelemetry.io/collector/config/confighttp  \
		-dropreplace go.opentelemetry.io/collector/config/confignet  \
		-dropreplace go.opentelemetry.io/collector/config/configopaque  \
		-dropreplace go.opentelemetry.io/collector/config/configretry  \
		-dropreplace go.opentelemetry.io/collector/config/configtelemetry  \
		-dropreplace go.opentelemetry.io/collector/config/configtls  \
		-dropreplace go.opentelemetry.io/collector/config/internal  \
		-dropreplace go.opentelemetry.io/collector/confmap  \
		-dropreplace go.opentelemetry.io/collector/confmap/converter/expandconverter  \
		-dropreplace go.opentelemetry.io/collector/confmap/provider/envprovider  \
		-dropreplace go.opentelemetry.io/collector/confmap/provider/fileprovider  \
		-dropreplace go.opentelemetry.io/collector/confmap/provider/httpprovider  \
		-dropreplace go.opentelemetry.io/collector/confmap/provider/httpsprovider  \
		-dropreplace go.opentelemetry.io/collector/confmap/provider/yamlprovider  \
		-dropreplace go.opentelemetry.io/collector/connector  \
		-dropreplace go.opentelemetry.io/collector/connector/connectortest  \
		-dropreplace go.opentelemetry.io/collector/connector/connectorprofiles  \
		-dropreplace go.opentelemetry.io/collector/connector/forwardconnector  \
		-dropreplace go.opentelemetry.io/collector/consumer  \
		-dropreplace go.opentelemetry.io/collector/consumer/consumererror/consumererrorprofiles  \
		-dropreplace go.opentelemetry.io/collector/consumer/consumerprofiles  \
		-dropreplace go.opentelemetry.io/collector/consumer/consumertest  \
		-dropreplace go.opentelemetry.io/collector/exporter  \
		-dropreplace go.opentelemetry.io/collector/exporter/exporterhelper/exporterhelperprofiles  \
		-dropreplace go.opentelemetry.io/collector/exporter/exportertest  \
		-dropreplace go.opentelemetry.io/collector/exporter/debugexporter  \
		-dropreplace go.opentelemetry.io/collector/exporter/nopexporter  \
		-dropreplace go.opentelemetry.io/collector/exporter/otlpexporter  \
		-dropreplace go.opentelemetry.io/collector/exporter/otlphttpexporter  \
		-dropreplace go.opentelemetry.io/collector/extension  \
		-dropreplace go.opentelemetry.io/collector/extension/auth  \
		-dropreplace go.opentelemetry.io/collector/extension/memorylimiterextension  \
		-dropreplace go.opentelemetry.io/collector/extension/zpagesextension  \
		-dropreplace go.opentelemetry.io/collector/featuregate  \
		-dropreplace go.opentelemetry.io/collector/filter  \
		-dropreplace go.opentelemetry.io/collector/internal/memorylimiter \
		-dropreplace go.opentelemetry.io/collector/otelcol  \
		-dropreplace go.opentelemetry.io/collector/otelcol/otelcoltest  \
		-dropreplace go.opentelemetry.io/collector/pdata  \
		-dropreplace go.opentelemetry.io/collector/pdata/testdata  \
		-dropreplace go.opentelemetry.io/collector/pdata/pprofile  \
		-dropreplace go.opentelemetry.io/collector/pipeline  \
		-dropreplace go.opentelemetry.io/collector/pipeline/pipelineprofiles \
		-dropreplace go.opentelemetry.io/collector/processor  \
		-dropreplace go.opentelemetry.io/collector/processortest  \
		-dropreplace go.opentelemetry.io/collector/processor/batchprocessor  \
		-dropreplace go.opentelemetry.io/collector/processor/memorylimiterprocessor  \
		-dropreplace go.opentelemetry.io/collector/receiver  \
		-dropreplace go.opentelemetry.io/collector/receiver/nopreceiver  \
		-dropreplace go.opentelemetry.io/collector/receiver/otlpreceiver  \
		-dropreplace go.opentelemetry.io/collector/semconv  \
		-dropreplace go.opentelemetry.io/collector/service"
	@$(MAKE) -C $(CONTRIB_PATH) -j2 gotidy

# List of directories where certificates are stored for unit tests.
CERT_DIRS := localhost|""|config/configgrpc/testdata \
             localhost|""|config/confighttp/testdata \
             example1|"-1"|config/configtls/testdata \
             example2|"-2"|config/configtls/testdata
cert-domain = $(firstword $(subst |, ,$1))
cert-suffix = $(word 2,$(subst |, ,$1))
cert-dir = $(word 3,$(subst |, ,$1))

# Generate certificates for unit tests relying on certificates.
.PHONY: certs
certs:
	$(foreach dir, $(CERT_DIRS), $(call exec-command, @internal/buildscripts/gen-certs.sh -o $(call cert-dir,$(dir)) -s $(call cert-suffix,$(dir)) -m $(call cert-domain,$(dir))))

# Generate certificates for unit tests relying on certificates without copying certs to specific test directories.
.PHONY: certs-dryrun
certs-dryrun:
	@internal/buildscripts/gen-certs.sh -d

# Verify existence of READMEs for components specified as default components in the collector.
.PHONY: checkdoc
checkdoc: $(CHECKFILE)
	$(CHECKFILE) --project-path $(CURDIR) --component-rel-path $(COMP_REL_PATH) --module-name $(MOD_NAME) --file-name "README.md"

# Construct new API state snapshots
.PHONY: apidiff-build
apidiff-build: $(APIDIFF)
	@$(foreach pkg,$(ALL_PKGS),$(call exec-command,./internal/buildscripts/gen-apidiff.sh -p $(pkg)))

# If we are running in CI, change input directory
ifeq ($(CI), true)
APICOMPARE_OPTS=$(COMPARE_OPTS)
else
APICOMPARE_OPTS=-d "./internal/data/apidiff"
endif

# Compare API state snapshots
.PHONY: apidiff-compare
apidiff-compare: $(APIDIFF)
	@$(foreach pkg,$(ALL_PKGS),$(call exec-command,./internal/buildscripts/compare-apidiff.sh -p $(pkg)))

.PHONY: multimod-verify
multimod-verify: $(MULTIMOD)
	@echo "Validating versions.yaml"
	$(MULTIMOD) verify

MODSET?=stable
.PHONY: multimod-prerelease
multimod-prerelease: $(MULTIMOD)
	$(MULTIMOD) prerelease -s=true -b=false -v ./versions.yaml -m ${MODSET}
	$(MAKE) gotidy

COMMIT?=HEAD
REMOTE?=git@github.com:open-telemetry/opentelemetry-collector.git
.PHONY: push-tags
push-tags: $(MULTIMOD)
	$(MULTIMOD) verify
	set -e; for tag in `$(MULTIMOD) tag -m ${MODSET} -c ${COMMIT} --print-tags | grep -v "Using" `; do \
		echo "pushing tag $${tag}"; \
		git push ${REMOTE} $${tag}; \
	done;

.PHONY: check-changes
check-changes: $(MULTIMOD)
	$(MULTIMOD) diff -p $(PREVIOUS_VERSION) -m $(MODSET)

.PHONY: prepare-release
prepare-release:
ifndef MODSET
	@echo "MODSET not defined"
	@echo "usage: make prepare-release RELEASE_CANDIDATE=<version eg 0.53.0> PREVIOUS_VERSION=<version eg 0.52.0> MODSET=beta"
	exit 1
endif
ifdef PREVIOUS_VERSION
	@echo "Previous version $(PREVIOUS_VERSION)"
else
	@echo "PREVIOUS_VERSION not defined"
	@echo "usage: make prepare-release RELEASE_CANDIDATE=<version eg 0.53.0> PREVIOUS_VERSION=<version eg 0.52.0> MODSET=beta"
	exit 1
endif
ifdef RELEASE_CANDIDATE
	@echo "Preparing ${MODSET} release $(RELEASE_CANDIDATE)"
else
	@echo "RELEASE_CANDIDATE not defined"
	@echo "usage: make prepare-release RELEASE_CANDIDATE=<version eg 0.53.0> PREVIOUS_VERSION=<version eg 0.52.0> MODSET=beta"
	exit 1
endif
	# ensure a clean branch
	git diff -s --exit-code || (echo "local repository not clean"; exit 1)
	# update files with new version
	sed -i.bak 's/$(PREVIOUS_VERSION)/$(RELEASE_CANDIDATE)/g' versions.yaml
	sed -i.bak 's/$(PREVIOUS_VERSION)/$(RELEASE_CANDIDATE)/g' ./cmd/builder/internal/builder/config.go
	sed -i.bak 's/$(PREVIOUS_VERSION)/$(RELEASE_CANDIDATE)/g' ./cmd/builder/test/core.builder.yaml
	sed -i.bak 's/$(PREVIOUS_VERSION)/$(RELEASE_CANDIDATE)/g' ./cmd/otelcorecol/builder-config.yaml
	sed -i.bak 's/$(PREVIOUS_VERSION)/$(RELEASE_CANDIDATE)/g' examples/k8s/otel-config.yaml
	find . -name "*.bak" -type f -delete
	# commit changes before running multimod
	git add .
	git commit -m "prepare release $(RELEASE_CANDIDATE)"
	$(MAKE) multimod-prerelease
	# regenerate files
	$(MAKE) -C cmd/builder config
	$(MAKE) genotelcorecol
	git add .
	git commit -m "add multimod changes $(RELEASE_CANDIDATE)" || (echo "no multimod changes to commit")

.PHONY: clean
clean:
	test -d bin && $(RM) bin/*

.PHONY: checklinks
checklinks:
	command -v markdown-link-check >/dev/null 2>&1 || { echo >&2 "markdown-link-check not installed. Run 'npm install -g markdown-link-check'"; exit 1; }
	find . -name \*.md -print0 | xargs -0 -n1 \
		markdown-link-check -q -c ./.github/workflows/check_links_config.json || true

# error message "failed to sync logger:  sync /dev/stderr: inappropriate ioctl for device"
# is a known issue but does not affect function.
.PHONY: crosslink
crosslink: $(CROSSLINK)
	@echo "Executing crosslink"
	$(CROSSLINK) --root=$(shell pwd) --prune

FILENAME?=$(shell git branch --show-current)
.PHONY: chlog-new
chlog-new: $(CHLOGGEN)
	$(CHLOGGEN) new --config $(CHLOGGEN_CONFIG) --filename $(FILENAME)

.PHONY: chlog-validate
chlog-validate: $(CHLOGGEN)
	$(CHLOGGEN) validate --config $(CHLOGGEN_CONFIG)

.PHONY: chlog-preview
chlog-preview: $(CHLOGGEN)
	$(CHLOGGEN) update --config $(CHLOGGEN_CONFIG) --dry

.PHONY: chlog-update
chlog-update: $(CHLOGGEN)
	$(CHLOGGEN) update --config $(CHLOGGEN_CONFIG) --version $(VERSION)

.PHONY: builder-integration-test
builder-integration-test: $(ENVSUBST)
	cd ./cmd/builder && ./test/test.sh

.PHONY: mdatagen-test
mdatagen-test:
	cd cmd/mdatagen && $(GOCMD) install .
	cd cmd/mdatagen && $(GOCMD) generate ./...
	cd cmd/mdatagen && $(GOCMD) test ./...
