// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package obsreport

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

const (
	transport = "fakeTransport"
	format    = "fakeFormat"
)

var (
	receiver  = config.NewID("fakeReicever")
	scraper   = config.NewID("fakeScraper")
	processor = config.NewID("fakeProcessor")
	exporter  = config.NewID("fakeExporter")

	errFake        = errors.New("errFake")
	partialErrFake = scrapererror.NewPartialScrapeError(errFake, 1)
)

type testParams struct {
	items int
	err   error
}

func TestReceiveTraceDataOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	sr := new(tracetest.SpanRecorder)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	parentCtx, parentSpan := tp.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	params := []testParams{
		{items: 13, err: errFake},
		{items: 42, err: nil},
	}
	for i, param := range params {
		rec := NewReceiver(ReceiverSettings{ReceiverID: receiver, Transport: transport})
		ctx := rec.StartTracesOp(parentCtx)
		assert.NotNil(t, ctx)
		rec.EndTracesOp(ctx, format, params[i].items, param.err)
	}

	spans := sr.Ended()
	require.Equal(t, len(params), len(spans))

	var acceptedSpans, refusedSpans int
	for i, span := range spans {
		assert.Equal(t, "receiver/"+receiver.String()+"/TraceDataReceived", span.Name())
		switch params[i].err {
		case nil:
			acceptedSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedSpansKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errFake:
			refusedSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedSpansKey, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected param: %v", params[i])
		}
	}
	obsreporttest.CheckReceiverTraces(t, receiver, transport, int64(acceptedSpans), int64(refusedSpans))
}

func TestReceiveLogsOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	sr := new(tracetest.SpanRecorder)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	parentCtx, parentSpan := tp.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	params := []testParams{
		{items: 13, err: errFake},
		{items: 42, err: nil},
	}
	for i, param := range params {
		rec := NewReceiver(ReceiverSettings{ReceiverID: receiver, Transport: transport})
		ctx := rec.StartLogsOp(parentCtx)
		assert.NotNil(t, ctx)
		rec.EndLogsOp(ctx, format, params[i].items, param.err)
	}

	spans := sr.Ended()
	require.Equal(t, len(params), len(spans))

	var acceptedLogRecords, refusedLogRecords int
	for i, span := range spans {
		assert.Equal(t, "receiver/"+receiver.String()+"/LogsReceived", span.Name())
		switch params[i].err {
		case nil:
			acceptedLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedLogRecordsKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errFake:
			refusedLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedLogRecordsKey, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected param: %v", params[i])
		}
	}
	obsreporttest.CheckReceiverLogs(t, receiver, transport, int64(acceptedLogRecords), int64(refusedLogRecords))
}

func TestReceiveMetricsOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	sr := new(tracetest.SpanRecorder)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	parentCtx, parentSpan := tp.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	params := []testParams{
		{items: 23, err: errFake},
		{items: 29, err: nil},
	}
	for i, param := range params {
		rec := NewReceiver(ReceiverSettings{ReceiverID: receiver, Transport: transport})
		ctx := rec.StartMetricsOp(parentCtx)
		assert.NotNil(t, ctx)
		rec.EndMetricsOp(ctx, format, params[i].items, param.err)
	}

	spans := sr.Ended()
	require.Equal(t, len(params), len(spans))

	var acceptedMetricPoints, refusedMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "receiver/"+receiver.String()+"/MetricsReceived", span.Name())
		switch params[i].err {
		case nil:
			acceptedMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedMetricPointsKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errFake:
			refusedMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedMetricPointsKey, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected param: %v", params[i])
		}
	}

	obsreporttest.CheckReceiverMetrics(t, receiver, transport, int64(acceptedMetricPoints), int64(refusedMetricPoints))
}

func TestScrapeMetricsDataOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	sr := new(tracetest.SpanRecorder)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	parentCtx, parentSpan := tp.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	params := []testParams{
		{items: 23, err: partialErrFake},
		{items: 29, err: errFake},
		{items: 15, err: nil},
	}
	for i := range params {
		scrp := NewScraper(ScraperSettings{ReceiverID: receiver, Scraper: scraper})
		ctx := scrp.StartMetricsOp(parentCtx)
		assert.NotNil(t, ctx)
		scrp.EndMetricsOp(ctx, params[i].items, params[i].err)
	}

	spans := sr.Ended()
	require.Equal(t, len(params), len(spans))

	var scrapedMetricPoints, erroredMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "scraper/"+receiver.String()+"/"+scraper.String()+"/MetricsScraped", span.Name())
		switch params[i].err {
		case nil:
			scrapedMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ScrapedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ErroredMetricPointsKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errFake:
			erroredMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ScrapedMetricPointsKey, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ErroredMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)

		case partialErrFake:
			scrapedMetricPoints += params[i].items
			erroredMetricPoints++
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ScrapedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ErroredMetricPointsKey, Value: attribute.Int64Value(1)})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected err param: %v", params[i].err)
		}
	}

	obsreporttest.CheckScraperMetrics(t, receiver, scraper, int64(scrapedMetricPoints), int64(erroredMetricPoints))
}

func TestExportTraceDataOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	set := componenttest.NewNopExporterCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	parentCtx, parentSpan := set.TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	obsrep := NewExporter(ExporterSettings{
		Level:                  configtelemetry.LevelNormal,
		ExporterID:             exporter,
		ExporterCreateSettings: set,
	})

	params := []testParams{
		{items: 22, err: nil},
		{items: 14, err: errFake},
	}
	for i := range params {
		ctx := obsrep.StartTracesOp(parentCtx)
		assert.NotNil(t, ctx)
		obsrep.EndTracesOp(ctx, params[i].items, params[i].err)
	}

	spans := sr.Ended()
	require.Equal(t, len(params), len(spans))

	var sentSpans, failedToSendSpans int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporter.String()+"/traces", span.Name())
		switch params[i].err {
		case nil:
			sentSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendSpansKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errFake:
			failedToSendSpans += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentSpansKey, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	obsreporttest.CheckExporterTraces(t, exporter, int64(sentSpans), int64(failedToSendSpans))
}

func TestExportMetricsOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	set := componenttest.NewNopExporterCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	parentCtx, parentSpan := set.TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	obsrep := NewExporter(ExporterSettings{
		Level:                  configtelemetry.LevelNormal,
		ExporterID:             exporter,
		ExporterCreateSettings: set,
	})

	params := []testParams{
		{items: 17, err: nil},
		{items: 23, err: errFake},
	}
	for i := range params {
		ctx := obsrep.StartMetricsOp(parentCtx)
		assert.NotNil(t, ctx)

		obsrep.EndMetricsOp(ctx, params[i].items, params[i].err)
	}

	spans := sr.Ended()
	require.Equal(t, len(params), len(spans))

	var sentMetricPoints, failedToSendMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporter.String()+"/metrics", span.Name())
		switch params[i].err {
		case nil:
			sentMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendMetricPointsKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errFake:
			failedToSendMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentMetricPointsKey, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	obsreporttest.CheckExporterMetrics(t, exporter, int64(sentMetricPoints), int64(failedToSendMetricPoints))
}

func TestExportLogsOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	set := componenttest.NewNopExporterCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	parentCtx, parentSpan := set.TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	obsrep := NewExporter(ExporterSettings{
		Level:                  configtelemetry.LevelNormal,
		ExporterID:             exporter,
		ExporterCreateSettings: set,
	})

	params := []testParams{
		{items: 17, err: nil},
		{items: 23, err: errFake},
	}
	for i := range params {
		ctx := obsrep.StartLogsOp(parentCtx)
		assert.NotNil(t, ctx)

		obsrep.EndLogsOp(ctx, params[i].items, params[i].err)
	}

	spans := sr.Ended()
	require.Equal(t, len(params), len(spans))

	var sentLogRecords, failedToSendLogRecords int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporter.String()+"/logs", span.Name())
		switch params[i].err {
		case nil:
			sentLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendLogRecordsKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errFake:
			failedToSendLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.SentLogRecordsKey, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendLogRecordsKey, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}

	obsreporttest.CheckExporterLogs(t, exporter, int64(sentLogRecords), int64(failedToSendLogRecords))
}

func TestReceiveWithLongLivedCtx(t *testing.T) {
	sr := new(tracetest.SpanRecorder)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	longLivedCtx, parentSpan := tp.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	params := []testParams{
		{items: 17, err: nil},
		{items: 23, err: errFake},
	}
	for i := range params {
		// Use a new context on each operation to simulate distinct operations
		// under the same long lived context.
		rec := NewReceiver(ReceiverSettings{ReceiverID: receiver, Transport: transport, LongLivedCtx: true})
		ctx := rec.StartTracesOp(longLivedCtx)
		assert.NotNil(t, ctx)
		rec.EndTracesOp(ctx, format, params[i].items, params[i].err)
	}

	spans := sr.Ended()
	require.Equal(t, len(params), len(spans))

	for i, span := range spans {
		assert.False(t, span.Parent().IsValid())
		require.Equal(t, 1, len(span.Links()))
		link := span.Links()[0]
		assert.Equal(t, parentSpan.SpanContext().TraceID(), link.SpanContext.TraceID())
		assert.Equal(t, parentSpan.SpanContext().SpanID(), link.SpanContext.SpanID())
		assert.Equal(t, "receiver/"+receiver.String()+"/TraceDataReceived", span.Name())
		require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.TransportKey, Value: attribute.StringValue(transport)})
		switch params[i].err {
		case nil:
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedSpansKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errFake:
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.AcceptedSpansKey, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.RefusedSpansKey, Value: attribute.Int64Value(int64(params[i].items))})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected error: %v", params[i].err)
		}
	}
}

func TestProcessorTraceData(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	const acceptedSpans = 27
	const refusedSpans = 19
	const droppedSpans = 13

	obsrep := NewProcessor(ProcessorSettings{Level: configtelemetry.LevelNormal, ProcessorID: processor})
	obsrep.TracesAccepted(context.Background(), acceptedSpans)
	obsrep.TracesRefused(context.Background(), refusedSpans)
	obsrep.TracesDropped(context.Background(), droppedSpans)

	obsreporttest.CheckProcessorTraces(t, processor, acceptedSpans, refusedSpans, droppedSpans)
}

func TestProcessorMetricsData(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	const acceptedPoints = 29
	const refusedPoints = 11
	const droppedPoints = 17

	obsrep := NewProcessor(ProcessorSettings{Level: configtelemetry.LevelNormal, ProcessorID: processor})
	obsrep.MetricsAccepted(context.Background(), acceptedPoints)
	obsrep.MetricsRefused(context.Background(), refusedPoints)
	obsrep.MetricsDropped(context.Background(), droppedPoints)

	obsreporttest.CheckProcessorMetrics(t, processor, acceptedPoints, refusedPoints, droppedPoints)
}

func TestBuildProcessorCustomMetricName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "firstMeasure",
			want: "processor/test_type/firstMeasure",
		},
		{
			name: "secondMeasure",
			want: "processor/test_type/secondMeasure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildProcessorCustomMetricName("test_type", tt.name)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProcessorLogRecords(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	const acceptedRecords = 29
	const refusedRecords = 11
	const droppedRecords = 17

	obsrep := NewProcessor(ProcessorSettings{Level: configtelemetry.LevelNormal, ProcessorID: processor})
	obsrep.LogsAccepted(context.Background(), acceptedRecords)
	obsrep.LogsRefused(context.Background(), refusedRecords)
	obsrep.LogsDropped(context.Background(), droppedRecords)

	obsreporttest.CheckProcessorLogs(t, processor, acceptedRecords, refusedRecords, droppedRecords)
}
