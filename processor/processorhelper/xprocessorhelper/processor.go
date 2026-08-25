// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xprocessorhelper // import "go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// Option apply changes to internalOptions.
type Option interface {
	apply(*baseSettings)
}

type optionFunc func(*baseSettings)

func (of optionFunc) apply(e *baseSettings) {
	of(e)
}

// WithStart overrides the default Start function for an processor.
// The default shutdown function does nothing and always returns nil.
func WithStart(start component.StartFunc) Option {
	return optionFunc(func(o *baseSettings) {
		o.StartFunc = start
	})
}

// WithShutdown overrides the default Shutdown function for an processor.
// The default shutdown function does nothing and always returns nil.
func WithShutdown(shutdown component.ShutdownFunc) Option {
	return optionFunc(func(o *baseSettings) {
		o.ShutdownFunc = shutdown
	})
}

// WithCapabilities overrides the default GetCapabilities function for an processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return optionFunc(func(o *baseSettings) {
		o.consumerOptions = append(o.consumerOptions, consumer.WithCapabilities(capabilities))
	})
}

type baseSettings struct {
	component.StartFunc
	component.ShutdownFunc
	consumerOptions []consumer.Option
}

// fromOptions returns the internal settings starting from the default and applying all options.
func fromOptions(options []Option) *baseSettings {
	// Start from the default options:
	opts := &baseSettings{
		consumerOptions: []consumer.Option{consumer.WithCapabilities(consumer.Capabilities{MutatesData: true})},
	}

	for _, op := range options {
		op.apply(opts)
	}

	return opts
}
