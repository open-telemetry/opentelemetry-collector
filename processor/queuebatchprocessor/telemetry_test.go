// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pipeline"
)

func requireSum(t *testing.T, tt *componenttest.Telemetry, name string, value int64, attrs attribute.Set) {
	got, err := tt.GetMetric(name)
	require.NoError(t, err)
	point := got.Data.(metricdata.Sum[int64]).DataPoints[0]
	require.Equal(t, value, point.Value)
	require.Equal(t, attrs, point.Attributes)
}

func requireHistogram(t *testing.T, tt *componenttest.Telemetry, name string, sum int64, attrs attribute.Set) {
	got, err := tt.GetMetric(name)
	require.NoError(t, err)
	point := got.Data.(metricdata.Histogram[int64]).DataPoints[0]
	require.Equal(t, sum, point.Sum)
	require.Equal(t, attrs, point.Attributes)
}

func requireGauge(t *testing.T, tt *componenttest.Telemetry, name string, value int64, attrs attribute.Set) {
	got, err := tt.GetMetric(name)
	require.NoError(t, err)
	point := got.Data.(metricdata.Gauge[int64]).DataPoints[0]
	require.Equal(t, value, point.Value)
	require.Equal(t, attrs, point.Attributes)
}

func TestProcessorMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	cfg.WaitForResult = true
	sink := new(consumertest.TracesSink)
	p, err := newTracesProcessor(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, p.ConsumeTraces(context.Background(), generateTraces(5)))
	require.Equal(t, 5, sink.SpanCount())

	attrs := attribute.NewSet(
		attribute.String(processorKey, set.ID.String()),
		attribute.String(dataTypeKey, pipeline.SignalTraces.String()),
	)
	sent, err := tt.GetMetric("otelcol_processor_queuebatch_sent_items")
	require.NoError(t, err)
	require.Equal(t, int64(5), sent.Data.(metricdata.Sum[int64]).DataPoints[0].Value)
	require.Equal(t, attrs, sent.Data.(metricdata.Sum[int64]).DataPoints[0].Attributes)

	batch, err := tt.GetMetric("otelcol_processor_queuebatch_batch_send_size")
	require.NoError(t, err)
	require.Equal(t, int64(5), batch.Data.(metricdata.Histogram[int64]).DataPoints[0].Sum)

	_, err = tt.GetMetric("otelcol_exporter_sent_spans")
	require.Error(t, err)
	require.NoError(t, p.Shutdown(context.Background()))
	_, err = tt.GetMetric("otelcol_processor_queuebatch_queue_size")
	require.Error(t, err, "shutdown must unregister queue observers")
}

func TestObsMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })
	set, _ := testSettings(tt)

	obsMetrics, err := newObsMetrics(
		set.TelemetrySettings,
		set.ID,
		pipeline.SignalLogs,
		exporterhelper.RequestSizerTypeBytes,
	)
	require.NoError(t, err)

	ctx := context.Background()
	bytesCalls := 0
	obsMetrics.EnqueueFailure(ctx, 2)
	obsMetrics.EnqueueSize(ctx, 3, func() int64 {
		bytesCalls++
		return 30
	})
	require.NoError(t, obsMetrics.RegisterQueue(func() int64 { return 7 }, func() int64 { return 9 }))
	obsMetrics.BatchSendSize(ctx, 4, func() int64 {
		bytesCalls++
		return 40
	})
	obsMetrics.InFlight(ctx, 2)
	obsMetrics.InFlight(ctx, -1)
	obsMetrics.Sent(ctx, 5)
	obsMetrics.SendFailure(ctx, 6, metric.WithAttributes(
		attribute.String("error.type", "test"),
		attribute.Bool("error.permanent", true),
	))
	require.Equal(t, 2, bytesCalls)

	attrs := attribute.NewSet(
		attribute.String(processorKey, set.ID.String()),
		attribute.String(dataTypeKey, pipeline.SignalLogs.String()),
	)
	queueAttrs := attribute.NewSet(
		attribute.String(processorKey, set.ID.String()),
		attribute.String(dataTypeKey, pipeline.SignalLogs.String()),
		attribute.String(sizerKey, exporterhelper.RequestSizerTypeBytes.String()),
	)
	failureAttrs := attribute.NewSet(
		attribute.String(processorKey, set.ID.String()),
		attribute.String(dataTypeKey, pipeline.SignalLogs.String()),
		attribute.String("error.type", "test"),
		attribute.Bool("error.permanent", true),
	)

	requireSum(t, tt, "otelcol_processor_queuebatch_enqueue_failed_items", 2, attrs)
	requireHistogram(t, tt, "otelcol_processor_queuebatch_enqueue_size", 3, attrs)
	requireHistogram(t, tt, "otelcol_processor_queuebatch_enqueue_size_bytes", 30, attrs)
	requireGauge(t, tt, "otelcol_processor_queuebatch_queue_size", 7, queueAttrs)
	requireGauge(t, tt, "otelcol_processor_queuebatch_queue_capacity", 9, queueAttrs)
	requireHistogram(t, tt, "otelcol_processor_queuebatch_batch_send_size", 4, attrs)
	requireHistogram(t, tt, "otelcol_processor_queuebatch_batch_send_size_bytes", 40, attrs)
	requireSum(t, tt, "otelcol_processor_queuebatch_in_flight_requests", 1, attrs)
	requireSum(t, tt, "otelcol_processor_queuebatch_sent_items", 5, attrs)
	requireSum(t, tt, "otelcol_processor_queuebatch_send_failed_items", 6, failureAttrs)

	obsMetrics.Shutdown()
	obsMetrics.Shutdown()
}
