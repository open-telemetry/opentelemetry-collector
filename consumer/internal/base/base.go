// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package base // import "go.opentelemetry.io/collector/consumer/internal/base"

import "go.opentelemetry.io/collector/consumer"

// Option to construct new consumers.
type Option func(*Impl)

// Consumer is the base interface for any consumer implementation.
type Consumer interface {
	Capabilities() consumer.Capabilities
}

// Impl is the base implementation for the consumer implementation.
type Impl struct {
	capabilities consumer.Capabilities
}

// Capabilities returns the capabilities of the component
func (bs Impl) Capabilities() consumer.Capabilities {
	return bs.capabilities
}

// WithCapabilities overrides the default GetCapabilities function for a processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return func(o *Impl) {
		o.capabilities = capabilities
	}
}

// NewImpl builds a new instance of Impl
func NewImpl(options ...Option) *Impl {
	bs := &Impl{
		capabilities: consumer.Capabilities{MutatesData: false},
	}

	for _, op := range options {
		op(bs)
	}

	return bs
}
