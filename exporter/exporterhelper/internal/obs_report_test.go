// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	exporterID = component.MustNewID("fakeExporter")

	errFake = errors.New("errFake")
)

func TestExportTraceDataOp(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(exporterID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	obsrep, err := NewObsReport(ObsReportSettings{
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		Signal:           pipeline.SignalTraces,
	})
	require.NoError(t, err)

	params := []testParams{
		{items: 22, err: nil},
		{items: 14, err: errFake},
	}
	for i := range params {
		ctx := obsrep.StartOp(parentCtx)
		assert.NotNil(t, ctx)
		obsrep.EndOp(ctx, params[i].items, params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Equal(t, len(params), len(spans))

	var sentSpans, failedToSendSpans int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/traces", span.Name())
		switch {
		case params[i].err == nil:
			sentSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			failedToSendSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	require.NoError(t, tt.CheckExporterTraces(int64(sentSpans), int64(failedToSendSpans)))
}

func TestExportMetricsOp(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(exporterID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	obsrep, err := NewObsReport(ObsReportSettings{
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		Signal:           pipeline.SignalMetrics,
	})
	require.NoError(t, err)

	params := []testParams{
		{items: 17, err: nil},
		{items: 23, err: errFake},
	}
	for i := range params {
		ctx := obsrep.StartOp(parentCtx)
		assert.NotNil(t, ctx)

		obsrep.EndOp(ctx, params[i].items, params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Equal(t, len(params), len(spans))

	var sentMetricPoints, failedToSendMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/metrics", span.Name())
		switch {
		case params[i].err == nil:
			sentMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			failedToSendMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	require.NoError(t, tt.CheckExporterMetrics(int64(sentMetricPoints), int64(failedToSendMetricPoints)))
}

func TestExportLogsOp(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(exporterID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	obsrep, err := NewObsReport(ObsReportSettings{
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		Signal:           pipeline.SignalLogs,
	})
	require.NoError(t, err)

	params := []testParams{
		{items: 17, err: nil},
		{items: 23, err: errFake},
	}
	for i := range params {
		ctx := obsrep.StartOp(parentCtx)
		assert.NotNil(t, ctx)

		obsrep.EndOp(ctx, params[i].items, params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Equal(t, len(params), len(spans))

	var sentLogRecords, failedToSendLogRecords int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporterID.String()+"/logs", span.Name())
		switch {
		case params[i].err == nil:
			sentLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			failedToSendLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsSent, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: ItemsFailed, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	metadatatest.AssertEqualExporterSentLogRecords(t, tt.Telemetry,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(sentLogRecords),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	if failedToSendLogRecords > 0 {
		metadatatest.AssertEqualExporterSendFailedLogRecords(t, tt.Telemetry,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", exporterID.String())),
					Value: int64(failedToSendLogRecords),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	}
}

func TestExportEnqueueFailure(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(exporterID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := NewObsReport(ObsReportSettings{
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		Signal:           pipeline.SignalLogs,
	})
	require.NoError(t, err)
	obsrep.RecordEnqueueFailure(context.Background(), int64(7))

	metadatatest.AssertEqualExporterEnqueueFailedLogRecords(t, tt.Telemetry,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(7),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	obsrep, err = NewObsReport(ObsReportSettings{
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		Signal:           pipeline.SignalTraces,
	})
	require.NoError(t, err)
	obsrep.RecordEnqueueFailure(context.Background(), int64(12))

	metadatatest.AssertEqualExporterEnqueueFailedSpans(t, tt.Telemetry,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(12),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	obsrep, err = NewObsReport(ObsReportSettings{
		ExporterSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		Signal:           pipeline.SignalMetrics,
	})
	require.NoError(t, err)
	obsrep.RecordEnqueueFailure(context.Background(), int64(21))

	metadatatest.AssertEqualExporterEnqueueFailedMetricPoints(t, tt.Telemetry,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String("exporter", exporterID.String())),
				Value: int64(21),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

type testParams struct {
	items int
	err   error
}
