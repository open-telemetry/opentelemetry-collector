// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/consumer/internal"

// Capabilities describes the capabilities of a Processor.
type Capabilities struct {
	// MutatesData is set to true if Consume* function of the
	// processor modifies the input Traces, Logs or Metrics argument.
	// Processors which modify the input data MUST set this flag to true. If the processor
	// does not modify the data it MUST set this flag to false. If the processor creates
	// a copy of the data before modifying then this flag can be safely set to false.
	MutatesData bool

	// HasReceivedData is set to true if the component has received the data from the upstream
	// and has taken over the responsibility to move them forward without blocking the upstream.
	// Components that process data asynchronously (e.g. batch processor, queue-backed exporters)
	// MUST set this flag to true, as they accept data immediately but process it asynchronously.
	// Synchronous components that block the upstream until processing is complete MUST NOT set
	// this flag (or set it to false).
	//
	// This flag is used to validate expectations about the pipeline behavior:
	// - Ensure a collector is running synchronously end-to-end (for error back-propagation).
	// - Identify components which may be adding substantial latency or burying errors.
	HasReceivedData bool
}

type BaseConsumer interface {
	Capabilities() Capabilities
}

type BaseImpl struct {
	Cap Capabilities
}

// Option to construct new consumers.
type Option interface {
	apply(*BaseImpl)
}

type OptionFunc func(*BaseImpl)

func (of OptionFunc) apply(e *BaseImpl) {
	of(e)
}

// Capabilities returns the capabilities of the component
func (bs BaseImpl) Capabilities() Capabilities {
	return bs.Cap
}

func NewBaseImpl(options ...Option) *BaseImpl {
	bs := &BaseImpl{
		Cap: Capabilities{MutatesData: false},
	}

	for _, op := range options {
		op.apply(bs)
	}

	return bs
}
