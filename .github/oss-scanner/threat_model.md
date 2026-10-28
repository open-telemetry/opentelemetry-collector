# Threat model

## What this project does and where untrusted input enters
This is the core repository of the OpenTelemetry Collector: a Go framework and a set of core components that receive telemetry (traces, metrics, logs, profiles), process it, and export it to backends. It is about 100 Go modules in one repository with no `go.work`, so most commands have to be run from inside a module directory. This repository does not build any official binary. `cmd/otelcorecol` is a small local binary for testing, and the real distributions are assembled elsewhere from these modules. The components here are the foundation that those distributions, and a large ecosystem of third-party components, build on.

The trust boundary is authentication. The collector is meant to be deployed so that clients must authenticate before it accepts their telemetry, and an endpoint the operator leaves open is outside this model.

Untrusted input:
- Anything a network peer can send up to and including the authentication decision, which is everything an unauthenticated peer can reach: the connection and TLS handshake (`config/configtls`), HTTP and gRPC framing, headers and the credentials presented to the authentication extensions (`config/configauth`, `config/configmiddleware`), and whatever the shared server code in `config/confighttp`, `config/configgrpc` and `config/configcompression` (request handling, decompression, size limits, CORS) does with a request before authentication has accepted it. Extensions that serve HTTP, such as zPages, are in the same position.
- Responses from the backends an exporter talks to (`exporter/otlpexporter`, `exporter/otlphttpexporter`, the client side of `confighttp` and `configgrpc`). The operator chooses the endpoint, but a response can still be hostile, for example a malformed body or an unexpected status.

Trusted input:
- Telemetry from an authenticated client. This is what `receiver/otlpreceiver` accepts over gRPC and HTTP (protobuf and JSON) and decodes in `pdata/`, which holds the in-memory data model and the proto and JSON decoders for all signals, including `pdata/pprofile`, and what then flows through the pipeline. An authenticated client is not an attacker here. Separate clients are still separate principals: one client's data, credentials and auth context must not reach another.
- Operator-controlled input. The collector configuration and everything it can reference: YAML files, `--set` flags, environment variables, and the `file:`, `env:`, `yaml:`, `http:` and `https:` config providers in `confmap/`. An attacker who can change the configuration already controls the collector.
- The builder manifest (`cmd/builder`), `metadata.yaml` files (`cmd/mdatagen`), and the set of components compiled into a binary. A component the operator compiled in is trusted code.

## Components that matter most / least
Most:
- `config/confighttp`, `config/configgrpc`, `config/configcompression`, `config/configtls`, `config/configauth`, `config/configmiddleware`: everything that runs before and during authentication (request parsing, decompression, TLS handshake and client-certificate checks, how credentials are validated and when authentication is applied), and the handling of credentials and headers on outbound requests.
- `receiver/otlpreceiver`: how it hands requests to the server code above, and that authentication and per-client context are applied to every request on every protocol. The payload decoders it calls in `pdata/` (`pdata/internal`, `pdata/plog`, `pdata/pmetric`, `pdata/ptrace`, `pdata/pprofile` and their `*otlp` packages) only see authenticated input, so a flaw in them matters less unless it can be reached without authenticating or crosses between clients.
- `config/configopaque` and `confmap/`: secrets that are supposed to be masked when the configuration is printed or logged, and the semantics of `${...}` expansion.
- `otelcol/` and `service/`: configuration validation, startup, the collector's own telemetry endpoints.
- `extension/zpagesextension` and the other extensions: anything that serves internal state over HTTP.
- `exporter/exporterhelper` and the queue and batch code under `exporter/exporterhelper/internal` and `processor/`: where data is buffered and, when a storage extension is configured, persisted.

Least, still in scope but unlikely to matter: `receiver/nopreceiver`, `exporter/nopexporter`, `exporter/debugexporter` (it prints telemetry on purpose), `connector/forwardconnector`, and the `*helper` packages other than their security-relevant paths above.

Out of scope: `cmd/mdatagen`, `cmd/builder`, `cmd/schemagen`, `cmd/githubgen` and `internal/cmd` (developer and build-time tools driven only by trusted input), `internal/e2e`, every `*test` package (`componenttest`, `receivertest`, etc.), `examples/`, `docs/`, and `.github/`. The generators are out of scope as programs, but the code they emit is not: files named `generated_*` (most of `pdata/`, the `generated_config.go` files) are overwritten by `make genpdata` and `make gogenerate`, so when you find a flaw in one, say which generator template produces it and patch the template as well.

## How to exercise it
- The core collector is built at `/usr/local/bin/otelcorecol`. `otelcorecol --config /src/examples/local/otel-config.yaml` starts an OTLP receiver on `localhost:4317` (gRPC) and `localhost:4318` (HTTP) that logs what it receives through the debug exporter, plus zPages on `localhost:55679`. `otelcorecol validate --config <file>` and `otelcorecol components` also work.
- With the collector running, send it OTLP over HTTP with `curl`, for example `curl -X POST localhost:4318/v1/traces -H 'Content-Type: application/json' -d '{...}'`, or craft protobuf bodies with a small Go program.
- There is no network here and `GOPROXY=off`. Every module and the Go build and test cache were fetched during the image build.
- This is a multi-module repository: run tests from inside a module, for example `cd /src/pdata && go test ./...` or `cd /src/receiver/otlpreceiver && go test ./...`. `go test ./...` from `/src` matches no packages, because the root module has none.
- Fuzz targets exist for the decoders and the receiver: `pdata/plog/fuzz_test.go`, `pdata/pmetric/fuzz_test.go`, `pdata/ptrace/fuzz_test.go`, `pdata/pprofile/fuzz_test.go`, `pdata/xpdata/fuzz_test.go`, the `*otlp` packages beside them, and `receiver/otlpreceiver/fuzz_test.go`. Run one with, for example, `cd /src/pdata && go test ./plog -run '^$' -fuzz FuzzUnmarshalJSONLogs -fuzztime 60s`.
- The unit tests were run during the image build, module by module, and their output is in `/opt/oss-scanner-go-test.log`. A built image does not prove they passed, so check that log before relying on a baseline. Tests that bind to loopback work; tests that need the Internet do not.
- Certificates and keys under `testdata/` directories are throwaway fixtures for the tests.

## How you rate severity
The project's published security model applies (https://github.com/open-telemetry/.github/blob/main/SECURITY.md): confidentiality and integrity are the topmost concerns. Availability is not: defaults are not considered secure against availability attacks, users are expected to put authentication in front of their endpoints, and a denial of service by an authenticated client is not a vulnerability. Rate with that in mind.

- **critical**: remote code execution or memory corruption reachable by an unauthenticated peer, or by a hostile backend response.
- **high**: bypassing authentication, authorization or TLS verification that the operator configured (for example a client certificate that is not checked, or `configauth` not being applied to a request on one of the receiver's protocols). Disclosure of credentials or other configured secrets, such as values `configopaque` should mask, bearer tokens or API keys sent in headers, to logs, zPages, the collector's own telemetry, or a host the operator did not configure. Reading or writing files outside what the configuration allows, or making requests to hosts the configuration does not name, driven by unauthenticated input or a backend response.
- **medium**: data leaking between clients, tenants or pipelines against the configuration, including client metadata or auth context carried over to another request.
- **low**: availability problems with a high asymmetry, such as one small unauthenticated request that crashes the process or grows memory without bound. Everything else about availability (volume-based floods, slow clients, resource use that is proportional to the input) is a bug, not a vulnerability, and should not be rated above low.

Not a vulnerability:
- Anything that needs a valid client credential. Telemetry from an authenticated client is trusted, so a payload that crashes, stalls or exhausts memory in a decoder, a processor or an exporter, or one that makes the collector mishandle its own data, is a bug and should be reported as one. The exception is a flaw that breaks isolation between clients, which is rated medium above.
- Anything that needs control over the configuration, environment variables, the builder manifest or the compiled-in components.
- Behaviour the operator opted into explicitly, such as `insecure: true`, `insecure_skip_verify`, or running an endpoint with no authentication configured.
- `debugexporter` printing telemetry, and zPages exposing internal state on its default `localhost` endpoint. Both are by design.
- Weaknesses in a dependency that cannot be reached from this code. If one can, show the call path.

Reports should name the module and commit, and include the input or a Go test that triggers the issue. A proposed patch should be a minimal diff against `main` that includes a test and follows `docs/coding-guidelines.md`.
