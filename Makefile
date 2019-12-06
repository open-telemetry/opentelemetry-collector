# More exclusions can be added similar with: -not -path './testbed/*'
ALL_SRC := $(shell find . -name '*.go' \
                                -not -path './testbed/*' \
                                -type f | sort)

# All source code and documents. Used in spell check.
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

# ALL_PKGS is used with 'go cover'
ALL_PKGS := $(shell go list $(sort $(dir $(ALL_SRC))))

GOTEST_OPT?= -race -timeout 30s
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -coverprofile=coverage.txt -covermode=atomic
GOTEST=go test
GOOS=$(shell go env GOOS)
ADDLICENCESE= addlicense
MISSPELL=misspell -error
MISSPELL_CORRECTION=misspell -w
LINT=golangci-lint
IMPI=impi

GIT_SHA=$(shell git rev-parse --short HEAD)
BUILD_INFO_IMPORT_PATH=github.com/open-telemetry/opentelemetry-collector/internal/version
BUILD_X1=-X $(BUILD_INFO_IMPORT_PATH).GitHash=$(GIT_SHA)
ifdef VERSION
BUILD_X2=-X $(BUILD_INFO_IMPORT_PATH).Version=$(VERSION)
endif
BUILD_INFO=-ldflags "${BUILD_X1} ${BUILD_X2}"

all-pkgs:
	@echo $(ALL_PKGS) | tr ' ' '\n' | sort

all-srcs:
	@echo $(ALL_SRC) | tr ' ' '\n' | sort

.DEFAULT_GOAL := all

.PHONY: all
all: addlicense impi lint misspell test otelcol

.PHONY: e2e-test
e2e-test: otelcol
	$(MAKE) -C testbed runtests

.PHONY: test
test:
	$(GOTEST) $(GOTEST_OPT) $(ALL_PKGS)

.PHONY: benchmark
benchmark:
	$(GOTEST) -bench=. -run=notests $(ALL_PKGS)

.PHONY: travis-ci
travis-ci: all test-with-cover
	$(MAKE) -C testbed install-tools
	$(MAKE) -C testbed runtests

.PHONY: test-with-cover
test-with-cover:
	@echo Verifying that all packages have test files to count in coverage
	@scripts/check-test-files.sh $(subst github.com/open-telemetry/opentelemetry-collector/,./,$(ALL_PKGS))
	@echo pre-compiling tests
	@time go test -i $(ALL_PKGS)
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(ALL_PKGS)
	go tool cover -html=coverage.txt -o coverage.html

.PHONY: addlicense
addlicense:
	@ADDLICENCESEOUT=`$(ADDLICENCESE) -y 2019 -c 'OpenTelemetry Authors' $(ALL_SRC) 2>&1`; \
		if [ "$$ADDLICENCESEOUT" ]; then \
			echo "$(ADDLICENCESE) FAILED => add License errors:\n"; \
			echo "$$ADDLICENCESEOUT\n"; \
			exit 1; \
		else \
			echo "Add License finished successfully"; \
		fi

.PHONY: misspell
misspell:
	$(MISSPELL) $(ALL_DOC)

.PHONY: misspell-correction
misspell-correction:
	$(MISSPELL_CORRECTION) $(ALL_DOC)

.PHONY: lint
lint:
	$(LINT) run

.PHONY: impi
impi:
	@$(IMPI) --local github.com/open-telemetry/opentelemetry-collector --scheme stdThirdPartyLocal ./...

.PHONY: install-tools
install-tools:
	GO111MODULE=on go install \
	  github.com/google/addlicense \
	  github.com/golangci/golangci-lint/cmd/golangci-lint \
	  github.com/client9/misspell/cmd/misspell \
	  github.com/pavius/impi/cmd/impi

.PHONY: otelcol
otelcol:
	GO111MODULE=on CGO_ENABLED=0 go build -o ./bin/$(GOOS)/otelcol $(BUILD_INFO) ./cmd/otelcol

.PHONY: docker-component # Not intended to be used directly
docker-component: check-component
	GOOS=linux $(MAKE) $(COMPONENT)
	cp ./bin/linux/$(COMPONENT) ./cmd/$(COMPONENT)/
	docker build -t $(COMPONENT) ./cmd/$(COMPONENT)/
	rm ./cmd/$(COMPONENT)/$(COMPONENT)

.PHONY: check-component
check-component:
ifndef COMPONENT
	$(error COMPONENT variable was not defined)
endif

.PHONY: docker-otelcol
docker-otelcol:
	COMPONENT=otelcol $(MAKE) docker-component

.PHONY: binaries
binaries: otelcol

.PHONY: binaries-all-sys
binaries-all-sys:
	GOOS=darwin $(MAKE) binaries
	GOOS=linux $(MAKE) binaries
	GOOS=windows $(MAKE) binaries
