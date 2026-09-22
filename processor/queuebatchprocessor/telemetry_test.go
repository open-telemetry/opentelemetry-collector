// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadatatest"
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
	set := metadatatest.NewSettings(tt)

	obsMetrics, err := newObsMetrics(
		set.TelemetrySettings,
		set.ID,
		pipeline.SignalLogs,
		exporterhelper.RequestSizerTypeBytes,
	)
	require.NoError(t, err)

	ctx := context.Background()
	bytesCalls := 0
	obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricEnqueueFailure, 2)
	obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricEnqueueSize, 3)
	if obsMetrics.ShouldRecord(ctx, queuebatchtelemetry.MetricEnqueueSizeBytes) {
		bytesCalls++
		obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricEnqueueSizeBytes, 30)
	}
	require.NoError(t, obsMetrics.RegisterInt(queuebatchtelemetry.MetricQueueSize, func() int64 { return 7 }))
	require.NoError(t, obsMetrics.RegisterInt(queuebatchtelemetry.MetricQueueCapacity, func() int64 { return 9 }))
	obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricBatchSendSize, 4)
	if obsMetrics.ShouldRecord(ctx, queuebatchtelemetry.MetricBatchSendSizeBytes) {
		bytesCalls++
		obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricBatchSendSizeBytes, 40)
	}
	obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricInFlight, 2)
	obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricInFlight, -1)
	obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricSent, 5)
	obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricSendFailure, 6, metric.WithAttributes(
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

func TestNewProcessorReleasesMetricsOnError(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })
	set := metadatatest.NewSettings(tt)
	wantErr := errors.New("create failed")

	_, err := newProcessor(set, pipeline.SignalTraces, exporterhelper.RequestSizerTypeRequests,
		func(metrics queuebatchtelemetry.ObsMetrics) (struct{}, error) {
			require.NoError(t, metrics.RegisterInt(queuebatchtelemetry.MetricQueueSize, func() int64 { return 1 }))
			require.NoError(t, metrics.RegisterInt(queuebatchtelemetry.MetricQueueCapacity, func() int64 { return 2 }))
			return struct{}{}, wantErr
		})
	require.ErrorIs(t, err, wantErr)
	_, err = tt.GetMetric("otelcol_processor_queuebatch_queue_size")
	require.Error(t, err, "construction failure must unregister queue observers")
}
