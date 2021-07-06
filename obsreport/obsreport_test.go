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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/trace"

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

type receiveTestParams struct {
	items int
	err   error
}

func TestReceiveTraceDataOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(), t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	params := []receiveTestParams{
		{items: 13, err: errFake},
		{items: 42, err: nil},
	}
	for i, param := range params {
		rec := NewReceiver(ReceiverSettings{ReceiverID: receiver, Transport: transport})
		ctx := rec.StartTracesOp(parentCtx)
		assert.NotNil(t, ctx)
		rec.EndTracesOp(ctx, format, params[i].items, param.err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(params), len(spans))

	var acceptedSpans, refusedSpans int
	for i, span := range spans {
		assert.Equal(t, "receiver/"+receiver.String()+"/TraceDataReceived", span.Name)
		switch params[i].err {
		case nil:
			acceptedSpans += params[i].items
			assert.Equal(t, int64(params[i].items), span.Attributes[obsmetrics.AcceptedSpansKey])
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.RefusedSpansKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			refusedSpans += params[i].items
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.AcceptedSpansKey])
			assert.Equal(t, int64(params[i].items), span.Attributes[obsmetrics.RefusedSpansKey])
			assert.Equal(t, params[i].err.Error(), span.Status.Message)
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

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(),
		t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	params := []receiveTestParams{
		{items: 13, err: errFake},
		{items: 42, err: nil},
	}
	for i, param := range params {
		rec := NewReceiver(ReceiverSettings{ReceiverID: receiver, Transport: transport})
		ctx := rec.StartLogsOp(parentCtx)
		assert.NotNil(t, ctx)
		rec.EndLogsOp(ctx, format, params[i].items, param.err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(params), len(spans))

	var acceptedLogRecords, refusedLogRecords int
	for i, span := range spans {
		assert.Equal(t, "receiver/"+receiver.String()+"/LogsReceived", span.Name)
		switch params[i].err {
		case nil:
			acceptedLogRecords += params[i].items
			assert.Equal(t, int64(params[i].items), span.Attributes[obsmetrics.AcceptedLogRecordsKey])
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.RefusedLogRecordsKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			refusedLogRecords += params[i].items
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.AcceptedLogRecordsKey])
			assert.Equal(t, int64(params[i].items), span.Attributes[obsmetrics.RefusedLogRecordsKey])
			assert.Equal(t, params[i].err.Error(), span.Status.Message)
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

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(),
		t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	params := []receiveTestParams{
		{items: 23, err: errFake},
		{items: 29, err: nil},
	}
	for i, param := range params {
		rec := NewReceiver(ReceiverSettings{ReceiverID: receiver, Transport: transport})
		ctx := rec.StartMetricsOp(parentCtx)
		assert.NotNil(t, ctx)
		rec.EndMetricsOp(ctx, format, params[i].items, param.err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(params), len(spans))

	var acceptedMetricPoints, refusedMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "receiver/"+receiver.String()+"/MetricsReceived", span.Name)
		switch params[i].err {
		case nil:
			acceptedMetricPoints += params[i].items
			assert.Equal(t, int64(params[i].items), span.Attributes[obsmetrics.AcceptedMetricPointsKey])
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.RefusedMetricPointsKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			refusedMetricPoints += params[i].items
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.AcceptedMetricPointsKey])
			assert.Equal(t, int64(params[i].items), span.Attributes[obsmetrics.RefusedMetricPointsKey])
			assert.Equal(t, params[i].err.Error(), span.Status.Message)
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

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(),
		t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	receiverCtx := ScraperContext(parentCtx, receiver, scraper)
	errParams := []error{partialErrFake, errFake, nil}
	scrapedMetricPts := []int{23, 29, 15}
	for i, err := range errParams {
		scrp := NewScraper(ScraperSettings{ReceiverID: receiver, Scraper: scraper})
		ctx := scrp.StartMetricsOp(receiverCtx)
		assert.NotNil(t, ctx)

		scrp.EndMetricsOp(
			ctx,
			scrapedMetricPts[i],
			err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errParams), len(spans))

	var scrapedMetricPoints, erroredMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "scraper/"+receiver.String()+"/"+scraper.String()+"/MetricsScraped", span.Name)
		switch errParams[i] {
		case nil:
			scrapedMetricPoints += scrapedMetricPts[i]
			assert.Equal(t, int64(scrapedMetricPts[i]), span.Attributes[obsmetrics.ScrapedMetricPointsKey])
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.ErroredMetricPointsKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			erroredMetricPoints += scrapedMetricPts[i]
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.ScrapedMetricPointsKey])
			assert.Equal(t, int64(scrapedMetricPts[i]), span.Attributes[obsmetrics.ErroredMetricPointsKey])
			assert.Equal(t, errParams[i].Error(), span.Status.Message)
		case partialErrFake:
			scrapedMetricPoints += scrapedMetricPts[i]
			erroredMetricPoints++
			assert.Equal(t, int64(scrapedMetricPts[i]), span.Attributes[obsmetrics.ScrapedMetricPointsKey])
			assert.Equal(t, int64(1), span.Attributes[obsmetrics.ErroredMetricPointsKey])
			assert.Equal(t, errParams[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected err param: %v", errParams[i])
		}
	}

	obsreporttest.CheckScraperMetrics(t, receiver, scraper, int64(scrapedMetricPoints), int64(erroredMetricPoints))
}

func TestExportTraceDataOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(),
		t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	obsrep := NewExporter(ExporterSettings{Level: configtelemetry.LevelNormal, ExporterID: exporter})
	errs := []error{nil, errFake}
	numExportedSpans := []int{22, 14}
	for i, err := range errs {
		ctx := obsrep.StartTracesOp(parentCtx)
		assert.NotNil(t, ctx)
		obsrep.EndTracesOp(ctx, numExportedSpans[i], err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errs), len(spans))

	var sentSpans, failedToSendSpans int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporter.String()+"/traces", span.Name)
		switch errs[i] {
		case nil:
			sentSpans += numExportedSpans[i]
			assert.Equal(t, int64(numExportedSpans[i]), span.Attributes[obsmetrics.SentSpansKey])
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.FailedToSendSpansKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			failedToSendSpans += numExportedSpans[i]
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.SentSpansKey])
			assert.Equal(t, int64(numExportedSpans[i]), span.Attributes[obsmetrics.FailedToSendSpansKey])
			assert.Equal(t, errs[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected error: %v", errs[i])
		}
	}

	obsreporttest.CheckExporterTraces(t, exporter, int64(sentSpans), int64(failedToSendSpans))
}

func TestExportMetricsOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(),
		t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	obsrep := NewExporter(ExporterSettings{Level: configtelemetry.LevelNormal, ExporterID: exporter})

	errs := []error{nil, errFake}
	toSendMetricPoints := []int{17, 23}
	for i, err := range errs {
		ctx := obsrep.StartMetricsOp(parentCtx)
		assert.NotNil(t, ctx)

		obsrep.EndMetricsOp(ctx, toSendMetricPoints[i], err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errs), len(spans))

	var sentMetricPoints, failedToSendMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporter.String()+"/metrics", span.Name)
		switch errs[i] {
		case nil:
			sentMetricPoints += toSendMetricPoints[i]
			assert.Equal(t, int64(toSendMetricPoints[i]), span.Attributes[obsmetrics.SentMetricPointsKey])
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.FailedToSendMetricPointsKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			failedToSendMetricPoints += toSendMetricPoints[i]
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.SentMetricPointsKey])
			assert.Equal(t, int64(toSendMetricPoints[i]), span.Attributes[obsmetrics.FailedToSendMetricPointsKey])
			assert.Equal(t, errs[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected error: %v", errs[i])
		}
	}

	obsreporttest.CheckExporterMetrics(t, exporter, int64(sentMetricPoints), int64(failedToSendMetricPoints))
}

func TestExportLogsOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(),
		t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	obsrep := NewExporter(ExporterSettings{Level: configtelemetry.LevelNormal, ExporterID: exporter})
	errs := []error{nil, errFake}
	toSendLogRecords := []int{17, 23}
	for i, err := range errs {
		ctx := obsrep.StartLogsOp(parentCtx)
		assert.NotNil(t, ctx)

		obsrep.EndLogsOp(ctx, toSendLogRecords[i], err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errs), len(spans))

	var sentLogRecords, failedToSendLogRecords int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporter.String()+"/logs", span.Name)
		switch errs[i] {
		case nil:
			sentLogRecords += toSendLogRecords[i]
			assert.Equal(t, int64(toSendLogRecords[i]), span.Attributes[obsmetrics.SentLogRecordsKey])
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.FailedToSendLogRecordsKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			failedToSendLogRecords += toSendLogRecords[i]
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.SentLogRecordsKey])
			assert.Equal(t, int64(toSendLogRecords[i]), span.Attributes[obsmetrics.FailedToSendLogRecordsKey])
			assert.Equal(t, errs[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected error: %v", errs[i])
		}
	}

	obsreporttest.CheckExporterLogs(t, exporter, int64(sentLogRecords), int64(failedToSendLogRecords))
}

func TestReceiveWithLongLivedCtx(t *testing.T) {
	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})
	defer func() {
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.ProbabilitySampler(1e-4),
		})
	}()

	longLivedCtx, parentSpan := trace.StartSpan(context.Background(), t.Name())
	defer parentSpan.End()

	ops := []struct {
		numSpans int
		err      error
	}{
		{numSpans: 13},
		{numSpans: 42, err: errFake},
	}
	for _, op := range ops {
		// Use a new context on each operation to simulate distinct operations
		// under the same long lived context.
		rec := NewReceiver(ReceiverSettings{ReceiverID: receiver, Transport: transport, LongLivedCtx: true})
		ctx := rec.StartTracesOp(longLivedCtx)
		assert.NotNil(t, ctx)

		rec.EndTracesOp(
			ctx,
			format,
			op.numSpans,
			op.err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(ops), len(spans))

	for i, span := range spans {
		assert.Equal(t, trace.SpanID{}, span.ParentSpanID)
		require.Equal(t, 1, len(span.Links))
		link := span.Links[0]
		assert.Equal(t, trace.LinkTypeParent, link.Type)
		assert.Equal(t, parentSpan.SpanContext().TraceID, link.TraceID)
		assert.Equal(t, parentSpan.SpanContext().SpanID, link.SpanID)
		assert.Equal(t, "receiver/"+receiver.String()+"/TraceDataReceived", span.Name)
		assert.Equal(t, transport, span.Attributes[obsmetrics.TransportKey])
		switch ops[i].err {
		case nil:
			assert.Equal(t, int64(ops[i].numSpans), span.Attributes[obsmetrics.AcceptedSpansKey])
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.RefusedSpansKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			assert.Equal(t, int64(0), span.Attributes[obsmetrics.AcceptedSpansKey])
			assert.Equal(t, int64(ops[i].numSpans), span.Attributes[obsmetrics.RefusedSpansKey])
			assert.Equal(t, ops[i].err.Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected error: %v", ops[i].err)
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

type spanStore struct {
	sync.Mutex
	spans []*trace.SpanData
}

func (ss *spanStore) ExportSpan(sd *trace.SpanData) {
	ss.Lock()
	ss.spans = append(ss.spans, sd)
	ss.Unlock()
}

func (ss *spanStore) PullAllSpans() []*trace.SpanData {
	ss.Lock()
	capturedSpans := ss.spans
	ss.spans = nil
	ss.Unlock()
	return capturedSpans
}
