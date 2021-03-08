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

// obsreport_test instead of just obsreport to avoid dependency cycle between
// obsreport_test and obsreporttest
package obsreport_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

const (
	exporter  = "fakeExporter"
	processor = "fakeProcessor"
	receiver  = "fakeReicever"
	scraper   = "fakeScraper"
	transport = "fakeTransport"
	format    = "fakeFormat"
)

var (
	errFake        = errors.New("errFake")
	partialErrFake = scrapererror.NewPartialScrapeError(errFake, 1)
)

type receiveTestParams struct {
	transport string
	err       error
}

func TestConfigure(t *testing.T) {
	tests := []struct {
		name      string
		level     configtelemetry.Level
		wantViews []*view.View
	}{
		{
			name:  "none",
			level: configtelemetry.LevelNone,
		},
		{
			name:      "basic",
			level:     configtelemetry.LevelBasic,
			wantViews: obsreport.AllViews(),
		},
		{
			name:      "normal",
			level:     configtelemetry.LevelNormal,
			wantViews: obsreport.AllViews(),
		},
		{
			name:      "detailed",
			level:     configtelemetry.LevelDetailed,
			wantViews: obsreport.AllViews(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotViews := obsreport.Configure(tt.level)
			assert.Equal(t, tt.wantViews, gotViews)
		})
	}
}

func TestReceiveTraceDataOp(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(),
		t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	receiverCtx := obsreport.ReceiverContext(parentCtx, receiver, transport)
	params := []receiveTestParams{
		{transport, errFake},
		{"", nil},
	}
	rcvdSpans := []int{13, 42}
	for i, param := range params {
		ctx := obsreport.StartTraceDataReceiveOp(receiverCtx, receiver, param.transport)
		assert.NotNil(t, ctx)

		obsreport.EndTraceDataReceiveOp(
			ctx,
			format,
			rcvdSpans[i],
			param.err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(params), len(spans))

	var acceptedSpans, refusedSpans int
	for i, span := range spans {
		assert.Equal(t, "receiver/"+receiver+"/TraceDataReceived", span.Name)
		switch params[i].err {
		case nil:
			acceptedSpans += rcvdSpans[i]
			assert.Equal(t, int64(rcvdSpans[i]), span.Attributes[obsreport.AcceptedSpansKey])
			assert.Equal(t, int64(0), span.Attributes[obsreport.RefusedSpansKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			refusedSpans += rcvdSpans[i]
			assert.Equal(t, int64(0), span.Attributes[obsreport.AcceptedSpansKey])
			assert.Equal(t, int64(rcvdSpans[i]), span.Attributes[obsreport.RefusedSpansKey])
			assert.Equal(t, params[i].err.Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected param: %v", params[i])
		}
		switch params[i].transport {
		case "":
			assert.NotContains(t, span.Attributes, obsreport.TransportKey)
		default:
			assert.Equal(t, params[i].transport, span.Attributes[obsreport.TransportKey])
		}
	}
	obsreporttest.CheckReceiverTracesViews(t, receiver, transport, int64(acceptedSpans), int64(refusedSpans))
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

	receiverCtx := obsreport.ReceiverContext(parentCtx, receiver, transport)
	params := []receiveTestParams{
		{transport, errFake},
		{"", nil},
	}
	rcvdLogRecords := []int{13, 42}
	for i, param := range params {
		ctx := obsreport.StartLogsReceiveOp(receiverCtx, receiver, param.transport)
		assert.NotNil(t, ctx)

		obsreport.EndLogsReceiveOp(
			ctx,
			format,
			rcvdLogRecords[i],
			param.err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(params), len(spans))

	var acceptedLogRecords, refusedLogRecords int
	for i, span := range spans {
		assert.Equal(t, "receiver/"+receiver+"/LogsReceived", span.Name)
		switch params[i].err {
		case nil:
			acceptedLogRecords += rcvdLogRecords[i]
			assert.Equal(t, int64(rcvdLogRecords[i]), span.Attributes[obsreport.AcceptedLogRecordsKey])
			assert.Equal(t, int64(0), span.Attributes[obsreport.RefusedLogRecordsKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			refusedLogRecords += rcvdLogRecords[i]
			assert.Equal(t, int64(0), span.Attributes[obsreport.AcceptedLogRecordsKey])
			assert.Equal(t, int64(rcvdLogRecords[i]), span.Attributes[obsreport.RefusedLogRecordsKey])
			assert.Equal(t, params[i].err.Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected param: %v", params[i])
		}
		switch params[i].transport {
		case "":
			assert.NotContains(t, span.Attributes, obsreport.TransportKey)
		default:
			assert.Equal(t, params[i].transport, span.Attributes[obsreport.TransportKey])
		}
	}
	obsreporttest.CheckReceiverLogsViews(t, receiver, transport, int64(acceptedLogRecords), int64(refusedLogRecords))
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

	receiverCtx := obsreport.ReceiverContext(parentCtx, receiver, transport)
	params := []receiveTestParams{
		{transport, errFake},
		{"", nil},
	}
	rcvdMetricPts := []int{23, 29}
	for i, param := range params {
		ctx := obsreport.StartMetricsReceiveOp(receiverCtx, receiver, param.transport)
		assert.NotNil(t, ctx)

		obsreport.EndMetricsReceiveOp(
			ctx,
			format,
			rcvdMetricPts[i],
			param.err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(params), len(spans))

	var acceptedMetricPoints, refusedMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "receiver/"+receiver+"/MetricsReceived", span.Name)
		switch params[i].err {
		case nil:
			acceptedMetricPoints += rcvdMetricPts[i]
			assert.Equal(t, int64(rcvdMetricPts[i]), span.Attributes[obsreport.AcceptedMetricPointsKey])
			assert.Equal(t, int64(0), span.Attributes[obsreport.RefusedMetricPointsKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			refusedMetricPoints += rcvdMetricPts[i]
			assert.Equal(t, int64(0), span.Attributes[obsreport.AcceptedMetricPointsKey])
			assert.Equal(t, int64(rcvdMetricPts[i]), span.Attributes[obsreport.RefusedMetricPointsKey])
			assert.Equal(t, params[i].err.Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected param: %v", params[i])
		}
		switch params[i].transport {
		case "":
			assert.NotContains(t, span.Attributes, obsreport.TransportKey)
		default:
			assert.Equal(t, params[i].transport, span.Attributes[obsreport.TransportKey])
		}
	}

	obsreporttest.CheckReceiverMetricsViews(t, receiver, transport, int64(acceptedMetricPoints), int64(refusedMetricPoints))
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

	receiverCtx := obsreport.ScraperContext(parentCtx, receiver, scraper)
	errParams := []error{partialErrFake, errFake, nil}
	scrapedMetricPts := []int{23, 29, 15}
	for i, err := range errParams {
		ctx := obsreport.StartMetricsScrapeOp(receiverCtx, receiver, scraper)
		assert.NotNil(t, ctx)

		obsreport.EndMetricsScrapeOp(
			ctx,
			scrapedMetricPts[i],
			err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errParams), len(spans))

	var scrapedMetricPoints, erroredMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "scraper/"+receiver+"/"+scraper+"/MetricsScraped", span.Name)
		switch errParams[i] {
		case nil:
			scrapedMetricPoints += scrapedMetricPts[i]
			assert.Equal(t, int64(scrapedMetricPts[i]), span.Attributes[obsreport.ScrapedMetricPointsKey])
			assert.Equal(t, int64(0), span.Attributes[obsreport.ErroredMetricPointsKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			erroredMetricPoints += scrapedMetricPts[i]
			assert.Equal(t, int64(0), span.Attributes[obsreport.ScrapedMetricPointsKey])
			assert.Equal(t, int64(scrapedMetricPts[i]), span.Attributes[obsreport.ErroredMetricPointsKey])
			assert.Equal(t, errParams[i].Error(), span.Status.Message)
		case partialErrFake:
			scrapedMetricPoints += scrapedMetricPts[i]
			erroredMetricPoints++
			assert.Equal(t, int64(scrapedMetricPts[i]), span.Attributes[obsreport.ScrapedMetricPointsKey])
			assert.Equal(t, int64(1), span.Attributes[obsreport.ErroredMetricPointsKey])
			assert.Equal(t, errParams[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected err param: %v", errParams[i])
		}
	}

	obsreporttest.CheckScraperMetricsViews(t, receiver, scraper, int64(scrapedMetricPoints), int64(erroredMetricPoints))
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

	exporterCtx := obsreport.ExporterContext(parentCtx, exporter)
	obsrep := obsreport.NewExporterObsReport(configtelemetry.LevelNormal, exporter)
	errs := []error{nil, errFake}
	numExportedSpans := []int{22, 14}
	for i, err := range errs {
		ctx := obsrep.StartTracesExportOp(exporterCtx)
		assert.NotNil(t, ctx)
		obsrep.EndTracesExportOp(ctx, numExportedSpans[i], err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errs), len(spans))

	var sentSpans, failedToSendSpans int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporter+"/traces", span.Name)
		switch errs[i] {
		case nil:
			sentSpans += numExportedSpans[i]
			assert.Equal(t, int64(numExportedSpans[i]), span.Attributes[obsreport.SentSpansKey])
			assert.Equal(t, int64(0), span.Attributes[obsreport.FailedToSendSpansKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			failedToSendSpans += numExportedSpans[i]
			assert.Equal(t, int64(0), span.Attributes[obsreport.SentSpansKey])
			assert.Equal(t, int64(numExportedSpans[i]), span.Attributes[obsreport.FailedToSendSpansKey])
			assert.Equal(t, errs[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected error: %v", errs[i])
		}
	}

	obsreporttest.CheckExporterTracesViews(t, exporter, int64(sentSpans), int64(failedToSendSpans))
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

	exporterCtx := obsreport.ExporterContext(parentCtx, exporter)
	obsrep := obsreport.NewExporterObsReport(configtelemetry.LevelNormal, exporter)

	errs := []error{nil, errFake}
	toSendMetricPoints := []int{17, 23}
	for i, err := range errs {
		ctx := obsrep.StartMetricsExportOp(exporterCtx)
		assert.NotNil(t, ctx)

		obsrep.EndMetricsExportOp(ctx, toSendMetricPoints[i], err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errs), len(spans))

	var sentMetricPoints, failedToSendMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporter+"/metrics", span.Name)
		switch errs[i] {
		case nil:
			sentMetricPoints += toSendMetricPoints[i]
			assert.Equal(t, int64(toSendMetricPoints[i]), span.Attributes[obsreport.SentMetricPointsKey])
			assert.Equal(t, int64(0), span.Attributes[obsreport.FailedToSendMetricPointsKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			failedToSendMetricPoints += toSendMetricPoints[i]
			assert.Equal(t, int64(0), span.Attributes[obsreport.SentMetricPointsKey])
			assert.Equal(t, int64(toSendMetricPoints[i]), span.Attributes[obsreport.FailedToSendMetricPointsKey])
			assert.Equal(t, errs[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected error: %v", errs[i])
		}
	}

	obsreporttest.CheckExporterMetricsViews(t, exporter, int64(sentMetricPoints), int64(failedToSendMetricPoints))
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

	exporterCtx := obsreport.ExporterContext(parentCtx, exporter)
	obsrep := obsreport.NewExporterObsReport(configtelemetry.LevelNormal, exporter)

	errs := []error{nil, errFake}
	toSendLogRecords := []int{17, 23}
	for i, err := range errs {
		ctx := obsrep.StartLogsExportOp(exporterCtx)
		assert.NotNil(t, ctx)

		obsrep.EndLogsExportOp(ctx, toSendLogRecords[i], err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errs), len(spans))

	var sentLogRecords, failedToSendLogRecords int
	for i, span := range spans {
		assert.Equal(t, "exporter/"+exporter+"/logs", span.Name)
		switch errs[i] {
		case nil:
			sentLogRecords += toSendLogRecords[i]
			assert.Equal(t, int64(toSendLogRecords[i]), span.Attributes[obsreport.SentLogRecordsKey])
			assert.Equal(t, int64(0), span.Attributes[obsreport.FailedToSendLogRecordsKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			failedToSendLogRecords += toSendLogRecords[i]
			assert.Equal(t, int64(0), span.Attributes[obsreport.SentLogRecordsKey])
			assert.Equal(t, int64(toSendLogRecords[i]), span.Attributes[obsreport.FailedToSendLogRecordsKey])
			assert.Equal(t, errs[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected error: %v", errs[i])
		}
	}

	obsreporttest.CheckExporterLogsViews(t, exporter, int64(sentLogRecords), int64(failedToSendLogRecords))
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

	parentCtx, parentSpan := trace.StartSpan(context.Background(), t.Name())
	defer parentSpan.End()

	longLivedCtx := obsreport.ReceiverContext(parentCtx, receiver, transport)
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
		ctx := obsreport.StartTraceDataReceiveOp(
			longLivedCtx,
			receiver,
			transport,
			obsreport.WithLongLivedCtx())
		assert.NotNil(t, ctx)

		obsreport.EndTraceDataReceiveOp(
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
		assert.Equal(t, "receiver/"+receiver+"/TraceDataReceived", span.Name)
		assert.Equal(t, transport, span.Attributes[obsreport.TransportKey])
		switch ops[i].err {
		case nil:
			assert.Equal(t, int64(ops[i].numSpans), span.Attributes[obsreport.AcceptedSpansKey])
			assert.Equal(t, int64(0), span.Attributes[obsreport.RefusedSpansKey])
			assert.Equal(t, trace.Status{Code: trace.StatusCodeOK}, span.Status)
		case errFake:
			assert.Equal(t, int64(0), span.Attributes[obsreport.AcceptedSpansKey])
			assert.Equal(t, int64(ops[i].numSpans), span.Attributes[obsreport.RefusedSpansKey])
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

	obsrep := obsreport.NewProcessorObsReport(configtelemetry.LevelNormal, processor)
	obsrep.TracesAccepted(context.Background(), acceptedSpans)
	obsrep.TracesRefused(context.Background(), refusedSpans)
	obsrep.TracesDropped(context.Background(), droppedSpans)

	obsreporttest.CheckProcessorTracesViews(t, processor, acceptedSpans, refusedSpans, droppedSpans)
}

func TestProcessorMetricsData(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	const acceptedPoints = 29
	const refusedPoints = 11
	const droppedPoints = 17

	obsrep := obsreport.NewProcessorObsReport(configtelemetry.LevelNormal, processor)
	obsrep.MetricsAccepted(context.Background(), acceptedPoints)
	obsrep.MetricsRefused(context.Background(), refusedPoints)
	obsrep.MetricsDropped(context.Background(), droppedPoints)

	obsreporttest.CheckProcessorMetricsViews(t, processor, acceptedPoints, refusedPoints, droppedPoints)
}

func TestProcessorMetricViews(t *testing.T) {
	measures := []stats.Measure{
		stats.Int64("firstMeasure", "test firstMeasure", stats.UnitDimensionless),
		stats.Int64("secondMeasure", "test secondMeasure", stats.UnitBytes),
	}
	legacyViews := []*view.View{
		{
			Name:        measures[0].Name(),
			Description: measures[0].Description(),
			Measure:     measures[0],
			Aggregation: view.Sum(),
		},
		{
			Measure:     measures[1],
			Aggregation: view.Count(),
		},
	}

	tests := []struct {
		name  string
		level configtelemetry.Level
		want  []*view.View
	}{
		{
			name:  "none",
			level: configtelemetry.LevelNone,
		},
		{
			name:  "basic",
			level: configtelemetry.LevelBasic,
			want: []*view.View{
				{
					Name:        "processor/test_type/" + measures[0].Name(),
					Description: measures[0].Description(),
					Measure:     measures[0],
					Aggregation: view.Sum(),
				},
				{
					Name:        "processor/test_type/" + measures[1].Name(),
					Measure:     measures[1],
					Aggregation: view.Count(),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obsreport.Configure(tt.level)
			got := obsreport.ProcessorMetricViews("test_type", legacyViews)
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

	obsrep := obsreport.NewProcessorObsReport(configtelemetry.LevelNormal, processor)
	obsrep.LogsAccepted(context.Background(), acceptedRecords)
	obsrep.LogsRefused(context.Background(), refusedRecords)
	obsrep.LogsDropped(context.Background(), droppedRecords)

	obsreporttest.CheckProcessorLogsViews(t, processor, acceptedRecords, refusedRecords, droppedRecords)
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
