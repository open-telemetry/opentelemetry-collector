// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func ViewOptionsFromConfig(views []View) []sdkmetric.Option {
	opts := []sdkmetric.Option{}
	for _, view := range views {
		if view.Selector == nil || view.Stream == nil {
			continue
		}
		opts = append(opts, sdkmetric.WithView(
			sdkmetric.NewView(
				sdkmetric.Instrument{
					Name: view.Selector.InstrumentNameStr(),
					Kind: instrumentTypeToKind(view.Selector.InstrumentTypeStr()),
					// TODO: add unit once https://github.com/open-telemetry/opentelemetry-configuration/pull/38
					//       is merged
					// Unit: *view.Selector.Unit,
					Scope: instrumentation.Scope{
						Name:      view.Selector.MeterNameStr(),
						Version:   view.Selector.MeterVersionStr(),
						SchemaURL: view.Selector.MeterSchemaURLStr(),
					},
				},
				sdkmetric.Stream{
					Name:            view.Stream.NameStr(),
					Description:     view.Stream.DescriptionStr(),
					Aggregation:     viewStreamAggregationToAggregation(view.Stream.Aggregation),
					AttributeFilter: attributeKeysToAttributeFilter(view.Stream.AttributeKeys),
				},
			),
		))
	}
	return opts
}

var invalidInstrumentKind = sdkmetric.InstrumentKind(0)

func instrumentTypeToKind(instrument string) sdkmetric.InstrumentKind {
	switch instrument {
	case "counter":
		return sdkmetric.InstrumentKindCounter
	case "histogram":
		return sdkmetric.InstrumentKindHistogram
	case "observable_counter":
		return sdkmetric.InstrumentKindObservableCounter
	case "observable_gauge":
		return sdkmetric.InstrumentKindObservableGauge
	case "observable_updown_counter":
		return sdkmetric.InstrumentKindObservableUpDownCounter
	case "updown_counter":
		return sdkmetric.InstrumentKindUpDownCounter
	}
	return invalidInstrumentKind
}

func attributeKeysToAttributeFilter(keys []string) attribute.Filter {
	kvs := make([]attribute.KeyValue, len(keys))
	for i, key := range keys {
		kvs[i] = attribute.Bool(key, true)
	}
	filter := attribute.NewSet(kvs...)
	return func(kv attribute.KeyValue) bool {
		return !filter.HasValue(kv.Key)
	}
}

func viewStreamAggregationToAggregation(agg *ViewStreamAggregation) sdkmetric.Aggregation {
	if agg == nil {
		return sdkmetric.AggregationDefault{}
	}
	if agg.Sum != nil {
		return sdkmetric.AggregationSum{}
	}
	if agg.Drop != nil {
		return sdkmetric.AggregationDrop{}
	}
	if agg.LastValue != nil {
		return sdkmetric.AggregationLastValue{}
	}
	if agg.ExplicitBucketHistogram != nil {
		return sdkmetric.AggregationExplicitBucketHistogram{
			Boundaries: agg.ExplicitBucketHistogram.Boundaries,
			NoMinMax:   !agg.ExplicitBucketHistogram.RecordMinMaxBool(),
		}
	}
	return sdkmetric.AggregationDefault{}
}
