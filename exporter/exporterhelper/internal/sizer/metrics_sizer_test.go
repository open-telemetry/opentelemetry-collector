// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package sizer // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMetricsCountSizer(t *testing.T) {
	md := testdata.GenerateMetrics(7)
	sizer := MetricsCountSizer{}
	require.Equal(t, 14, sizer.MetricsSize(md))

	rm := md.ResourceMetrics().At(0)
	require.Equal(t, 14, sizer.ResourceMetricsSize(rm))

	sm := rm.ScopeMetrics().At(0)
	require.Equal(t, 14, sizer.ScopeMetricsSize(sm))

	// Test different metric types
	require.Equal(t, 2, sizer.MetricSize(sm.Metrics().At(0)))

	// Test data point sizes
	require.Equal(t, 1, sizer.NumberDataPointSize(sm.Metrics().At(0).Gauge().DataPoints().At(0)))
	require.Equal(t, 1, sizer.NumberDataPointSize(sm.Metrics().At(1).Gauge().DataPoints().At(0)))
	require.Equal(t, 1, sizer.NumberDataPointSize(sm.Metrics().At(2).Sum().DataPoints().At(0)))
	require.Equal(t, 1, sizer.NumberDataPointSize(sm.Metrics().At(3).Sum().DataPoints().At(0)))
	require.Equal(t, 1, sizer.HistogramDataPointSize(sm.Metrics().At(4).Histogram().DataPoints().At(0)))
	require.Equal(t, 1, sizer.ExponentialHistogramDataPointSize(sm.Metrics().At(5).ExponentialHistogram().DataPoints().At(0)))
	require.Equal(t, 1, sizer.SummaryDataPointSize(sm.Metrics().At(6).Summary().DataPoints().At(0)))

	prevSize := sizer.ScopeMetricsSize(sm)
	sm.Metrics().At(0).CopyTo(sm.Metrics().AppendEmpty())
	require.Equal(t, sizer.ScopeMetricsSize(sm), prevSize+sizer.DeltaSize(sizer.MetricSize(sm.Metrics().At(0))))
}

func TestMetricsBytesSizer(t *testing.T) {
	md := testdata.GenerateMetrics(7)
	sizer := MetricsBytesSizer{}
	require.Equal(t, 1594, sizer.MetricsSize(md))

	rm := md.ResourceMetrics().At(0)
	require.Equal(t, 1591, sizer.ResourceMetricsSize(rm))

	sm := rm.ScopeMetrics().At(0)
	require.Equal(t, 1546, sizer.ScopeMetricsSize(sm))

	// Test different metric types
	require.Equal(t, 130, sizer.MetricSize(sm.Metrics().At(0)))

	// Test data point sizes
	require.Equal(t, 55, sizer.NumberDataPointSize(sm.Metrics().At(0).Gauge().DataPoints().At(0)))
	require.Equal(t, 83, sizer.NumberDataPointSize(sm.Metrics().At(1).Gauge().DataPoints().At(0)))
	require.Equal(t, 55, sizer.NumberDataPointSize(sm.Metrics().At(2).Sum().DataPoints().At(0)))
	require.Equal(t, 83, sizer.NumberDataPointSize(sm.Metrics().At(3).Sum().DataPoints().At(0)))
	require.Equal(t, 92, sizer.HistogramDataPointSize(sm.Metrics().At(4).Histogram().DataPoints().At(0)))
	require.Equal(t, 119, sizer.ExponentialHistogramDataPointSize(sm.Metrics().At(5).ExponentialHistogram().DataPoints().At(0)))
	require.Equal(t, 92, sizer.SummaryDataPointSize(sm.Metrics().At(6).Summary().DataPoints().At(0)))

	prevSize := sizer.ScopeMetricsSize(sm)
	sm.Metrics().At(0).CopyTo(sm.Metrics().AppendEmpty())
	require.Equal(t, sizer.ScopeMetricsSize(sm), prevSize+sizer.DeltaSize(sizer.MetricSize(sm.Metrics().At(0))))
}
