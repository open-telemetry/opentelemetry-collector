// Code generated by mdatagen. DO NOT EDIT.

package metadatatest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type Telemetry struct {
	componenttest.Telemetry
}

func SetupTelemetry() Telemetry {
	return Telemetry{Telemetry: componenttest.NewTelemetry()}
}

func (tt *Telemetry) NewSettings() receiver.Settings {
	set := receivertest.NewNopSettings()
	set.ID = component.NewID(component.MustNewType("sample"))
	set.TelemetrySettings = tt.NewTelemetrySettings()
	return set
}

func (tt *Telemetry) AssertMetrics(t *testing.T, expected []metricdata.Metrics, opts ...metricdatatest.Option) {
	var md metricdata.ResourceMetrics
	require.NoError(t, tt.Reader.Collect(context.Background(), &md))
	// ensure all required metrics are present
	for _, want := range expected {
		got := getMetricFromResource(want.Name, md)
		metricdatatest.AssertEqual(t, want, got, opts...)
	}

	// ensure no additional metrics are emitted
	require.Equal(t, len(expected), lenMetrics(md))
}

func AssertEqualBatchSizeTriggerSend(t *testing.T, tt componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_batch_size_trigger_send",
		Description: "Number of times the batch was sent due to a size trigger",
		Unit:        "{times}",
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got := getMetric(t, tt, "otelcol_batch_size_trigger_send")
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualProcessRuntimeTotalAllocBytes(t *testing.T, tt componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_process_runtime_total_alloc_bytes",
		Description: "Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')",
		Unit:        "By",
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got := getMetric(t, tt, "otelcol_process_runtime_total_alloc_bytes")
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualQueueCapacity(t *testing.T, tt componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_queue_capacity",
		Description: "Queue capacity - sync gauge example.",
		Unit:        "{items}",
		Data: metricdata.Gauge[int64]{
			DataPoints: dps,
		},
	}
	got := getMetric(t, tt, "otelcol_queue_capacity")
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualQueueLength(t *testing.T, tt componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_queue_length",
		Description: "This metric is optional and therefore not initialized in NewTelemetryBuilder.",
		Unit:        "{items}",
		Data: metricdata.Gauge[int64]{
			DataPoints: dps,
		},
	}
	got := getMetric(t, tt, "otelcol_queue_length")
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualRequestDuration(t *testing.T, tt componenttest.Telemetry, dps []metricdata.HistogramDataPoint[float64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_request_duration",
		Description: "Duration of request",
		Unit:        "s",
		Data: metricdata.Histogram[float64]{
			Temporality: metricdata.CumulativeTemporality,
			DataPoints:  dps,
		},
	}
	got := getMetric(t, tt, "otelcol_request_duration")
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func getMetric(t *testing.T, tt componenttest.Telemetry, name string) metricdata.Metrics {
	var md metricdata.ResourceMetrics
	require.NoError(t, tt.Reader.Collect(context.Background(), &md))
	return getMetricFromResource(name, md)
}

func getMetricFromResource(name string, got metricdata.ResourceMetrics) metricdata.Metrics {
	for _, sm := range got.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}

	return metricdata.Metrics{}
}

func lenMetrics(got metricdata.ResourceMetrics) int {
	metricsCount := 0
	for _, sm := range got.ScopeMetrics {
		metricsCount += len(sm.Metrics)
	}

	return metricsCount
}
