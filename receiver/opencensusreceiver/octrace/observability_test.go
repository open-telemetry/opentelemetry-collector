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

package octrace

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

// Ensure that if we add a metrics exporter that our target metrics
// will be recorded but also with the proper tag keys and values.
// See Issue https://github.com/census-instrumentation/opencensus-service/issues/63
//
// Note: we are intentionally skipping the ocgrpc.ServerDefaultViews as this
// test is to ensure exactness, but with the mentioned views registered, the
// output will be quite noisy.
func TestEnsureRecordedMetrics(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	port, doneReceiverFn := ocReceiverOnGRPCServer(t, consumertest.NewTracesNop())
	defer doneReceiverFn()

	n := 20
	// Now for the traceExporter that sends 0 length spans
	traceSvcClient, traceSvcDoneFn, err := makeTraceServiceClient(port)
	require.NoError(t, err, "Failed to create the trace service client: %v", err)
	spans := []*tracepb.Span{{TraceId: []byte("abcdefghijklmnop"), SpanId: []byte("12345678")}}
	for i := 0; i < n; i++ {
		err = traceSvcClient.Send(&agenttracepb.ExportTraceServiceRequest{Spans: spans, Node: &commonpb.Node{}})
		require.NoError(t, err, "Failed to send requests to the service: %v", err)
	}
	flush(traceSvcDoneFn)

	obsreporttest.CheckReceiverTracesViews(t, "oc_trace", "grpc", int64(n), 0)
}

func TestEnsureRecordedMetrics_zeroLengthSpansSender(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	port, doneFn := ocReceiverOnGRPCServer(t, consumertest.NewTracesNop())
	defer doneFn()

	n := 20
	// Now for the traceExporter that sends 0 length spans
	traceSvcClient, traceSvcDoneFn, err := makeTraceServiceClient(port)
	require.NoError(t, err, "Failed to create the trace service client: %v", err)
	for i := 0; i <= n; i++ {
		err = traceSvcClient.Send(&agenttracepb.ExportTraceServiceRequest{Spans: nil, Node: &commonpb.Node{}})
		require.NoError(t, err, "Failed to send requests to the service: %v", err)
	}
	flush(traceSvcDoneFn)

	obsreporttest.CheckReceiverTracesViews(t, "oc_trace", "grpc", 0, 0)
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

	port, doneFn := ocReceiverOnGRPCServer(t, consumertest.NewTracesNop())
	defer doneFn()

	traceSvcClient, traceSvcDoneFn, err := makeTraceServiceClient(port)
	require.NoError(t, err, "Failed to create the trace service client: %v", err)

	n := 5
	for i := 0; i < n; i++ {
		sl := []*tracepb.Span{{TraceId: []byte("abcdefghijklmnop"), SpanId: []byte{byte(i + 1), 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}}
		err = traceSvcClient.Send(&agenttracepb.ExportTraceServiceRequest{Spans: sl, Node: &commonpb.Node{}})
		require.NoError(t, err, "Failed to send requests to the service: %v", err)
	}

	flush(traceSvcDoneFn)

	// Inspection time!
	ocSpansSaver.mu.Lock()
	defer ocSpansSaver.mu.Unlock()

	require.NotEqual(
		t,
		len(ocSpansSaver.spanData),
		0,
		"Unfortunately did not receive an exported span data. Please check this library's implementation or go.opencensus.io/trace",
	)

	gotSpanData := ocSpansSaver.spanData
	if g, w := len(gotSpanData), n+1; g != w {
		blob, _ := json.MarshalIndent(gotSpanData, "  ", " ")
		t.Fatalf("Spandata count: Got %d Want %d\n\nData: %s", g, w, blob)
	}

	receiverSpanData := gotSpanData[0]
	if g, w := len(receiverSpanData.Links), 1; g != w {
		t.Fatalf("Links count: Got %d Want %d\nGotSpanData: %#v", g, w, receiverSpanData)
	}

	// The rpc span is always last in the list
	rpcSpanData := gotSpanData[len(gotSpanData)-1]

	// Ensure that the link matches up exactly!
	wantLink := trace.Link{
		SpanID:  rpcSpanData.SpanID,
		TraceID: rpcSpanData.TraceID,
		Type:    trace.LinkTypeParent,
	}
	if g, w := receiverSpanData.Links[0], wantLink; !reflect.DeepEqual(g, w) {
		t.Errorf("Link:\nGot: %#v\nWant: %#v\n", g, w)
	}
	if g, w := receiverSpanData.Name, "receiver/oc_trace/TraceDataReceived"; g != w {
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

type testOCTraceExporter struct {
	mu       sync.Mutex
	spanData []*trace.SpanData
}

func (tote *testOCTraceExporter) ExportSpan(sd *trace.SpanData) {
	tote.mu.Lock()
	defer tote.mu.Unlock()

	tote.spanData = append(tote.spanData, sd)
}

// TODO: Determine how to do this deterministic.
func flush(traceSvcDoneFn func()) {
	// Give it enough time to process the streamed spans.
	<-time.After(40 * time.Millisecond)

	// End the gRPC service to complete the RPC trace so that we
	// can examine the RPC trace as well.
	traceSvcDoneFn()

	// Give it some more time to complete the RPC trace and export.
	<-time.After(40 * time.Millisecond)
}
