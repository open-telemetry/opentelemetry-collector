// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/pipeline"
)

type fakeQueue[T any] struct {
	exporterqueue.Queue[T]
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

func newFakeQueue[T request.Request](offerErr error, size, capacity int64) exporterqueue.Queue[T] {
	return &fakeQueue[T]{offerErr: offerErr, size: size, capacity: capacity}
}

func TestObsQueueLogsSizeCapacity(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](exporterqueue.Settings{
		Signal:           pipeline.SignalLogs,
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	}, newFakeQueue[request.Request](nil, 7, 9))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 2}))
	metadatatest.AssertEqualExporterQueueSize(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String()),
					attribute.String(DataTypeKey, pipeline.SignalLogs.String())),
				Value: int64(7),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualExporterQueueCapacity(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String()),
					attribute.String(DataTypeKey, pipeline.SignalLogs.String())),
				Value: int64(9),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestObsQueueLogsFailure(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](exporterqueue.Settings{
		Signal:           pipeline.SignalLogs,
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	}, newFakeQueue[request.Request](errors.New("my error"), 7, 9))
	require.NoError(t, err)
	require.Error(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 2}))
	metadatatest.AssertEqualExporterEnqueueFailedLogRecords(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(2),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestObsQueueTracesSizeCapacity(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](exporterqueue.Settings{
		Signal:           pipeline.SignalTraces,
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	}, newFakeQueue[request.Request](nil, 17, 19))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 12}))
	metadatatest.AssertEqualExporterQueueSize(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String()),
					attribute.String(DataTypeKey, pipeline.SignalTraces.String())),
				Value: int64(17),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualExporterQueueCapacity(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String()),
					attribute.String(DataTypeKey, pipeline.SignalTraces.String())),
				Value: int64(19),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestObsQueueTracesFailure(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](exporterqueue.Settings{
		Signal:           pipeline.SignalTraces,
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	}, newFakeQueue[request.Request](errors.New("my error"), 0, 0))
	require.NoError(t, err)
	require.Error(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 12}))
	metadatatest.AssertEqualExporterEnqueueFailedSpans(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(12),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestObsQueueMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](exporterqueue.Settings{
		Signal:           pipeline.SignalMetrics,
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	}, newFakeQueue[request.Request](nil, 27, 29))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 22}))
	metadatatest.AssertEqualExporterQueueSize(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String()),
					attribute.String(DataTypeKey, pipeline.SignalMetrics.String())),
				Value: int64(27),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualExporterQueueCapacity(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String()),
					attribute.String(DataTypeKey, pipeline.SignalMetrics.String())),
				Value: int64(29),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestObsQueueMetricsFailure(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	te, err := newObsQueue[request.Request](exporterqueue.Settings{
		Signal:           pipeline.SignalMetrics,
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	}, newFakeQueue[request.Request](errors.New("my error"), 0, 0))
	require.NoError(t, err)
	require.Error(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 22}))
	metadatatest.AssertEqualExporterEnqueueFailedMetricPoints(t, tt,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(22),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}
