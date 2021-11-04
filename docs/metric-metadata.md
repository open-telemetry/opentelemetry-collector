# Metric Receiver Metadata

Receivers can contain a `metadata.yaml` file that documents the metrics that may be emitted by the receiver.

Current examples:

* hostmetricsreceiver scrapers like the [cpuscraper](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/hostmetricsreceiver/internal/scraper/cpuscraper/metadata.yaml)

See [metric-metadata.yaml](metric-metadata.yaml) for file format documentation.

If adding a new receiver a `codegen.go` file should also be added to trigger the generation. See below for details.

## Build

When `go generate` is run (it is run automatically in the make build targets) there are a few special build directives in `codegen.go` files:

`make install-tools` results in `cmd/mdatagen` being installed to `GOBIN`

[/receiver/hostmetricsreceiver/internal/scraper/cpuscraper/codegen.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/hostmetricsreceiver/internal/scraper/cpuscraper/codegen.go) runs `mdatagen` for the `hostmetricsreceiver` metadata.yaml which generates the [/receiver/hostmetricsreceiver/internal/scraper/cpuscraper/internal/metadata](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/hostmetricsreceiver/internal/scraper/cpuscraper/internal/metadata) package which has Go files containing metric and label metadata.
