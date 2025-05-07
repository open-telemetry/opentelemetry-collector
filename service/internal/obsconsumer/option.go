// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	ComponentOutcome = "otelcol.component.outcome"
)

// Option modifies the consumer behavior.
type Option interface {
	apply(*options)
}

type options struct {
	staticDataPointAttributes []attribute.KeyValue
}

// WithStaticDataPointAttribute returns an Option that adds a static attribute to data points.
func WithStaticDataPointAttribute(attr attribute.KeyValue) Option {
	return staticDataPointAttributeOption(attr)
}

type staticDataPointAttributeOption attribute.KeyValue

func (o staticDataPointAttributeOption) apply(opts *options) {
	opts.staticDataPointAttributes = append(opts.staticDataPointAttributes, attribute.KeyValue(o))
}

type compiledOptions struct {
	withSuccessAttrs metric.AddOption
	withFailureAttrs metric.AddOption
}

func (o *options) compile() compiledOptions {
	successAttrs := make([]attribute.KeyValue, 0, 1+len(o.staticDataPointAttributes))
	successAttrs = append(successAttrs, attribute.String(ComponentOutcome, "success"))
	successAttrs = append(successAttrs, o.staticDataPointAttributes...)

	failureAttrs := make([]attribute.KeyValue, 0, 1+len(o.staticDataPointAttributes))
	failureAttrs = append(failureAttrs, attribute.String(ComponentOutcome, "failure"))
	failureAttrs = append(failureAttrs, o.staticDataPointAttributes...)

	return compiledOptions{
		withSuccessAttrs: metric.WithAttributes(successAttrs...),
		withFailureAttrs: metric.WithAttributes(failureAttrs...),
	}
}
