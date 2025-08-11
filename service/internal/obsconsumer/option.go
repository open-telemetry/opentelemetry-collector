// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const (
	ComponentOutcome = "otelcol.component.outcome"
)

// Option modifies the consumer behavior.
type Option func(*options)

type options struct {
	staticDataPointAttributes []attribute.KeyValue
}

// WithStaticDataPointAttribute returns an Option that adds a static attribute to data points.
func WithStaticDataPointAttribute(attr attribute.KeyValue) Option {
	return func(opts *options) {
		opts.staticDataPointAttributes = append(opts.staticDataPointAttributes, attr)
	}
}

type compiledOptions struct {
	withSuccessAttrs metric.AddOption
	withFailureAttrs metric.AddOption
	withRefusedAttrs metric.AddOption
}

func (o *options) compile() compiledOptions {
	successAttrs := make([]attribute.KeyValue, 0, 1+len(o.staticDataPointAttributes))
	successAttrs = append(successAttrs, attribute.String(ComponentOutcome, "success"))
	successAttrs = append(successAttrs, o.staticDataPointAttributes...)

	failureAttrs := make([]attribute.KeyValue, 0, 1+len(o.staticDataPointAttributes))
	failureAttrs = append(failureAttrs, attribute.String(ComponentOutcome, "failure"))
	failureAttrs = append(failureAttrs, o.staticDataPointAttributes...)

	refusedAttrs := make([]attribute.KeyValue, 0, 1+len(o.staticDataPointAttributes))
	refusedAttrs = append(refusedAttrs, attribute.String(ComponentOutcome, "refused"))
	refusedAttrs = append(refusedAttrs, o.staticDataPointAttributes...)

	return compiledOptions{
		withSuccessAttrs: metric.WithAttributes(successAttrs...),
		withFailureAttrs: metric.WithAttributes(failureAttrs...),
		withRefusedAttrs: metric.WithAttributes(refusedAttrs...),
	}
}

func zapFields(attrs []attribute.KeyValue) []zap.Field {
	zapFields := make([]zap.Field, 0, len(attrs))
	for _, attr := range attrs {
		var zapField zap.Field
		key := string(attr.Key)
		switch attr.Value.Type() {
		case attribute.BOOL:
			zapField = zap.Bool(key, attr.Value.AsBool())
		case attribute.INT64:
			zapField = zap.Int64(key, attr.Value.AsInt64())
		case attribute.FLOAT64:
			zapField = zap.Float64(key, attr.Value.AsFloat64())
		case attribute.STRING:
			zapField = zap.String(key, attr.Value.AsString())
		case attribute.BOOLSLICE:
			zapField = zap.Bools(key, attr.Value.AsBoolSlice())
		case attribute.INT64SLICE:
			zapField = zap.Int64s(key, attr.Value.AsInt64Slice())
		case attribute.FLOAT64SLICE:
			zapField = zap.Float64s(key, attr.Value.AsFloat64Slice())
		case attribute.STRINGSLICE:
			zapField = zap.Strings(key, attr.Value.AsStringSlice())
		default:
			zapField = zap.Any(key, attr.Value.AsInterface())
		}
		zapFields = append(zapFields, zapField)
	}
	return zapFields
}
