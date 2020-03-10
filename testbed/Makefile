GOTEST_OPT?=-v -race -timeout 30s
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -coverprofile=coverage.txt -covermode=atomic
GOTEST=go test
GOFMT=gofmt
GOLINT=golint
GOVET=go vet
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

.DEFAULT_GOAL := fmt-vet-lint-test

.PHONY: fmt-vet-lint-test
fmt-vet-lint-test: fmt vet lint test

.PHONY: test
test:
	$(GOTEST) $(GOTEST_OPT) ./...

.PHONY: runtests
runtests: test
	./runtests.sh

.PHONY: fmt
fmt:
	$(GOFMT) -s -l ./..

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
	$(GOVET) ./...

.PHONY: install-tools
install-tools:
	go install golang.org/x/lint/golint
	go install github.com/jstemmer/go-junit-report
