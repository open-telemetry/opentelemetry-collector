ALL_SRC := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

GOTEST_OPT=-v -race
GOTEST=go test
GOFMT=gofmt
GOOS=$(shell go env GOOS)

GIT_SHA=$(shell git rev-parse --short HEAD)
BUILD_INFO_IMPORT_PATH=github.com/census-instrumentation/opencensus-service/internal/version
BUILD_INFO=-ldflags "-X $(BUILD_INFO_IMPORT_PATH).GitHash=$(GIT_SHA)"

.DEFAULT_GOAL := default_goal

.PHONY: default_goal
default_goal: fmt test

.PHONY: clean
clean:
	rm -rf bin/ 

.PHONY: test
test:
	$(GOTEST) $(GOTEST_OPT) ./...

.PHONY: fmt
fmt:
	@FMTOUT=`$(GOFMT) -s -l $(ALL_SRC) 2>&1`; \
	if [ "$$FMTOUT" ]; then \
		echo "$(GOFMT) FAILED => gofmt the following files:\n"; \
		echo "$$FMTOUT\n"; \
		exit 1; \
	fi

.PHONY: agent
agent:
	CGO_ENABLED=0 go build -o ./bin/ocagent_$(GOOS) $(BUILD_INFO) ./cmd/ocagent

.PHONY: collector
collector:
	CGO_ENABLED=0 go build -o ./bin/occollector_$(GOOS) $(BUILD_INFO) ./cmd/occollector

.PHONY: binaries
binaries: agent collector

.PHONY: binaries-all-sys
binaries-all-sys:
	GOOS=darwin $(MAKE) binaries
	GOOS=linux $(MAKE) binaries
	GOOS=windows $(MAKE) binaries
