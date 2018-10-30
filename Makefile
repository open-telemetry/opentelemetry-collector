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

.PHONY: agent-all-sys
agent-all-platforms:
	GOOS=darwin $(MAKE) agent
	GOOS=linux $(MAKE) agent
	GOOS=windows $(MAKE) agent
