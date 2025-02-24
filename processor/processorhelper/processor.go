// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/internal"
)

// ErrSkipProcessingData is a sentinel value to indicate when traces or metrics should intentionally be dropped
// from further processing in the pipeline because the data is determined to be irrelevant. A processor can return this error
// to stop further processing without propagating an error back up the pipeline to logs.
var ErrSkipProcessingData = errors.New("sentinel error to skip processing data from the remainder of the pipeline")

// Option apply changes to internalOptions.
type Option interface {
	apply(*baseSettings)
}

type optionFunc func(*baseSettings)

func (of optionFunc) apply(e *baseSettings) {
	of(e)
}

// WithComponentOptions overrides the default component (start/shutdown) functions for a processor.
// The default functions do nothing and always returns nil.
func WithComponentOptions(option ...component.Option) Option {
	return optionFunc(func(e *baseSettings) {
		e.componentOptions = append(e.componentOptions, option...)
	})
}

// Deprecated: [v0.121.0] use WithComponentOptions.
func WithStart(start component.StartFunc) Option {
	return WithComponentOptions(component.WithStartFunc(start))
}

// Deprecated: [v0.121.0] use WithComponentOptions.
func WithShutdown(shutdown component.ShutdownFunc) Option {
	return WithComponentOptions(component.WithShutdownFunc(shutdown))
}

// WithCapabilities overrides the default GetCapabilities function for an processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return optionFunc(func(o *baseSettings) {
		o.consumerOptions = append(o.consumerOptions, consumer.WithCapabilities(capabilities))
	})
}

type baseSettings struct {
	componentOptions []component.Option
	consumerOptions  []consumer.Option
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

func spanAttributes(id component.ID) trace.EventOption {
	return trace.WithAttributes(attribute.String(internal.ProcessorKey, id.String()))
}
