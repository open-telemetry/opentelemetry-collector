package main

import (
	"C"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/processor/batchprocessor"
)

// In the real impl, we'll add all sorts of version info here and then,
// in the real (modified) main (or, part of `otelcol`) we will validate
// for matching versions.

func NewFactory() component.Factory {
	return batchprocessor.NewFactory()
}

func main() {
	// In the real world, be polite about it, or,
	// In the real world, this is the entry point for a Collector that uses RPC.
	panic("NOT USED")
}
