// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
)

func TestContextWithAttributes_Tracer(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := componentattribute.TracerProviderWithAttributes(
		sdkTrace.NewTracerProvider(sdkTrace.WithSpanProcessor(sdkTrace.NewSimpleSpanProcessor(exporter))),
		attribute.NewSet(),
	)
	tracer := provider.Tracer("test")

	ctx1 := componentattribute.ContextWithAttributes(context.Background(),
		attribute.String("k1", "k1_ctx1"),
		attribute.String("k2", "k2_ctx1"),
	)
	ctx2 := componentattribute.ContextWithAttributes(ctx1,
		attribute.String("k1", "k1_ctx2"),
		attribute.String("k3", "k3_ctx2"),
		attribute.String("k4", "k4_ctx2"),
	)

	tests := []struct {
		name          string
		ctx           context.Context
		expectedAttrs []attribute.KeyValue
	}{
		{
			name: "background_context",
			ctx:  context.Background(),
			expectedAttrs: []attribute.KeyValue{
				attribute.String("k4", "span"),
			},
		},
		{
			name: "context_no_attrs",
			ctx:  componentattribute.ContextWithAttributes(context.Background()),
			expectedAttrs: []attribute.KeyValue{
				attribute.String("k4", "span"),
			},
		},
		{
			name: "context_layered_attrs",
			ctx:  ctx2,
			expectedAttrs: []attribute.KeyValue{
				attribute.String("k1", "k1_ctx2"),
				attribute.String("k2", "k2_ctx1"),
				attribute.String("k3", "k3_ctx2"),
				attribute.String("k4", "span"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, span := tracer.Start(tt.ctx, "span", trace.WithAttributes(
				attribute.String("k4", "span"),
			))
			span.End()

			spans := exporter.GetSpans()
			spanStub := spans[len(spans)-1]
			assert.Equal(t, tt.expectedAttrs, spanStub.Attributes)
		})
	}
}

func TestContextWithAttributes_Meter(t *testing.T) {
	tests := []struct {
		name     string
		execTest func(t *testing.T, ctx context.Context, meter metric.Meter, attrs ...attribute.KeyValue)
	}{
		{
			name: "Int64Counter",
			execTest: func(
				t *testing.T, ctx context.Context,
				meter metric.Meter, attrs ...attribute.KeyValue,
			) {
				counter, err := meter.Int64Counter("counter")
				require.NoError(t, err)
				counter.Add(ctx, 1, metric.WithAttributes(attrs...))
			},
		},
		{
			name: "Int64UpDownCounter",
			execTest: func(
				t *testing.T, ctx context.Context,
				meter metric.Meter, attrs ...attribute.KeyValue,
			) {
				counter, err := meter.Int64UpDownCounter("updowncounter")
				require.NoError(t, err)
				counter.Add(ctx, 1, metric.WithAttributes(attrs...))
			},
		},
		{
			name: "Int64Histogram",
			execTest: func(
				t *testing.T, ctx context.Context,
				meter metric.Meter, attrs ...attribute.KeyValue,
			) {
				hist, err := meter.Int64Histogram("histogram")
				require.NoError(t, err)
				hist.Record(ctx, 10, metric.WithAttributes(attrs...))
			},
		},
		{
			name: "Int64Gauge",
			execTest: func(
				t *testing.T, ctx context.Context,
				meter metric.Meter, attrs ...attribute.KeyValue,
			) {
				gauge, err := meter.Int64Gauge("gauge")
				require.NoError(t, err)
				gauge.Record(ctx, 5, metric.WithAttributes(attrs...))
			},
		},
		{
			name: "Float64Counter",
			execTest: func(
				t *testing.T, ctx context.Context,
				meter metric.Meter, attrs ...attribute.KeyValue,
			) {
				counter, err := meter.Float64Counter("floatcounter")
				require.NoError(t, err)
				counter.Add(ctx, 1.5, metric.WithAttributes(attrs...))
			},
		},
		{
			name: "Float64UpDownCounter",
			execTest: func(
				t *testing.T, ctx context.Context,
				meter metric.Meter, attrs ...attribute.KeyValue,
			) {
				counter, err := meter.Float64UpDownCounter("floatupdowncounter")
				require.NoError(t, err)
				counter.Add(ctx, 1.5, metric.WithAttributes(attrs...))
			},
		},
		{
			name: "Float64Histogram",
			execTest: func(
				t *testing.T, ctx context.Context,
				meter metric.Meter, attrs ...attribute.KeyValue,
			) {
				hist, err := meter.Float64Histogram("floathistogram")
				require.NoError(t, err)
				hist.Record(ctx, 10.5, metric.WithAttributes(attrs...))
			},
		},
		{
			name: "Float64Gauge",
			execTest: func(
				t *testing.T, ctx context.Context,
				meter metric.Meter, attrs ...attribute.KeyValue,
			) {
				gauge, err := meter.Float64Gauge("floatgauge")
				require.NoError(t, err)
				gauge.Record(ctx, 5.5, metric.WithAttributes(attrs...))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := sdkMetric.NewManualReader()
			provider := componentattribute.MeterProviderWithAttributes(
				sdkMetric.NewMeterProvider(sdkMetric.WithReader(reader)),
				attribute.NewSet(),
			)
			meter := provider.Meter("test")

			ctx1 := componentattribute.ContextWithAttributes(context.Background(),
				attribute.String("k1", "k1_ctx1"),
				attribute.String("k2", "k2_ctx1"),
			)
			ctx2 := componentattribute.ContextWithAttributes(ctx1,
				attribute.String("k1", "k1_ctx2"),
				attribute.String("k3", "k3_ctx2"),
				attribute.String("k4", "k4_ctx2"),
			)
			tt.execTest(t, ctx2, meter, attribute.String("k4", "measurement"))

			expectedAttrs := []attribute.KeyValue{
				attribute.String("k1", "k1_ctx2"),
				attribute.String("k2", "k2_ctx1"),
				attribute.String("k3", "k3_ctx2"),
				attribute.String("k4", "measurement"),
			}

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(context.Background(), &rm))

			// Get the first metric and verify its data points
			require.NotEmpty(t, rm.ScopeMetrics, "No scope metrics recorded")
			scopeMetrics := rm.ScopeMetrics[0]
			require.NotEmpty(t, scopeMetrics.Metrics, "No metrics recorded")
			metricData := scopeMetrics.Metrics[0]

			// Handle different metric types
			switch data := metricData.Data.(type) {
			case metricdata.Sum[int64]:
				require.NotEmpty(t, data.DataPoints, "No data points recorded")
				assert.ElementsMatch(t, expectedAttrs, data.DataPoints[0].Attributes.ToSlice())

			case metricdata.Sum[float64]:
				require.NotEmpty(t, data.DataPoints, "No data points recorded")
				assert.ElementsMatch(t, expectedAttrs, data.DataPoints[0].Attributes.ToSlice())

			case metricdata.Histogram[int64]:
				require.NotEmpty(t, data.DataPoints, "No data points recorded")
				assert.ElementsMatch(t, expectedAttrs, data.DataPoints[0].Attributes.ToSlice())

			case metricdata.Histogram[float64]:
				require.NotEmpty(t, data.DataPoints, "No data points recorded")
				assert.ElementsMatch(t, expectedAttrs, data.DataPoints[0].Attributes.ToSlice())

			case metricdata.Gauge[int64]:
				require.NotEmpty(t, data.DataPoints, "No data points recorded")
				assert.ElementsMatch(t, expectedAttrs, data.DataPoints[0].Attributes.ToSlice())

			case metricdata.Gauge[float64]:
				require.NotEmpty(t, data.DataPoints, "No data points recorded")
				assert.ElementsMatch(t, expectedAttrs, data.DataPoints[0].Attributes.ToSlice())

			default:
				t.Fatalf("Unhandled metric data type: %T", metricData.Data)
			}
		})
	}
}
