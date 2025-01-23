// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proctelemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/service/internal/metadatatest"
)

func TestProcessTelemetry(t *testing.T) {
	tel := metadatatest.SetupTelemetry()
	require.NoError(t, RegisterProcessMetrics(tel.NewTelemetrySettings()))
	tel.AssertMetrics(t, []metricdata.Metrics{
		{
			Name:        "otelcol_process_uptime",
			Description: "Uptime of the process [alpha]",
			Unit:        "s",
			Data: metricdata.Sum[float64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[float64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_process_runtime_heap_alloc_bytes",
			Description: "Bytes of allocated heap objects (see 'go doc runtime.MemStats.HeapAlloc') [alpha]",
			Unit:        "By",
			Data: metricdata.Gauge[int64]{
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_process_runtime_total_alloc_bytes",
			Description: "Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc') [alpha]",
			Unit:        "By",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_process_runtime_total_sys_memory_bytes",
			Description: "Total bytes of memory obtained from the OS (see 'go doc runtime.MemStats.Sys') [alpha]",
			Unit:        "By",
			Data: metricdata.Gauge[int64]{
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_process_cpu_seconds",
			Description: "Total CPU user and system time in seconds [alpha]",
			Unit:        "s",
			Data: metricdata.Sum[float64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[float64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_process_memory_rss",
			Description: "Total physical memory (resident set size) [alpha]",
			Unit:        "By",
			Data: metricdata.Gauge[int64]{
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
	}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}
