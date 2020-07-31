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
	"testing"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/api/global"

	"go.opentelemetry.io/collector/exporter/exportertest"
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

	port, doneReceiverFn := ocReceiverOnGRPCServer(t, exportertest.NewNopTraceExporter(), global.TraceProvider())
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

	port, doneFn := ocReceiverOnGRPCServer(t, exportertest.NewNopTraceExporter(), global.TraceProvider())
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
	tProvider, ime := obsreporttest.SetupSdkTraceProviderTest(t)
	port, doneFn := ocReceiverOnGRPCServer(t, exportertest.NewNopTraceExporter(), tProvider)
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
	gotSpanData := ime.GetSpans()
	require.Len(t, gotSpanData, n+1)

	receiverSpanData := gotSpanData[0]
	require.Len(t, receiverSpanData.Links, 1)
	assert.Equal(t, "receiver/oc_trace/TraceDataReceived", receiverSpanData.Name)

	// The rpc span is always last in the list
	rpcSpanData := gotSpanData[len(gotSpanData)-1]
	assert.Equal(t, rpcSpanData.SpanContext, receiverSpanData.Links[0].SpanContext)

	// And then for the receiverSpanData itself, it SHOULD NOT
	// have a ParentID, so let's enforce all the conditions below:
	// 1. That it doesn't have the RPC spanID as its ParentSpanID
	// 2. That it actually has no ParentSpanID i.e. has a blank SpanID
	assert.NotEqual(t, rpcSpanData.SpanContext.SpanID, receiverSpanData.ParentSpanID[:])
	assert.False(t, receiverSpanData.ParentSpanID.IsValid())
}

// TODO: Determine how to do this deterministic.
func flush(traceSvcDoneFn func()) {
	// Give it enough time to process the streamed spans.
	<-time.After(20 * time.Millisecond)

	// End the gRPC service to complete the RPC trace so that we
	// can examine the RPC trace as well.
	traceSvcDoneFn()

	// Give it some more time to complete the RPC trace and export.
	<-time.After(20 * time.Millisecond)
}
