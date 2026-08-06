// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
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

// TestExporterObsMetricsQueueInstruments covers the queue-oriented instruments
// for every signal: the enqueue failure counter is signal specific, while the
// queue size and capacity observers are shared and carry the data type
// attribute.
func TestExporterObsMetricsQueueInstruments(t *testing.T) {
	for _, tt := range []struct {
		name      string
		signal    pipeline.Signal
		assertErr func(*testing.T, *componenttest.Telemetry, []metricdata.DataPoint[int64], ...metricdatatest.Option)
	}{
		{"traces", pipeline.SignalTraces, metadatatest.AssertEqualExporterEnqueueFailedSpans},
		{"metrics", pipeline.SignalMetrics, metadatatest.AssertEqualExporterEnqueueFailedMetricPoints},
		{"logs", pipeline.SignalLogs, metadatatest.AssertEqualExporterEnqueueFailedLogRecords},
		{"profiles", xpipeline.SignalProfiles, metadatatest.AssertEqualExporterEnqueueFailedProfileSamples},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tel := componenttest.NewTelemetry()
			t.Cleanup(func() { require.NoError(t, tel.Shutdown(context.Background())) })

			id := component.NewID(exportertest.NopType)
			om, err := newExporterObsMetrics(tel.NewTelemetrySettings(), id, tt.signal, nil)
			require.NoError(t, err)
			t.Cleanup(om.Shutdown)

			require.NoError(t, om.RegisterQueueSize(func() int64 { return 7 }))
			require.NoError(t, om.RegisterQueueCapacity(func() int64 { return 9 }))
			om.RecordEnqueueFailure(context.Background(), 12)

			exporterAttrs := attribute.NewSet(attribute.String(ExporterKey, id.String()))
			asyncAttrs := attribute.NewSet(
				attribute.String(ExporterKey, id.String()),
				attribute.String(DataTypeKey, tt.signal.String()),
			)

			tt.assertErr(t, tel, []metricdata.DataPoint[int64]{{Attributes: exporterAttrs, Value: 12}},
				metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
			metadatatest.AssertEqualExporterQueueSize(t, tel,
				[]metricdata.DataPoint[int64]{{Attributes: asyncAttrs, Value: 7}}, metricdatatest.IgnoreTimestamp())
			metadatatest.AssertEqualExporterQueueCapacity(t, tel,
				[]metricdata.DataPoint[int64]{{Attributes: asyncAttrs, Value: 9}}, metricdatatest.IgnoreTimestamp())
		})
	}
}

// countingObsMetrics reports nothing but counts how often it is shut down.
func countingObsMetrics(shutdowns *int) *ObsMetrics {
	return NewObsMetrics(ObsMetricsConfig{Shutdown: func() { *shutdowns++ }})
}

// TestBaseExporterShutsDownObsMetricsOnConstructionFailure covers the ownership
// contract documented on WithObsMetrics: exporterhelper releases an injected
// ObsMetrics when construction fails after the options are applied.
func TestBaseExporterShutsDownObsMetricsOnConstructionFailure(t *testing.T) {
	shutdowns := 0

	// WithQueue without WithQueueBatchSettings fails after options are applied.
	_, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithObsMetrics(countingObsMetrics(&shutdowns)),
		WithQueue(configoptional.Some(NewDefaultQueueConfig())))
	require.Error(t, err)
	require.Equal(t, 1, shutdowns)
}

func TestBaseExporterShutsDownObsMetricsOnShutdown(t *testing.T) {
	shutdowns := 0

	be, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithObsMetrics(countingObsMetrics(&shutdowns)))
	require.NoError(t, err)
	require.Equal(t, 0, shutdowns)
	require.NoError(t, be.Shutdown(context.Background()))
	require.Equal(t, 1, shutdowns)
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

var errRegisterCallback = errors.New("register callback failed")

// failingMeterProvider delegates to a real MeterProvider but fails the
// RegisterCallback call at index failAt (1-based).
type failingMeterProvider struct {
	metric.MeterProvider
	calls  *int
	failAt int
}

func (p failingMeterProvider) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	return failingMeter{Meter: p.MeterProvider.Meter(name, opts...), calls: p.calls, failAt: p.failAt}
}

type failingMeter struct {
	metric.Meter
	calls  *int
	failAt int
}

func (m failingMeter) RegisterCallback(f metric.Callback, instruments ...metric.Observable) (metric.Registration, error) {
	*m.calls++
	if *m.calls == m.failAt {
		return nil, errRegisterCallback
	}
	return m.Meter.RegisterCallback(f, instruments...)
}

// TestObsMetricsReleasedOnQueueRegistrationFailure verifies that the owner of
// the ObsMetrics releases them when the queue fails to register one of its
// observers, instead of leaving an already-registered callback observing a
// dead queue.
func TestObsMetricsReleasedOnQueueRegistrationFailure(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set := exportertest.NewNopSettings(exportertest.NopType)
	set.TelemetrySettings = tt.NewTelemetrySettings()
	calls := 0
	// The queue capacity callback is registered second.
	set.MeterProvider = failingMeterProvider{MeterProvider: set.MeterProvider, calls: &calls, failAt: 2}

	_, err := NewBaseExporter(set, pipeline.SignalTraces, noopExport,
		WithQueueBatchSettings(newFakeQueueBatch()),
		WithQueue(configoptional.Some(NewDefaultQueueConfig())))
	require.ErrorIs(t, err, errRegisterCallback)

	_, err = tt.GetMetric("otelcol_exporter_queue_size")
	require.Error(t, err, "queue size callback must be unregistered")
}
