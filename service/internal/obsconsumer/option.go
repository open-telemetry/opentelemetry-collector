// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"go.opentelemetry.io/otel/attribute"
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
