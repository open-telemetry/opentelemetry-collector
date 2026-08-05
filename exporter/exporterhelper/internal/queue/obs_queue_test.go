// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

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
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

var exporterID = component.NewID(exportertest.NopType)

type fakeQueue[T any] struct {
	Queue[T]
	offerErr error
	size     int64
	capacity int64
}

func (fq *fakeQueue[T]) Size() int64 {
	return fq.size
}

func (fq *fakeQueue[T]) Capacity() int64 {
	return fq.capacity
}

func (fq *fakeQueue[T]) Offer(context.Context, T) error {
	return fq.offerErr
}

func newFakeQueue[T request.Request](offerErr error, size, capacity int64) Queue[T] {
	return &fakeQueue[T]{offerErr: offerErr, size: size, capacity: capacity}
}

func TestObsQueueLogsSizeCapacity(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    pipeline.SignalLogs,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](nil, 7, 9))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 2}))
	metadatatest.AssertEqualExporterQueueSize(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
					attribute.String(dataTypeKey, pipeline.SignalLogs.String()),
				),
				Value: int64(7),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualExporterQueueCapacity(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
					attribute.String(dataTypeKey, pipeline.SignalLogs.String()),
				),
				Value: int64(9),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestObsQueueLogsFailure(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    pipeline.SignalLogs,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](errors.New("my error"), 7, 9))
	require.NoError(t, err)
	require.Error(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 2}))
	metadatatest.AssertEqualExporterEnqueueFailedLogRecords(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
				),
				Value: int64(2),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestObsQueueTracesSizeCapacity(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    pipeline.SignalTraces,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](nil, 17, 19))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 12}))
	metadatatest.AssertEqualExporterQueueSize(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
					attribute.String(dataTypeKey, pipeline.SignalTraces.String()),
				),
				Value: int64(17),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualExporterQueueCapacity(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
					attribute.String(dataTypeKey, pipeline.SignalTraces.String()),
				),
				Value: int64(19),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestObsQueueTracesFailure(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    pipeline.SignalTraces,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](errors.New("my error"), 0, 0))
	require.NoError(t, err)
	require.Error(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 12}))
	metadatatest.AssertEqualExporterEnqueueFailedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
				),
				Value: int64(12),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestObsQueueMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    pipeline.SignalMetrics,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](nil, 27, 29))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 22}))
	metadatatest.AssertEqualExporterQueueSize(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
					attribute.String(dataTypeKey, pipeline.SignalMetrics.String()),
				),
				Value: int64(27),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualExporterQueueCapacity(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
					attribute.String(dataTypeKey, pipeline.SignalMetrics.String()),
				),
				Value: int64(29),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestObsQueueMetricsFailure(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    pipeline.SignalMetrics,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](errors.New("my error"), 0, 0))
	require.NoError(t, err)
	require.Error(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 22}))
	metadatatest.AssertEqualExporterEnqueueFailedMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
				),
				Value: int64(22),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestObsQueueProfiles(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    xpipeline.SignalProfiles,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](nil, 27, 29))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 22}))
	metadatatest.AssertEqualExporterQueueSize(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
					attribute.String(dataTypeKey, xpipeline.SignalProfiles.String()),
				),
				Value: int64(27),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualExporterQueueCapacity(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
					attribute.String(dataTypeKey, xpipeline.SignalProfiles.String()),
				),
				Value: int64(29),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestObsQueueProfilesFailure(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    xpipeline.SignalProfiles,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](errors.New("my error"), 0, 0))
	require.NoError(t, err)
	require.Error(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 22}))
	metadatatest.AssertEqualExporterEnqueueFailedProfileSamples(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
				),
				Value: int64(22),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestObsQueueLogsBatchSize(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    pipeline.SignalLogs,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](nil, 7, 9))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 2, Bytes: 100}))
	metadatatest.AssertEqualExporterQueueBatchSendSize(t, tt,
		[]metricdata.HistogramDataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
				),
				Count:        1,
				Bounds:       []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
				BucketCounts: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Min:          metricdata.NewExtrema[int64](2),
				Max:          metricdata.NewExtrema[int64](2),
				Sum:          2,
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestObsQueueTracesBatchSize(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    pipeline.SignalTraces,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](nil, 17, 19))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 12, Bytes: 200}))
	metadatatest.AssertEqualExporterQueueBatchSendSize(t, tt,
		[]metricdata.HistogramDataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
				),
				Count:        1,
				Bounds:       []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
				BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Min:          metricdata.NewExtrema[int64](12),
				Max:          metricdata.NewExtrema[int64](12),
				Sum:          12,
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestObsQueueMetricsBatchSize(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    pipeline.SignalMetrics,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](nil, 27, 29))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 22, Bytes: 300}))
	metadatatest.AssertEqualExporterQueueBatchSendSize(t, tt,
		[]metricdata.HistogramDataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
				),
				Count:        1,
				Bounds:       []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
				BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Min:          metricdata.NewExtrema[int64](22),
				Max:          metricdata.NewExtrema[int64](22),
				Sum:          22,
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestObsQueueProfilesBatchSize(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    xpipeline.SignalProfiles,
		ID:        exporterID,
		Telemetry: tt.NewTelemetrySettings(),
	}, newFakeQueue[request.Request](nil, 27, 29))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 22, Bytes: 300}))
	metadatatest.AssertEqualExporterQueueBatchSendSize(t, tt,
		[]metricdata.HistogramDataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(exporterKey, exporterID.String()),
				),
				Count:        1,
				Bounds:       []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
				BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Min:          metricdata.NewExtrema[int64](22),
				Max:          metricdata.NewExtrema[int64](22),
				Sum:          22,
			},
		}, metricdatatest.IgnoreTimestamp())
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

// TestObsQueueReleasesOwnedTelemetryOnRegistrationFailure verifies that a queue
// owning its telemetry builder releases it when a later registration fails,
// instead of leaving the already-registered callback observing a dead queue.
func TestObsQueueReleasesOwnedTelemetryOnRegistrationFailure(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set := tt.NewTelemetrySettings()
	calls := 0
	// The queue capacity callback is registered second.
	set.MeterProvider = failingMeterProvider{MeterProvider: set.MeterProvider, calls: &calls, failAt: 2}

	_, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    pipeline.SignalLogs,
		ID:        exporterID,
		Telemetry: set,
	}, newFakeQueue[request.Request](nil, 7, 9))
	require.ErrorIs(t, err, errRegisterCallback)

	_, err = tt.GetMetric("otelcol_exporter_queue_size")
	require.Error(t, err, "queue size callback must be unregistered")
}
