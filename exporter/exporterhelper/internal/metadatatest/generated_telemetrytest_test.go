// Code generated by mdatagen. DO NOT EDIT.

package metadatatest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
)

func TestSetupTelemetry(t *testing.T) {
	testTel := SetupTelemetry()
	tb, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)
	defer tb.Shutdown()
	require.NoError(t, tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, observer metric.Int64Observer) error {
		observer.Observe(1)
		return nil
	}))
	require.NoError(t, tb.RegisterExporterQueueSizeCallback(func(_ context.Context, observer metric.Int64Observer) error {
		observer.Observe(1)
		return nil
	}))
	tb.ExporterEnqueueFailedLogRecords.Add(context.Background(), 1)
	tb.ExporterEnqueueFailedMetricPoints.Add(context.Background(), 1)
	tb.ExporterEnqueueFailedSpans.Add(context.Background(), 1)
	tb.ExporterSendFailedLogRecords.Add(context.Background(), 1)
	tb.ExporterSendFailedMetricPoints.Add(context.Background(), 1)
	tb.ExporterSendFailedSpans.Add(context.Background(), 1)
	tb.ExporterSentLogRecords.Add(context.Background(), 1)
	tb.ExporterSentMetricPoints.Add(context.Background(), 1)
	tb.ExporterSentSpans.Add(context.Background(), 1)

	testTel.AssertMetrics(t, []metricdata.Metrics{
		{
			Name:        "otelcol_exporter_enqueue_failed_log_records",
			Description: "Number of log records failed to be added to the sending queue. [alpha]",
			Unit:        "{records}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_exporter_enqueue_failed_metric_points",
			Description: "Number of metric points failed to be added to the sending queue. [alpha]",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_exporter_enqueue_failed_spans",
			Description: "Number of spans failed to be added to the sending queue. [alpha]",
			Unit:        "{spans}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_exporter_queue_capacity",
			Description: "Fixed capacity of the retry queue (in batches) [alpha]",
			Unit:        "{batches}",
			Data: metricdata.Gauge[int64]{
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_exporter_queue_size",
			Description: "Current size of the retry queue (in batches) [alpha]",
			Unit:        "{batches}",
			Data: metricdata.Gauge[int64]{
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_exporter_send_failed_log_records",
			Description: "Number of log records in failed attempts to send to destination. [alpha]",
			Unit:        "{records}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_exporter_send_failed_metric_points",
			Description: "Number of metric points in failed attempts to send to destination. [alpha]",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_exporter_send_failed_spans",
			Description: "Number of spans in failed attempts to send to destination. [alpha]",
			Unit:        "{spans}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_exporter_sent_log_records",
			Description: "Number of log record successfully sent to destination. [alpha]",
			Unit:        "{records}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_exporter_sent_metric_points",
			Description: "Number of metric points successfully sent to destination. [alpha]",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
		{
			Name:        "otelcol_exporter_sent_spans",
			Description: "Number of spans successfully sent to destination. [alpha]",
			Unit:        "{spans}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{},
				},
			},
		},
	}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
	AssertEqualExporterEnqueueFailedLogRecords(t, testTel.Telemetry,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualExporterEnqueueFailedMetricPoints(t, testTel.Telemetry,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualExporterEnqueueFailedSpans(t, testTel.Telemetry,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualExporterQueueCapacity(t, testTel.Telemetry,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualExporterQueueSize(t, testTel.Telemetry,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualExporterSendFailedLogRecords(t, testTel.Telemetry,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualExporterSendFailedMetricPoints(t, testTel.Telemetry,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualExporterSendFailedSpans(t, testTel.Telemetry,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualExporterSentLogRecords(t, testTel.Telemetry,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualExporterSentMetricPoints(t, testTel.Telemetry,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())
	AssertEqualExporterSentSpans(t, testTel.Telemetry,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp())

	require.NoError(t, testTel.Shutdown(context.Background()))
}
