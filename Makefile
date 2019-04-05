# More exclusions can be added similar with: -not -path './vendor/*'
ALL_SRC := $(shell find . -name '*.go' \
                                -not -path './vendor/*' \
                                -type f | sort)

# ALL_PKGS is used with 'go cover'
ALL_PKGS := $(shell go list $(sort $(dir $(ALL_SRC))))

GOTEST_OPT?=-v -race -timeout 30s
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -coverprofile=coverage.txt -covermode=atomic
GOTEST=go test
GOFMT=gofmt
GOLINT=golint
GOVET=go vet
GOOS=$(shell go env GOOS)

GIT_SHA=$(shell git rev-parse --short HEAD)
BUILD_INFO_IMPORT_PATH=github.com/census-instrumentation/opencensus-service/internal/version
BUILD_X1=-X $(BUILD_INFO_IMPORT_PATH).GitHash=$(GIT_SHA)
ifdef VERSION
BUILD_X2=-X $(BUILD_INFO_IMPORT_PATH).Version=$(VERSION)
endif
BUILD_INFO=-ldflags "${BUILD_X1} ${BUILD_X2}"

all-pkgs:
	@echo $(ALL_PKGS) | tr ' ' '\n' | sort

all-srcs:
	@echo $(ALL_SRC) | tr ' ' '\n' | sort

.DEFAULT_GOAL := fmt-vet-lint-test

.PHONY: fmt-vet-lint-test
fmt-vet-lint-test: fmt vet lint test

.PHONY: test
test:
	$(GOTEST) $(GOTEST_OPT) $(ALL_PKGS)

.PHONY: travis-ci
travis-ci: fmt install-go-cmp vet lint test-with-cover

.PHONY: test-with-cover
test-with-cover:
	@echo Verifying that all packages have test files to count in coverage
	@scripts/check-test-files.sh $(subst github.com/census-instrumentation/opencensus-service/,./,$(ALL_PKGS))
	@echo pre-compiling tests
	@time go test -i $(ALL_PKGS)
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(ALL_PKGS)
	go tool cover -html=coverage.txt -o coverage.html

.PHONY: fmt
fmt:
	@FMTOUT=`$(GOFMT) -s -l $(ALL_SRC) 2>&1`; \
	if [ "$$FMTOUT" ]; then \
		echo "$(GOFMT) FAILED => gofmt the following files:\n"; \
		echo "$$FMTOUT\n"; \
		exit 1; \
	else \
	    echo "Fmt finished successfully"; \
	fi

.PHONY: lint
lint:
	@LINTOUT=`$(GOLINT) $(ALL_PKGS) 2>&1`; \
	if [ "$$LINTOUT" ]; then \
		echo "$(GOLINT) FAILED => clean the following lint errors:\n"; \
		echo "$$LINTOUT\n"; \
		exit 1; \
	else \
	    echo "Lint finished successfully"; \
	fi

.PHONY: vet
vet:
	@VETOUT=`$(GOVET) ./... 2>&1`; \
	if [ "$$VETOUT" ]; then \
		echo "$(GOVET) FAILED => clean the following vet errors:\n"; \
		echo "$$VETOUT\n"; \
		exit 1; \
	else \
	    echo "Vet finished successfully"; \
	fi

.PHONY: install-tools
install-tools:
	go get golang.org/x/lint/golint

.PHONY: install-go-cmp
install-go-cmp:
	go get -u github.com/google/go-cmp/cmp

.PHONY: agent
agent:
	GO111MODULE=on CGO_ENABLED=0 go build -o ./bin/ocagent_$(GOOS) $(BUILD_INFO) ./cmd/ocagent

.PHONY: collector
collector:
	GO111MODULE=on CGO_ENABLED=0 go build -o ./bin/occollector_$(GOOS) $(BUILD_INFO) ./cmd/occollector

.PHONY: docker-component # Not intended to be used directly
docker-component: check-component
	GOOS=linux $(MAKE) $(COMPONENT)
	cp ./bin/oc$(COMPONENT)_linux ./cmd/oc$(COMPONENT)/
	docker build -t oc$(COMPONENT) ./cmd/oc$(COMPONENT)/
	rm ./cmd/oc$(COMPONENT)/oc$(COMPONENT)_linux

.PHONY: check-component
check-component:
ifndef COMPONENT
	$(error COMPONENT variable was not defined)
endif

.PHONY: docker-agent
docker-agent:
	COMPONENT=agent $(MAKE) docker-component

.PHONY: docker-collector
docker-collector:
	COMPONENT=collector $(MAKE) docker-component


.PHONY: binaries
binaries: agent collector

.PHONY: binaries-all-sys
binaries-all-sys:
	GOOS=darwin $(MAKE) binaries
	GOOS=linux $(MAKE) binaries
	GOOS=windows $(MAKE) binaries
