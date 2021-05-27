# Metric Receiver Metadata

Receivers can contain a `metadata.yaml` file that documents the metrics that may be emitted by the receiver.

Current examples:

* [hostmetricsreceiver](../receiver/hostmetricsreceiver/metadata.yaml)

See [metric-metadata.yaml](metric-metadata.yaml) for file format documentation.

If adding a new receiver a `codegen.go` file should also be added to trigger the generation. See below for details.

## Build

When `go generate` is run (it is run automatically in the make build targets) there are a few special build directives in `codegen.go` files:

`make install-tools` results in `cmd/mdatagen` being installed to `GOBIN`

[/receiver/hostmetricsreceiver/codegen.go](../receiver/hostmetricsreceiver/codegen.go) Runs `mdatagen` for the `hostmetricsreceiver` metadata.yaml which generates [/receiver/hostmetricsreceiver/internal/metadata](../receiver/hostmetricsreceiver/internal/metadata) package which has Go files containing metric and label metadata.
