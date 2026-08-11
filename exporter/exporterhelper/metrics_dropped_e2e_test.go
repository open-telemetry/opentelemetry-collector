// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
)

// TestMetrics_DroppedItemsErr_EndToEnd exercises the full public exporterhelper
// metrics path with a push function that returns NewDroppedItemsErr. It is the
// public-API contract test for the feature: the caller of ConsumeMetrics must
// see no error, exporter_sent_metric_points must reflect (total - dropped),
// exporter_dropped_metric_points must carry the dropped count plus the reason
// attribute, and exporter_send_failed_metric_points must remain unrecorded.
func TestMetrics_DroppedItemsErr_EndToEnd(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	const droppedItems = 2
	const reason = "incompatible temporality"

	pusher := consumer.ConsumeMetricsFunc(func(_ context.Context, _ pmetric.Metrics) error {
		return NewDroppedItemsErr(droppedItems, reason)
	})

	me, err := NewMetrics(
		context.Background(),
		exporter.Settings{
			ID:                fakeMetricsName,
			TelemetrySettings: tt.NewTelemetrySettings(),
			BuildInfo:         component.NewDefaultBuildInfo(),
		},
		&fakeMetricsConfig,
		pusher,
	)
	require.NoError(t, err)
	require.NotNil(t, me)
	require.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, me.Shutdown(context.Background())) })

	// testdata.GenerateMetrics(2) produces 4 metric data points across 2 resources.
	md := testdata.GenerateMetrics(2)
	totalItems := md.DataPointCount()
	require.Greater(t, totalItems, droppedItems, "test fixture must produce more items than we drop")

	// ConsumeMetrics must not surface the sentinel as an error.
	require.NoError(t, me.ConsumeMetrics(context.Background(), md))

	metadatatest.AssertEqualExporterSentMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ExporterKey, fakeMetricsName.String())),
				Value: int64(totalItems - droppedItems),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	metadatatest.AssertEqualExporterDroppedMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(internal.ExporterKey, fakeMetricsName.String()),
					attribute.String(internal.DroppedReasonKey, reason),
				),
				Value: int64(droppedItems),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	_, getErr := tt.GetMetric("otelcol_exporter_send_failed_metric_points")
	require.Error(t, getErr, "send_failed_metric_points must not be recorded when items are dropped")
}

// TestMetrics_DroppedItemsErr_TypeAlias verifies the public DroppedItemsErr is
// the same type as the internal sentinel so errors.As round-trips work for
// exporter authors who type-assert on the public symbol.
func TestMetrics_DroppedItemsErr_TypeAlias(t *testing.T) {
	err := NewDroppedItemsErr(7, "reason")
	var d *DroppedItemsErr
	require.ErrorAs(t, err, &d)
	assert.Equal(t, 7, d.Dropped)
	assert.Equal(t, "reason", d.Reason)
}
