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

	require.NoError(t, om.RegisterQueueSize(func() int64 { return 7 }))
	require.NoError(t, om.RegisterQueueCapacity(func() int64 { return 9 }))
	ctx := context.Background()
	om.RecordEnqueueFailure(ctx, 5)
	om.RecordEnqueueSize(ctx, 5, func() int64 { return 100 })
	om.RecordBatchSendSize(ctx, 5, func() int64 { return 100 })
	om.RecordInFlight(ctx, 1)
	om.RecordSent(ctx, 5)
	om.RecordSendFailure(ctx, 5)

	exporterAttr := attribute.String(ExporterKey, id.String())
	signalAttr := attribute.String(DataTypeKey, pipeline.SignalTraces.String())
	extraAttr := attribute.String("transport", "grpc")

	// The signal is an attribute only for instruments whose name omits it, and
	// the extra attributes appear only on those measuring the destination.
	for name, want := range map[string]attribute.Set{
		"otelcol_exporter_enqueue_failed_spans":        attribute.NewSet(exporterAttr),
		"otelcol_exporter_enqueue_size":                attribute.NewSet(exporterAttr, signalAttr),
		"otelcol_exporter_enqueue_size_bytes":          attribute.NewSet(exporterAttr, signalAttr),
		"otelcol_exporter_queue_batch_send_size":       attribute.NewSet(exporterAttr, signalAttr, extraAttr),
		"otelcol_exporter_queue_batch_send_size_bytes": attribute.NewSet(exporterAttr, signalAttr, extraAttr),
		"otelcol_exporter_queue_size":                  attribute.NewSet(exporterAttr, signalAttr),
		"otelcol_exporter_queue_capacity":              attribute.NewSet(exporterAttr, signalAttr),
		"otelcol_exporter_in_flight_requests":          attribute.NewSet(exporterAttr, signalAttr, extraAttr),
		"otelcol_exporter_sent_spans":                  attribute.NewSet(exporterAttr, extraAttr),
		"otelcol_exporter_send_failed_spans":           attribute.NewSet(exporterAttr, extraAttr),
	} {
		require.Equal(t, want, dataPointAttributes(t, tt, name), name)
	}
}

// dataPointAttributes returns the attributes of the metric's first data point.
func dataPointAttributes(t *testing.T, tt *componenttest.Telemetry, name string) attribute.Set {
	t.Helper()
	m, err := tt.GetMetric(name)
	require.NoError(t, err)
	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		return data.DataPoints[0].Attributes
	case metricdata.Gauge[int64]:
		return data.DataPoints[0].Attributes
	case metricdata.Histogram[int64]:
		return data.DataPoints[0].Attributes
	default:
		t.Fatalf("unexpected data type for %q", name)
		return attribute.Set{}
	}
}

// The enqueue failure counter is signal specific; the queue observers carry the data type.
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
func countingObsMetrics(shutdowns *int) ObsMetrics {
	return NewObsMetrics(nil, nil, nil, nil, func() { *shutdowns++ })
}

// A failed construction leaves an injected ObsMetrics for the caller to shut down.
func TestBaseExporterLeavesInjectedObsMetricsOnConstructionFailure(t *testing.T) {
	shutdowns := 0

	// WithQueue without WithQueueBatchSettings fails after options are applied.
	_, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithObsMetrics(countingObsMetrics(&shutdowns)),
		WithQueue(configoptional.Some(NewDefaultQueueConfig())))
	require.Error(t, err)
	require.Equal(t, 0, shutdowns)
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
		false,
		sender.NewSender(func(context.Context, request.Request) error { return nil }),
	)
	require.ErrorContains(t, err, "must not be nil")
}

// Regression test: one telemetry builder must still report both queue and sender instruments.
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
	_, err = tt.GetMetric("otelcol_exporter_enqueue_size")
	require.NoError(t, err)

	require.NoError(t, be.Shutdown(context.Background()))

	// Sender-owned instruments.
	_, err = tt.GetMetric("otelcol_exporter_sent_spans")
	require.NoError(t, err)
}

var errRegisterCallback = errors.New("register callback failed")

// failingMeterProvider fails the RegisterCallback call at index failAt (1-based).
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

// A failed observer registration releases the ObsMetrics rather than observing a dead queue.
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
