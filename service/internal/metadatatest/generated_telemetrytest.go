// Code generated by mdatagen. DO NOT EDIT.

package metadatatest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component/componenttest"
)

type Telemetry struct {
	componenttest.Telemetry
}

func SetupTelemetry(opts ...componenttest.TelemetryOption) Telemetry {
	return Telemetry{Telemetry: componenttest.NewTelemetry(opts...)}
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
	require.LessOrEqual(t, len(expected), lenMetrics(md))
}

func AssertEqualProcessCPUSeconds(t *testing.T, tt componenttest.Telemetry, dps []metricdata.DataPoint[float64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_process_cpu_seconds",
		Description: "Total CPU user and system time in seconds [alpha]",
		Unit:        "s",
		Data: metricdata.Sum[float64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got := getMetric(t, tt, "otelcol_process_cpu_seconds")
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualProcessMemoryRss(t *testing.T, tt componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_process_memory_rss",
		Description: "Total physical memory (resident set size) [alpha]",
		Unit:        "By",
		Data: metricdata.Gauge[int64]{
			DataPoints: dps,
		},
	}
	got := getMetric(t, tt, "otelcol_process_memory_rss")
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualProcessRuntimeHeapAllocBytes(t *testing.T, tt componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_process_runtime_heap_alloc_bytes",
		Description: "Bytes of allocated heap objects (see 'go doc runtime.MemStats.HeapAlloc') [alpha]",
		Unit:        "By",
		Data: metricdata.Gauge[int64]{
			DataPoints: dps,
		},
	}
	got := getMetric(t, tt, "otelcol_process_runtime_heap_alloc_bytes")
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualProcessRuntimeTotalAllocBytes(t *testing.T, tt componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_process_runtime_total_alloc_bytes",
		Description: "Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc') [alpha]",
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

func AssertEqualProcessRuntimeTotalSysMemoryBytes(t *testing.T, tt componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_process_runtime_total_sys_memory_bytes",
		Description: "Total bytes of memory obtained from the OS (see 'go doc runtime.MemStats.Sys') [alpha]",
		Unit:        "By",
		Data: metricdata.Gauge[int64]{
			DataPoints: dps,
		},
	}
	got := getMetric(t, tt, "otelcol_process_runtime_total_sys_memory_bytes")
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualProcessUptime(t *testing.T, tt componenttest.Telemetry, dps []metricdata.DataPoint[float64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_process_uptime",
		Description: "Uptime of the process [alpha]",
		Unit:        "s",
		Data: metricdata.Sum[float64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got := getMetric(t, tt, "otelcol_process_uptime")
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
