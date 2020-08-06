Exporter to a Cortex instance, or any Prometheus Remote Write compatible backend.

_Here is a link to the overall project [design](https://github.com/open-telemetry/opentelemetry-collector/pull/1464)_

File structure:

- cortex.go: exporter implementation. Converts and sends OTLP metrics

- helper.go: helper functions that cortex.go uses. Performs tasks such as sanitizing label and generating signature string

- config.go: configuration struct of the exporter

- factory.go: initialization methods for creating default configuration and the exporter

- tetsutil.go: helper methods for generating OTLP metrics and Cortex TimeSeries for testing.
