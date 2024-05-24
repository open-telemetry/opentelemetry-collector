// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer // import "go.opentelemetry.io/collector/consumer"

// Capabilities describes the capabilities of a Processor.
type Capabilities struct {
	// MutatesData is set to true if Consume* function of the
	// processor modifies the input Traces, Logs or Metrics argument.
	// Processors which modify the input data MUST set this flag to true. If the processor
	// does not modify the data it MUST set this flag to false. If the processor creates
	// a copy of the data before modifying then this flag can be safely set to false.
	MutatesData bool
}
