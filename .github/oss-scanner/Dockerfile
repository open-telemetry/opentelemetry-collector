FROM golang:1.27@sha256:f20d83a83e18dbd2a4714b8ffd4b30b287b6243962dd75608e86abcc63ba4efc AS build

# libssl-dev: the config/configtls tests use a TPM simulator that is built with cgo against OpenSSL.
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git make libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# scanner contract: the checkout lives inside the image, at /src
COPY . /src
WORKDIR /src

# This repository is ~100 Go modules with no go.work, so each one has to be downloaded on its own. `make otelcorecol`
# only fetches what the collector binary needs, so without this the tests of every other module fail offline.
RUN set -e; for mod in $(find . -name go.mod -not -path './.git/*' | sort); do \
      (cd "$(dirname "$mod")" && go mod download); \
    done

RUN make otelcorecol \
    && ln -s "/src/bin/otelcorecol_$(go env GOOS)_$(go env GOARCH)" /usr/local/bin/otelcorecol

# cmd/builder is in scope for the scan. Generating and compiling a collector with it needs modules that the steps above
# do not fetch (`go mod tidy` resolves them for every platform), so run it once here, while there is network, to put them
# in the module cache. The manifest's `replaces` paths are relative to the output directory, which therefore has to sit
# two levels below /src. The output is removed again to keep the checkout clean.
RUN cd cmd/builder && go run ./ --config ../otelcorecol/builder-config.yaml --output-path ../oss-scanner-warmup \
    && rm -rf /src/cmd/oss-scanner-warmup

# The build is complete and the scan has no network, so fail fast instead of waiting on proxy timeouts if something was
# missed above. GOTOOLCHAIN=local stops go from trying to fetch a newer toolchain.
ENV GOPROXY=off \
    GOTOOLCHAIN=local

# Run the unit tests so the test binaries are compiled and cached for the offline scan, but do not let a failing test
# abort the build. Every module is run on its own, as the repository's own `make gotest` does. The full output goes to
# /opt/oss-scanner-go-test.log (outside /src, so the checkout stays clean); a failure only prints a warning here.
#
# Two in-scope modules have tests that are expected to fail in this environment; they still run so that their test
# binaries are cached, and only print a warning:
#   cmd/builder              some tests run `go mod tidy`, which needs the network
#   cmd/mdatagen             tests that expect a permission error, which root (the scanner's user) never gets
# Two out-of-scope modules are skipped because their tests are expected to fail here:
#   internal/schemagen       tests that expect a permission error, as above
#   internal/e2e             TestMetricStability fails here (not investigated)
# The root module and internal/tools contain no packages, so they are skipped too.
RUN for mod in $(find . -name go.mod -not -path './.git/*' \
        -not -path './internal/schemagen/*' -not -path './internal/e2e/*' | sort); do \
      dir="$(dirname "$mod")"; \
      [ -n "$(cd "$dir" && go list ./... 2>/dev/null)" ] || continue; \
      echo "=== $dir" >> /opt/oss-scanner-go-test.log; \
      (cd "$dir" && go test -count=1 -timeout 300s ./... >> /opt/oss-scanner-go-test.log 2>&1) \
        || echo "WARNING: tests reported failures in $dir"; \
    done
