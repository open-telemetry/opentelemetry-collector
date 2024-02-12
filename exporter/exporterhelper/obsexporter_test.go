// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

var (
	exporterID = component.MustNewID("fakeExporter")

	errFake = errors.New("errFake")
)

func TestExportTraceDataOp(t *testing.T) {
	testTelemetry(t, exporterID, func(t *testing.T, tt componenttest.TestTelemetry) {
		parentCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
		defer parentSpan.End()

		obsrep, err := newExporter(ObsReportSettings{
			ExporterID:             exporterID,
			ExporterCreateSettings: exporter.CreateSettings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, err)

		params := []testParams{
			{items: 22, err: nil},
			{items: 14, err: errFake},
		}
		for i := range params {
			ctx := obsrep.StartTracesOp(parentCtx)
			assert.NotNil(t, ctx)
			obsrep.EndTracesOp(ctx, params[i].items, params[i].err)
		}

		spans := tt.SpanRecorder.Ended()
		require.Equal(t, len(params), len(spans))

		var sentSpans, failedToSendSpans int
		for i, span := range spans {
			assert.Equal(t, "exporter/"+exporterID.String()+"/traces", span.Name())
			switch {
			case params[i].err == nil:
				sentSpans += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendSpansKey, Value: attribute.Int64Value(0)})
				assert.Equal(t, codes.Unset, span.Status().Code)
			case errors.Is(params[i].err, errFake):
				failedToSendSpans += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentSpansKey, Value: attribute.Int64Value(0)})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, params[i].err.Error(), span.Status().Description)
			default:
				t.Fatalf("unexpected error: %v", params[i].err)
			}
		}

		require.NoError(t, tt.CheckExporterTraces(int64(sentSpans), int64(failedToSendSpans)))
	})
}

func TestExportMetricsOp(t *testing.T) {
	testTelemetry(t, exporterID, func(t *testing.T, tt componenttest.TestTelemetry) {
		parentCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
		defer parentSpan.End()

		obsrep, err := newExporter(ObsReportSettings{
			ExporterID:             exporterID,
			ExporterCreateSettings: exporter.CreateSettings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, err)

		params := []testParams{
			{items: 17, err: nil},
			{items: 23, err: errFake},
		}
		for i := range params {
			ctx := obsrep.StartMetricsOp(parentCtx)
			assert.NotNil(t, ctx)

			obsrep.EndMetricsOp(ctx, params[i].items, params[i].err)
		}

		spans := tt.SpanRecorder.Ended()
		require.Equal(t, len(params), len(spans))

		var sentMetricPoints, failedToSendMetricPoints int
		for i, span := range spans {
			assert.Equal(t, "exporter/"+exporterID.String()+"/metrics", span.Name())
			switch {
			case params[i].err == nil:
				sentMetricPoints += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendMetricPointsKey, Value: attribute.Int64Value(0)})
				assert.Equal(t, codes.Unset, span.Status().Code)
			case errors.Is(params[i].err, errFake):
				failedToSendMetricPoints += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentMetricPointsKey, Value: attribute.Int64Value(0)})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, params[i].err.Error(), span.Status().Description)
			default:
				t.Fatalf("unexpected error: %v", params[i].err)
			}
		}

		require.NoError(t, tt.CheckExporterMetrics(int64(sentMetricPoints), int64(failedToSendMetricPoints)))
	})
}

func TestExportLogsOp(t *testing.T) {
	testTelemetry(t, exporterID, func(t *testing.T, tt componenttest.TestTelemetry) {
		parentCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
		defer parentSpan.End()

		obsrep, err := newExporter(ObsReportSettings{
			ExporterID:             exporterID,
			ExporterCreateSettings: exporter.CreateSettings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, err)

		params := []testParams{
			{items: 17, err: nil},
			{items: 23, err: errFake},
		}
		for i := range params {
			ctx := obsrep.StartLogsOp(parentCtx)
			assert.NotNil(t, ctx)

			obsrep.EndLogsOp(ctx, params[i].items, params[i].err)
		}

		spans := tt.SpanRecorder.Ended()
		require.Equal(t, len(params), len(spans))

		var sentLogRecords, failedToSendLogRecords int
		for i, span := range spans {
			assert.Equal(t, "exporter/"+exporterID.String()+"/logs", span.Name())
			switch {
			case params[i].err == nil:
				sentLogRecords += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendLogRecordsKey, Value: attribute.Int64Value(0)})
				assert.Equal(t, codes.Unset, span.Status().Code)
			case errors.Is(params[i].err, errFake):
				failedToSendLogRecords += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentLogRecordsKey, Value: attribute.Int64Value(0)})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, params[i].err.Error(), span.Status().Description)
			default:
				t.Fatalf("unexpected error: %v", params[i].err)
			}
		}

		require.NoError(t, tt.CheckExporterLogs(int64(sentLogRecords), int64(failedToSendLogRecords)))
	})
}

func TestCheckExporterTracesViews(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(exporterID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := NewObsReport(ObsReportSettings{
		ExporterID:             exporterID,
		ExporterCreateSettings: exporter.CreateSettings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := obsrep.StartTracesOp(context.Background())
	require.NotNil(t, ctx)
	obsrep.EndTracesOp(ctx, 7, nil)

	assert.NoError(t, tt.CheckExporterTraces(7, 0))
	assert.Error(t, tt.CheckExporterTraces(7, 7))
	assert.Error(t, tt.CheckExporterTraces(0, 0))
	assert.Error(t, tt.CheckExporterTraces(0, 7))
}

func TestCheckExporterMetricsViews(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(exporterID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := NewObsReport(ObsReportSettings{
		ExporterID:             exporterID,
		ExporterCreateSettings: exporter.CreateSettings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := obsrep.StartMetricsOp(context.Background())
	require.NotNil(t, ctx)
	obsrep.EndMetricsOp(ctx, 7, nil)

	assert.NoError(t, tt.CheckExporterMetrics(7, 0))
	assert.Error(t, tt.CheckExporterMetrics(7, 7))
	assert.Error(t, tt.CheckExporterMetrics(0, 0))
	assert.Error(t, tt.CheckExporterMetrics(0, 7))
}

func TestCheckExporterLogsViews(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(exporterID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := NewObsReport(ObsReportSettings{
		ExporterID:             exporterID,
		ExporterCreateSettings: exporter.CreateSettings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := obsrep.StartLogsOp(context.Background())
	require.NotNil(t, ctx)
	obsrep.EndLogsOp(ctx, 7, nil)

	assert.NoError(t, tt.CheckExporterLogs(7, 0))
	assert.Error(t, tt.CheckExporterLogs(7, 7))
	assert.Error(t, tt.CheckExporterLogs(0, 0))
	assert.Error(t, tt.CheckExporterLogs(0, 7))
}

type testParams struct {
	items int
	err   error
}

func testTelemetry(t *testing.T, id component.ID, testFunc func(t *testing.T, tt componenttest.TestTelemetry)) {
	t.Run("WithOTel", func(t *testing.T) {
		tt, err := componenttest.SetupTelemetry(id)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

		testFunc(t, tt)
	})
}
