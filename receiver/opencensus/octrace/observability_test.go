// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package octrace_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensus/octrace"
)

// Ensure that if we add a metrics exporter that our target metrics
// will be recorded but also with the proper tag keys and values.
// See Issue https://github.com/census-instrumentation/opencensus-service/issues/63
//
// Note: we are intentionally skipping the ocgrpc.ServerDefaultViews as this
// test is to ensure exactness, but with the mentioned views registered, the
// output will be quite noisy.
func TestEnsureRecordedMetrics(t *testing.T) {
	t.Skip("Depends on precising timing in OpenCensus-Go's stats worker but that timing is thrown off by slower -race binaries")

	sappender := newSpanAppender()

	_, port, doneFn := ocReceiverOnGRPCServer(t, sappender, octrace.WithSpanBufferPeriod(2*time.Millisecond))
	defer doneFn()

	// Now the opencensus-agent exporter.
	address := fmt.Sprintf("localhost:%d", port)
	oce, err := ocagent.NewExporter(ocagent.WithAddress(address), ocagent.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to create the ocagent-exporter: %v", err)
	}
	trace.RegisterExporter(oce)

	metricsReportingPeriod := 5 * time.Millisecond
	view.SetReportingPeriod(metricsReportingPeriod)
	// On exit, revert the metrics reporting period.

	defer func() {
		oce.Stop()

		// Pause for a bit before exiting to give OpenCensus-Go trace
		// some time to export any remaining traces, before we unregister
		// the exporter.
		<-time.After(5 * metricsReportingPeriod)

		trace.UnregisterExporter(oce)
		view.SetReportingPeriod(60 * time.Second)
	}()

	// Now for the stats exporter
	if err := view.Register(internal.AllViews...); err != nil {
		t.Fatalf("Failed to register all views: %v", err)
	}
	defer view.Unregister(internal.AllViews...)

	cme := newCountMetricsExporter()
	view.RegisterExporter(cme)
	defer view.UnregisterExporter(cme)

	n := 20
	// Now it is time to send over some spans
	// and we'll count the numbers received.
	for i := 0; i < n; i++ {
		now := time.Now().UTC()
		oce.ExportSpan(&trace.SpanData{
			StartTime: now.Add(-10 * time.Second),
			EndTime:   now.Add(20 * time.Second),
			SpanContext: trace.SpanContext{
				TraceID:      trace.TraceID{byte(0x20 + i), 0x4E, 0x4D, 0x4C, 0x4B, 0x4A, 0x49, 0x48, 0x47, 0x46, 0x45, 0x44, 0x43, 0x42, 0x41},
				SpanID:       trace.SpanID{0x7F, 0x7E, 0x7D, 0x7C, 0x7B, 0x7A, 0x79, 0x78},
				TraceOptions: trace.TraceOptions(i & 0x01),
			},
			ParentSpanID: trace.SpanID{byte(0x01 + i), 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37},
			Name:         fmt.Sprintf("Span-%d", i),
			Status:       trace.Status{Code: trace.StatusCodeInternal, Message: "Blocked by firewall"},
		})
	}

	// Give them some time to be exported.
	// say n * metricsReportingPeriod
	<-time.After(time.Duration(n+1) * metricsReportingPeriod)
	oce.Flush()

	checkCountMetricsExporterResults(t, cme, n, 1)
}

func TestEnsureRecordedMetrics_zeroLengthSpansSender(t *testing.T) {
	t.Skipf("Currently disabled, enable this test when the following are fixed:\nIssue %s\nPR %s",
		"https://github.com/census-instrumentation/opencensus-go/issues/862",
		"https://github.com/census-instrumentation/opencensus-go/pull/922",
	)
	sappender := newSpanAppender()

	_, port, doneFn := ocReceiverOnGRPCServer(t, sappender, octrace.WithSpanBufferPeriod(2*time.Millisecond))
	defer doneFn()

	// Now the opencensus-agent exporter.
	address := fmt.Sprintf("localhost:%d", port)
	oce, err := ocagent.NewExporter(ocagent.WithAddress(address), ocagent.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to create the ocagent-exporter: %v", err)
	}
	trace.RegisterExporter(oce)
	defer func() {
		oce.Stop()
		trace.UnregisterExporter(oce)
	}()

	// Now for the stats exporter
	if err := view.Register(internal.AllViews...); err != nil {
		t.Fatalf("Failed to register all views: %v", err)
	}
	defer view.Unregister(internal.AllViews...)

	metricsReportingPeriod := 10 * time.Millisecond
	view.SetReportingPeriod(metricsReportingPeriod)
	// On exit, revert the metrics reporting period.
	defer view.SetReportingPeriod(60 * time.Second)

	cme := newCountMetricsExporter()
	view.RegisterExporter(cme)
	defer view.UnregisterExporter(cme)

	n := 20
	// Now for the traceExporter that sends 0 length spans
	traceSvcClient, traceSvcDoneFn, err := makeTraceServiceClient(port)
	if err != nil {
		t.Fatalf("Failed to create the trace service client: %v", err)
	}
	defer traceSvcDoneFn()
	for i := 0; i <= n; i++ {
		_ = traceSvcClient.Send(&agenttracepb.ExportTraceServiceRequest{Spans: nil, Node: &commonpb.Node{}})
	}
	<-time.After(time.Duration(n) * metricsReportingPeriod)
	checkCountMetricsExporterResults(t, cme, n, 0)
}

type testOCTraceExporter struct {
	mu       sync.Mutex
	spanData []*trace.SpanData
}

func (tote *testOCTraceExporter) ExportSpan(sd *trace.SpanData) {
	tote.mu.Lock()
	defer tote.mu.Unlock()

	tote.spanData = append(tote.spanData, sd)
}

func TestExportSpanLinkingMaintainsParentLink(t *testing.T) {
	// Always sample for the purpose of examining all the spans in this test.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	// TODO: File an issue with OpenCensus-Go to ask for a method to retrieve
	// the default sampler because the current method of blindly changing the
	// global sampler makes testing hard.
	// Denoise this test by setting the sampler to never sample
	defer trace.ApplyConfig(trace.Config{DefaultSampler: trace.NeverSample()})

	ocSpansSaver := new(testOCTraceExporter)
	trace.RegisterExporter(ocSpansSaver)
	defer trace.UnregisterExporter(ocSpansSaver)

	spanSink := newSpanAppender()
	spansBufferPeriod := 10 * time.Millisecond
	_, port, doneFn := ocReceiverOnGRPCServer(t, spanSink, octrace.WithSpanBufferPeriod(spansBufferPeriod))
	defer doneFn()

	traceSvcClient, traceSvcDoneFn, err := makeTraceServiceClient(port)
	if err != nil {
		t.Fatalf("Failed to create the trace service client: %v", err)
	}
	defer traceSvcDoneFn()

	n := 5
	for i := 0; i <= n; i++ {
		sl := []*tracepb.Span{{TraceId: []byte("abcdefghijklmnop"), SpanId: []byte{byte(i + 1), 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}}
		_ = traceSvcClient.Send(&agenttracepb.ExportTraceServiceRequest{Spans: sl, Node: &commonpb.Node{}})
	}

	// Give it enough time to process the streamed spans.
	<-time.After(spansBufferPeriod * 2)

	// End the gRPC service to complete the RPC trace so that we
	// can examine the RPC trace as well.
	traceSvcDoneFn()

	// Give it some more time to complete the RPC trace and export its spanData.
	<-time.After(spansBufferPeriod * 2)

	// Inspection time!
	ocSpansSaver.mu.Lock()
	defer ocSpansSaver.mu.Unlock()

	if len(ocSpansSaver.spanData) == 0 {
		t.Fatal("Unfortunately did not receive an exported span data. Please check this library's implementation or go.opencensus.io/trace")
	}

	gotSpanData := ocSpansSaver.spanData[:]
	if g, w := len(gotSpanData), 2; g != w {
		blob, _ := json.MarshalIndent(gotSpanData, "  ", " ")
		t.Fatalf("Spandata count: Got %d Want %d\n\nData: %s", g, w, blob)
	}

	receiverSpanData := gotSpanData[0]
	if g, w := len(receiverSpanData.Links), 1; g != w {
		t.Fatalf("Links count: Got %d Want %d\nGotSpanData: %#v", g, w, receiverSpanData)
	}

	rpcSpanData := gotSpanData[1]

	// Ensure that the link matches up exactly!
	wantLink := trace.Link{
		SpanID:  rpcSpanData.SpanID,
		TraceID: rpcSpanData.TraceID,
		Type:    trace.LinkTypeParent,
	}
	if g, w := receiverSpanData.Links[0], wantLink; !reflect.DeepEqual(g, w) {
		t.Errorf("Link:\nGot: %#v\nWant: %#v\n", g, w)
	}
	if g, w := receiverSpanData.Name, "OpenCensusReceiver.Export"; g != w {
		t.Errorf("ReceiverExport span's SpanData.Name:\nGot:  %q\nWant: %q\n", g, w)
	}

	// And then for the receiverSpanData itself, it SHOULD NOT
	// have a ParentID, so let's enforce all the conditions below:
	// 1. That it doesn't have the RPC spanID as its ParentSpanID
	// 2. That it actually has no ParentSpanID i.e. has a blank SpanID
	if g, w := receiverSpanData.ParentSpanID[:], rpcSpanData.SpanID[:]; bytes.Equal(g, w) {
		t.Errorf("ReceiverSpanData.ParentSpanID unfortunately was linked to the RPC span\nGot:  %x\nWant: %x", g, w)
	}

	var blankSpanID trace.SpanID
	if g, w := receiverSpanData.ParentSpanID[:], blankSpanID[:]; !bytes.Equal(g, w) {
		t.Errorf("ReceiverSpanData unfortunately has a parent and isn't NULL\nGot:  %x\nWant: %x", g, w)
	}
}

func checkCountMetricsExporterResults(t *testing.T, cme *countMetricsExporter, n int, wantAllCountsToBe int64) {
	cme.mu.Lock()
	defer cme.mu.Unlock()

	// The only tags that we are expecting are "opencensus_receiver": "opencensus" * n
	wantTagKey, _ := tag.NewKey("opencensus_receiver")
	valuesPlusBlank := strings.Split(strings.Repeat("opencensus,opencensus,", n/2), ",")
	wantValues := valuesPlusBlank[:len(valuesPlusBlank)-1]
	wantTags := map[tag.Key][]string{
		wantTagKey: wantValues,
	}
	gotTags := cme.tags

	if len(gotTags) != len(wantTags) {
		t.Errorf("\nGotTags:\n\t%#v\n\nWantTags:\n\t%#v\n", gotTags, wantTags)
	} else {
		for gotTagKey, gotTagValues := range gotTags {
			wantValues := wantTags[gotTagKey]
			if g, w := len(gotTagValues), len(wantValues); g < w {
				t.Errorf("%s Got less tags than expected %d vs %d\nGot:\n\t%#v\nWant:\n\t%#v\n",
					gotTagKey, g, w, gotTagValues, wantValues)
				continue
			}

			// Because the test for exported metrics depends on precise timings of the results of
			// a non-deterministic global OpenCensus stats worker, the best that we can do here
			// is to compare and ensure that we got at least len(wantValues) that all match.
			// In cases such as with: -race enabled(which slows the program down), we'll get an
			// extra key or so, as per:
			//      https://travis-ci.org/census-instrumentation/opencensus-service/builds/442325321
			// Hence why we'll just compare at most len(wantValues) in case the worker's export time
			// exceeded that of the timings here.
			if g, w := gotTagValues[:len(wantValues)], wantValues; !reflect.DeepEqual(g, w) {
				t.Errorf("\nGotTags:\n\t%#v\n\nWantTags:\n\t%#v\n", gotTags, wantTags)
			}
		}
	}

	// The only data types we are expecting are:
	// * DistributionData
	for key, aggregation := range cme.data {
		switch agg := aggregation.(type) {
		case *view.DistributionData:
			if g, w := agg.Count, int64(1); g != w {
				t.Errorf("Data point #%d GotCount %d Want %d", key, g, w)
			}
		default:
			t.Errorf("Data point #%d Got %T want %T", key, agg, (*view.DistributionData)(nil))
		}
	}
}

type countMetricsExporter struct {
	mu   sync.Mutex
	tags map[tag.Key][]string
	data map[int]view.AggregationData
}

func newCountMetricsExporter() *countMetricsExporter {
	return &countMetricsExporter{
		tags: make(map[tag.Key][]string),
		data: make(map[int]view.AggregationData),
	}
}

func (cme *countMetricsExporter) clear() {
	cme.mu.Lock()
	defer cme.mu.Unlock()

	cme.data = make(map[int]view.AggregationData)
	cme.tags = make(map[tag.Key][]string)
}

var _ view.Exporter = (*countMetricsExporter)(nil)

func (cme *countMetricsExporter) ExportView(vd *view.Data) {
	cme.mu.Lock()
	defer cme.mu.Unlock()

	for _, row := range vd.Rows {
		cme.data[len(cme.data)] = row.Data
		for _, tag_ := range row.Tags {
			cme.tags[tag_.Key] = append(cme.tags[tag_.Key], tag_.Value)
		}
	}
}
