GOTEST_OPT?=-v -race -timeout 30s
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -coverprofile=coverage.txt -covermode=atomic
GOTEST=go test
LINT=golangci-lint
GOOS=$(shell go env GOOS)

.DEFAULT_GOAL := lint-test

.PHONY: lint-test
fmt-vet-lint-test: lint test

.PHONY: test
test:
	$(GOTEST) $(GOTEST_OPT) ./...

.PHONY: lint
lint:
	$(LINT) run -c ../.golangci.yml

.PHONY: runtests
runtests: test
	./runtests.sh

.PHONY: install-tools
install-tools:
	go install github.com/jstemmer/go-junit-report
