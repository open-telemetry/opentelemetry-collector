# More exclusions can be added similar with: -not -path './testbed/*'
ALL_SRC := $(shell find . -name '*.go' \
                                -type f | sort)

# More exclusions can be added similar with: -not -path './testbed/*'
ALL_SRC_NO_TESTBED := $(shell find . -name '*.go' \
                                            -not -path './testbed/*' \
                                            -type f | sort)

# All source code and documents. Used in spell check.
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

# ALL_PKGS is used with 'go test'
ALL_PKGS:= $(shell go list $(sort $(dir $(ALL_SRC))))

# ALL_PKGS_NO_TESTBED is used with 'go cover'
ALL_PKGS_NO_TESTBED := $(shell go list $(sort $(dir $(ALL_SRC_NO_TESTBED))))

GOTEST_OPT?= -race -timeout 180s
GO_ACC=go-acc
GOTEST=go test
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
ADDLICENCESE= addlicense
MISSPELL=misspell -error
MISSPELL_CORRECTION=misspell -w
LINT=golangci-lint
IMPI=impi
GOSEC=gosec
STATIC_CHECK=staticcheck
# BUILD_TYPE should be one of (dev, release).
BUILD_TYPE?=release

GIT_SHA=$(shell git rev-parse --short HEAD)
BUILD_INFO_IMPORT_PATH=github.com/open-telemetry/opentelemetry-collector/internal/version
BUILD_X1=-X $(BUILD_INFO_IMPORT_PATH).GitHash=$(GIT_SHA)
ifdef VERSION
BUILD_X2=-X $(BUILD_INFO_IMPORT_PATH).Version=$(VERSION)
endif
BUILD_X3=-X $(BUILD_INFO_IMPORT_PATH).BuildType=$(BUILD_TYPE)
BUILD_INFO=-ldflags "${BUILD_X1} ${BUILD_X2} ${BUILD_X3}"

RUN_CONFIG=local/config.yaml

all-srcs:
	@echo $(ALL_SRC) | tr ' ' '\n' | sort

all-pkgs:
	@echo $(ALL_PKGS) | tr ' ' '\n' | sort

.DEFAULT_GOAL := all

.PHONY: all
all: checklicense impi lint misspell test otelcol

.PHONY: testbed-runtests
testbed-runtests: otelcol
	cd ./testbed/tests && ./runtests.sh

.PHONY: testbed-listtests
testbed-listtests:
	TESTBED_CONFIG=local.yaml $(GOTEST) -v ./testbed/tests --test.list '.*'|head -n -1

.PHONY: test
test:
	echo $(ALL_PKGS) | xargs -n 10 $(GOTEST) $(GOTEST_OPT)

.PHONY: benchmark
benchmark:
	$(GOTEST) -bench=. -run=notests $(ALL_PKGS_NO_TESTBED)

.PHONY: test-with-cover
test-with-cover:
	@echo Verifying that all packages have test files to count in coverage
	@internal/buildscripts/check-test-files.sh $(subst github.com/open-telemetry/opentelemetry-collector/,./,$(ALL_PKGS_NO_TESTBED))
	@echo pre-compiling tests
	@time go test -i $(ALL_PKGS_NO_TESTBED)
	$(GO_ACC) $(ALL_PKGS_NO_TESTBED)
	go tool cover -html=coverage.txt -o coverage.html

.PHONY: addlicense
addlicense:
	$(ADDLICENCESE) -y '' -c 'OpenTelemetry Authors' $(ALL_SRC)

.PHONY: checklicense
checklicense:
	@ADDLICENCESEOUT=`$(ADDLICENCESE) -check $(ALL_SRC) 2>&1`; \
		if [ "$$ADDLICENCESEOUT" ]; then \
			echo "$(ADDLICENCESE) FAILED => add License errors:\n"; \
			echo "$$ADDLICENCESEOUT\n"; \
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

.PHONY: lint-gosec
lint-gosec:
	# TODO: Consider to use gosec from golangci-lint
	$(GOSEC) -quiet -exclude=G104 ./...

.PHONY: lint-static-check
lint-static-check:
	@STATIC_CHECK_OUT=`$(STATIC_CHECK) ./... 2>&1`; \
		if [ "$$STATIC_CHECK_OUT" ]; then \
			echo "$(STATIC_CHECK) FAILED => static check errors:\n"; \
			echo "$$STATIC_CHECK_OUT\n"; \
			exit 1; \
		else \
			echo "Static check finished successfully"; \
		fi

.PHONY: lint
lint: lint-static-check
	$(LINT) run

.PHONY: impi
impi:
	@$(IMPI) --local github.com/open-telemetry/opentelemetry-collector --scheme stdThirdPartyLocal ./...

.PHONY: fmt
fmt:
	gofmt  -w -s ./
	goimports -w  -local github.com/open-telemetry/opentelemetry-collector ./

.PHONY: install-tools
install-tools:
	go install github.com/client9/misspell/cmd/misspell
	go install github.com/google/addlicense
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go install github.com/jstemmer/go-junit-report
	go install github.com/ory/go-acc
	go install github.com/pavius/impi/cmd/impi
	go install github.com/securego/gosec/cmd/gosec
	go install honnef.co/go/tools/cmd/staticcheck
	go install github.com/tcnksm/ghr

.PHONY: otelcol
otelcol:
	GO111MODULE=on CGO_ENABLED=0 go build -o ./bin/otelcol_$(GOOS)_$(GOARCH) $(BUILD_INFO) ./cmd/otelcol

.PHONY: run
run:
	GO111MODULE=on go run --race ./cmd/otelcol/... --config ${RUN_CONFIG}

.PHONY: docker-component # Not intended to be used directly
docker-component: check-component
	GOOS=linux $(MAKE) $(COMPONENT)
	cp ./bin/$(COMPONENT)_linux_amd64 ./cmd/$(COMPONENT)/$(COMPONENT)
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
	GOOS=windows GOARCH=amd64 $(MAKE) binaries
