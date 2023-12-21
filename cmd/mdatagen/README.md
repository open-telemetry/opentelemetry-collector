# Metadata Generator

Every component's documentation should include a brief description of the component and guidance on how to use it.
There is also some information about the component (or metadata) that should be included to help end-users understand the current state of the component and whether it is right for their use case.
Examples of this metadata about a component are:

* its stability level
* the distributions containing it
* the types of pipelines it supports
* metrics emitted in the case of a scraping receiver

The metadata generator defines a schema for specifying this information to ensure it is complete and well-formed.
The metadata generator is then able to ingest the metadata, validate it against the schema and produce documentation in a standardized format.
An example of how this generated documentation looks can be found in [documentation.md](./documentation.md).

## Using the Metadata Generator

In order for a component to benefit from the metadata generator (`mdatagen`) these requirements need to be met:
1. A `metadata.yaml` file containing the metadata needs to be included in the component
2. The component should declare a `go:generate mdatagen` directive which tells `mdatagen` what to generate

As an example, here is a minimal `metadata.yaml` for the [OTLP receiver](https://github.com/open-telemetry/opentelemetry-collector/tree/main/receiver/otlpreceiver):
```yaml
type: otlp
status:
  class: receiver
  stability:
    beta: [logs]
    stable: [metrics, traces]
```

Detailed information about the schema of `metadata.yaml` can be found in [metadata-schema.yaml](./metadata-schema.yaml).

The `go:generate mdatagen` directive is usually defined in a `doc.go` file in the same package as the component, for example:
```go
//go:generate mdatagen metadata.yaml

package main
```

Below are some more examples that can be used for reference:

* The ElasticSearch receiver has an extensive [metadata.yaml](../../receiver/elasticsearchreceiver/metadata.yaml)
* The host metrics receiver has internal subcomponents, each with their own `metadata.yaml` and `doc.go`. See [cpuscraper](../../receiver/hostmetricsreceiver/internal/scraper/cpuscraper) for example.

You can run `cd cmd/mdatagen && $(GOCMD) install .` to install the `mdatagen` tool in `GOBIN` and then run `mdatagen metadata.yaml` to generate documentation for a specific component or you can run `make generate` to generate documentation for all components.

## Contributing to the Metadata Generator

The code for generating the documentation can be found in [loader.go](./loader.go) and the templates for rendering the documentation can be found in [templates](./templates).
When making updates to the metadata generator or introducing support for new functionality:

1. Ensure the [metadata-schema.yaml](./metadata-schema.yaml) and [./metadata.yaml](metadata.yaml) files reflect the changes.
2. Run `make mdatagen-test`.
3. Make sure all tests are passing including [generated tests](./internal/metadata/generated_metrics_test.go).
4. Run `make generate`.
