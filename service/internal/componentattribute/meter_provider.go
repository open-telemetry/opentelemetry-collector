// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/service/internal/componentattribute"

import (
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type meterProviderWithAttributes struct {
	metric.MeterProvider
	attrs []attribute.KeyValue
}

func (mpwa meterProviderWithAttributes) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	conf := metric.NewMeterConfig(opts...)
	attrSet := conf.InstrumentationAttributes()
	// prepend our attributes so they can be overwritten
	newAttrs := append(slices.Clone(mpwa.attrs), attrSet.ToSlice()...)
	// append our attribute set option to overwrite the old one
	opts = append(opts, metric.WithInstrumentationAttributes(newAttrs...))
	return mpwa.MeterProvider.Meter(name, opts...)
}

func (mpwa meterProviderWithAttributes) DropInjectedAttributes(droppedAttrs ...string) metric.MeterProvider {
	return meterProviderWithAttributes{
		MeterProvider: mpwa.MeterProvider,
		attrs: slices.DeleteFunc(slices.Clone(mpwa.attrs), func(kv attribute.KeyValue) bool {
			return slices.Contains(droppedAttrs, string(kv.Key))
		}),
	}
}
