// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

func TestExporterObsMetricsAttributes(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	id := component.NewID(component.MustNewType("test"))
	om, err := newExporterObsMetrics(
		tt.NewTelemetrySettings(),
		id,
		pipeline.SignalTraces,
		[]attribute.KeyValue{attribute.String("transport", "grpc")},
	)
	require.NoError(t, err)
	t.Cleanup(om.Shutdown)

	om.RecordBatchSendSize(context.Background(), 5, 100)
	om.RecordSent(context.Background(), 5)

	batchMetric, err := tt.GetMetric("otelcol_exporter_queue_batch_send_size")
	require.NoError(t, err)
	batch := batchMetric.Data.(metricdata.Histogram[int64])
	require.Equal(t, attribute.NewSet(
		attribute.String(ExporterKey, id.String()),
	), batch.DataPoints[0].Attributes)

	sentMetric, err := tt.GetMetric("otelcol_exporter_sent_spans")
	require.NoError(t, err)
	sent := sentMetric.Data.(metricdata.Sum[int64])
	require.Equal(t, attribute.NewSet(
		attribute.String(ExporterKey, id.String()),
		attribute.String("transport", "grpc"),
	), sent.DataPoints[0].Attributes)
}

type fakeObsMetrics struct {
	ObsMetrics
	shutdowns int
}

func (f *fakeObsMetrics) RegisterQueueSize(queue.QueueObserver) error     { return nil }
func (f *fakeObsMetrics) RegisterQueueCapacity(queue.QueueObserver) error { return nil }
func (f *fakeObsMetrics) RecordEnqueueFailure(context.Context, int64)     {}
func (f *fakeObsMetrics) RecordBatchSendSize(context.Context, int64, int64) {
}
func (f *fakeObsMetrics) RecordInFlight(context.Context, int64) {}
func (f *fakeObsMetrics) RecordSent(context.Context, int64)     {}
func (f *fakeObsMetrics) RecordSendFailure(context.Context, int64, ...metric.AddOption) {
}
func (f *fakeObsMetrics) Shutdown() { f.shutdowns++ }

// TestBaseExporterShutsDownObsMetricsOnConstructionFailure covers the ownership
// contract documented on WithObsMetrics: exporterhelper releases an injected
// ObsMetrics when construction fails after the options are applied.
func TestBaseExporterShutsDownObsMetricsOnConstructionFailure(t *testing.T) {
	om := &fakeObsMetrics{}

	// WithQueue without WithQueueBatchSettings fails after options are applied.
	_, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithObsMetrics(om),
		WithQueue(configoptional.Some(NewDefaultQueueConfig())))
	require.Error(t, err)
	require.Equal(t, 1, om.shutdowns)
}

func TestBaseExporterShutsDownObsMetricsOnShutdown(t *testing.T) {
	om := &fakeObsMetrics{}

	be, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithObsMetrics(om))
	require.NoError(t, err)
	require.Equal(t, 0, om.shutdowns)
	require.NoError(t, be.Shutdown(context.Background()))
	require.Equal(t, 1, om.shutdowns)
}

func TestWithObsMetricsRejectsNil(t *testing.T) {
	require.ErrorContains(t, WithObsMetrics(nil)(&BaseExporter{}), "must not be nil")
}

func TestNewObsReportSenderRejectsNilObsMetrics(t *testing.T) {
	_, err := newObsReportSender(
		exportertest.NewNopSettings(exportertest.NopType),
		pipeline.SignalTraces,
		nil,
		sender.NewSender(func(context.Context, request.Request) error { return nil }),
	)
	require.ErrorContains(t, err, "must not be nil")
}

// TestDefaultObsMetricsCoversQueueAndSender is a regression test for the
// consolidation of the queue and sender telemetry builders into one: both sets
// of instruments must still be reported, and shut down together.
func TestDefaultObsMetricsCoversQueueAndSender(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set := exportertest.NewNopSettings(exportertest.NopType)
	set.TelemetrySettings = tt.NewTelemetrySettings()

	be, err := NewBaseExporter(set, pipeline.SignalTraces, noopExport,
		WithQueueBatchSettings(newFakeQueueBatch()),
		WithQueue(configoptional.Some(NewDefaultQueueConfig())))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 3}))

	// Queue-owned instruments.
	_, err = tt.GetMetric("otelcol_exporter_queue_size")
	require.NoError(t, err)
	_, err = tt.GetMetric("otelcol_exporter_queue_capacity")
	require.NoError(t, err)
	_, err = tt.GetMetric("otelcol_exporter_queue_batch_send_size")
	require.NoError(t, err)

	require.NoError(t, be.Shutdown(context.Background()))

	// Sender-owned instruments.
	_, err = tt.GetMetric("otelcol_exporter_sent_spans")
	require.NoError(t, err)
}
