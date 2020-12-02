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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter/opencensusexporter"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/testutil"
	"go.opentelemetry.io/collector/translator/internaldata"
)

func TestReceiver_endToEnd(t *testing.T) {
	spanSink := new(consumertest.TracesSink)

	port, doneFn := ocReceiverOnGRPCServer(t, spanSink)
	defer doneFn()

	address := fmt.Sprintf("localhost:%d", port)
	expFactory := opencensusexporter.NewFactory()
	expCfg := expFactory.CreateDefaultConfig().(*opencensusexporter.Config)
	expCfg.GRPCClientSettings.TLSSetting.Insecure = true
	expCfg.Endpoint = address
	expCfg.WaitForReady = true
	oce, err := expFactory.CreateTracesExporter(context.Background(), component.ExporterCreateParams{Logger: zap.NewNop()}, expCfg)
	require.NoError(t, err)
	err = oce.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	defer func() {
		require.NoError(t, oce.Shutdown(context.Background()))
	}()

	td := testdata.GenerateTraceDataOneSpan()
	assert.NoError(t, oce.ConsumeTraces(context.Background(), td))

	testutil.WaitFor(t, func() bool {
		return len(spanSink.AllTraces()) != 0
	})
	gotTraces := spanSink.AllTraces()
	require.Len(t, gotTraces, 1)
	assert.Equal(t, td, gotTraces[0])
}

// Issue #43. Export should support node multiplexing.
// The goal is to ensure that Receiver can always support
// a passthrough mode where it initiates Export normally by firstly
// receiving the initiator node. However ti should still be able to
// accept nodes from downstream sources, but if a node isn't specified in
// an exportTrace request, assume it is from the last received and non-nil node.
func TestExportMultiplexing(t *testing.T) {
	spanSink := new(consumertest.TracesSink)

	port, doneFn := ocReceiverOnGRPCServer(t, spanSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeTraceServiceClient(port)
	require.NoError(t, err, "Failed to create the gRPC TraceService_ExportClient: %v", err)
	defer traceClientDoneFn()

	// Step 1) The initiation.
	initiatingNode := &commonpb.Node{
		Identifier: &commonpb.ProcessIdentifier{
			Pid:      1,
			HostName: "multiplexer",
		},
		LibraryInfo: &commonpb.LibraryInfo{Language: commonpb.LibraryInfo_JAVA},
	}

	err = traceClient.Send(&agenttracepb.ExportTraceServiceRequest{Node: initiatingNode})
	require.NoError(t, err, "Failed to send the initiating message: %v", err)

	// Step 1a) Send some spans without a node, they should be registered as coming from the initiating node.
	sLi := []*tracepb.Span{{TraceId: []byte("1234567890abcdef"), Status: &tracepb.Status{}}}
	err = traceClient.Send(&agenttracepb.ExportTraceServiceRequest{Node: nil, Spans: sLi})
	require.NoError(t, err, "Failed to send the proxied message from app1: %v", err)

	// Step 2) Send a "proxied" trace message from app1 with "node1"
	node1 := &commonpb.Node{
		Identifier:  &commonpb.ProcessIdentifier{Pid: 9489, HostName: "nodejs-host"},
		LibraryInfo: &commonpb.LibraryInfo{Language: commonpb.LibraryInfo_NODE_JS},
	}
	sL1 := []*tracepb.Span{{TraceId: []byte("abcdefghijklmnop"), Name: &tracepb.TruncatableString{Value: "test"}, Status: &tracepb.Status{}}}
	err = traceClient.Send(&agenttracepb.ExportTraceServiceRequest{Node: node1, Spans: sL1})
	require.NoError(t, err, "Failed to send the proxied message from app1: %v", err)

	// Step 3) Send a trace message without a node but with spans: this
	// should be registered as belonging to the last used node i.e. "node1".
	sLn1 := []*tracepb.Span{{TraceId: []byte("ABCDEFGHIJKLMNOP"), Status: &tracepb.Status{}}, {TraceId: []byte("1234567890abcdef"), Status: &tracepb.Status{}}}
	err = traceClient.Send(&agenttracepb.ExportTraceServiceRequest{Node: nil, Spans: sLn1})
	require.NoError(t, err, "Failed to send the proxied message without a node: %v", err)

	// Step 4) Send a trace message from a differently proxied node "node2" from app2
	node2 := &commonpb.Node{
		Identifier:  &commonpb.ProcessIdentifier{Pid: 7752, HostName: "golang-host"},
		LibraryInfo: &commonpb.LibraryInfo{Language: commonpb.LibraryInfo_GO_LANG},
	}
	sL2 := []*tracepb.Span{{TraceId: []byte("_B_D_F_H_J_L_N_O"), Status: &tracepb.Status{}}}
	err = traceClient.Send(&agenttracepb.ExportTraceServiceRequest{Node: node2, Spans: sL2})
	require.NoError(t, err, "Failed to send the proxied message from app2: %v", err)

	// Step 5a) Send a trace message without a node but with spans: this
	// should be registered as belonging to the last used node i.e. "node2".
	sLn2a := []*tracepb.Span{{TraceId: []byte("_BCDEFGHIJKLMNO_"), Status: &tracepb.Status{}}, {TraceId: []byte("_234567890abcde_"), Status: &tracepb.Status{}}}
	err = traceClient.Send(&agenttracepb.ExportTraceServiceRequest{Node: nil, Spans: sLn2a})
	require.NoError(t, err, "Failed to send the proxied message without a node: %v", err)

	// Step 5b)
	sLn2b := []*tracepb.Span{{TraceId: []byte("_xxxxxxxxxxxxxx_"), Status: &tracepb.Status{}}, {TraceId: []byte("B234567890abcdAB"), Status: &tracepb.Status{}}}
	err = traceClient.Send(&agenttracepb.ExportTraceServiceRequest{Node: nil, Spans: sLn2b})
	require.NoError(t, err, "Failed to send the proxied message without a node: %v", err)
	// Give the process sometime to send data over the wire and perform batching
	<-time.After(150 * time.Millisecond)

	// Examination time!
	resultsMapping := make(map[string][]*tracepb.Span)
	for _, td := range spanSink.AllTraces() {
		octds := internaldata.TraceDataToOC(td)
		for _, octd := range octds {
			resultsMapping[nodeToKey(octd.Node)] = append(resultsMapping[nodeToKey(octd.Node)], octd.Spans...)
		}
	}

	// First things first, we expect exactly 3 unique keys
	// 1. Initiating Node
	// 2. Node 1
	// 3. Node 2
	if g, w := len(resultsMapping), 3; g != w {
		t.Errorf("Got %d keys in the results map; Wanted exactly %d\n\nResultsMapping: %+v\n", g, w, resultsMapping)
	}

	// Want span counts
	wantSpanCounts := map[string]int{
		nodeToKey(initiatingNode): 1,
		nodeToKey(node1):          3,
		nodeToKey(node2):          5,
	}
	for key, wantSpanCounts := range wantSpanCounts {
		gotSpanCounts := len(resultsMapping[key])
		if gotSpanCounts != wantSpanCounts {
			t.Errorf("Key=%q gotSpanCounts %d wantSpanCounts %d", key, gotSpanCounts, wantSpanCounts)
		}
	}

	// Now ensure that the exported spans match up exactly with
	// the nodes and the last seen node expectation/behavior.
	// (or at least their serialized equivalents match up)
	wantContents := map[string][]*tracepb.Span{
		nodeToKey(initiatingNode): sLi,
		nodeToKey(node1):          append(sL1, sLn1...),
		nodeToKey(node2):          append(sL2, append(sLn2a, sLn2b...)...),
	}

	for nodeKey, wantSpans := range wantContents {
		gotSpans, ok := resultsMapping[nodeKey]
		if !ok {
			t.Errorf("Wanted to find a node that was not found for key: %s", nodeKey)
		}
		if len(gotSpans) != len(wantSpans) {
			t.Errorf("Unequal number of spans for nodeKey: %s", nodeKey)
		}
		for _, wantSpan := range wantSpans {
			found := false
			for _, gotSpan := range gotSpans {
				wantStr, _ := json.Marshal(wantSpan)
				gotStr, _ := json.Marshal(gotSpan)
				if bytes.Equal(wantStr, gotStr) {
					found = true
				}
			}
			if !found {
				t.Errorf("Unequal span serialization\nGot:\n\t%s\nWant:\n\t%s\n", gotSpans, wantSpans)
			}
		}
	}
}

// The first message without a Node MUST be rejected and teardown the connection.
// See https://github.com/census-instrumentation/opencensus-service/issues/53
func TestExportProtocolViolations_nodelessFirstMessage(t *testing.T) {
	spanSink := new(consumertest.TracesSink)

	port, doneFn := ocReceiverOnGRPCServer(t, spanSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeTraceServiceClient(port)
	require.NoError(t, err, "Failed to create the gRPC TraceService_ExportClient: %v", err)
	defer traceClientDoneFn()

	// Send a Nodeless first message
	err = traceClient.Send(&agenttracepb.ExportTraceServiceRequest{Node: nil})
	require.NoError(t, err, "Unexpectedly failed to send the first message: %v", err)

	longDuration := 2 * time.Second
	testDone := make(chan struct{})
	goroutineDone := make(chan struct{})
	go func() {
		// Our insurance policy to ensure that this test doesn't hang
		// forever and should quickly report if/when we regress.
		select {
		case <-testDone:
			t.Log("Test ended early enough")
		case <-time.After(longDuration):
			traceClientDoneFn()
			t.Errorf("Test took too long (%s) and is likely still hanging so this is a regression", longDuration)
		}
		close(goroutineDone)
	}()

	// Now the response should return an error and should have been torn down
	// regardless of the number of times after invocation below, or any attempt
	// to send the proper/corrective data should be rejected.
	for i := 0; i < 10; i++ {
		recv, err := traceClient.Recv()
		if recv != nil {
			t.Errorf("Iteration #%d: Unexpectedly got back a response: %#v", i, recv)
		}
		if err == nil {
			t.Errorf("Iteration #%d: Unexpectedly got back a nil error", i)
			continue
		}

		wantSubStr := "protocol violation: Export's first message must have a Node"
		if g := err.Error(); !strings.Contains(g, wantSubStr) {
			t.Errorf("Iteration #%d: Got error:\n\t%s\nWant substring:\n\t%s\n", i, g, wantSubStr)
		}

		// The connection should be invalid at this point and
		// no attempt to send corrections should succeed.
		n1 := &commonpb.Node{
			Identifier:  &commonpb.ProcessIdentifier{Pid: 9489, HostName: "nodejs-host"},
			LibraryInfo: &commonpb.LibraryInfo{Language: commonpb.LibraryInfo_NODE_JS},
		}
		if err = traceClient.Send(&agenttracepb.ExportTraceServiceRequest{Node: n1}); err == nil {
			t.Errorf("Iteration #%d: Unexpectedly succeeded in sending a message upstream. Connection must be in terminal state", i)
		} else if g, w := err, io.EOF; g != w {
			t.Errorf("Iteration #%d:\nGot error %q\nWant error %q", i, g, w)
		}
	}

	close(testDone)
	<-goroutineDone
}

// If the first message is valid (has a non-nil Node) and has spans, those
// spans should be received and NEVER discarded.
// See https://github.com/census-instrumentation/opencensus-service/issues/51
func TestExportProtocolConformation_spansInFirstMessage(t *testing.T) {
	spanSink := new(consumertest.TracesSink)

	port, doneFn := ocReceiverOnGRPCServer(t, spanSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeTraceServiceClient(port)
	require.NoError(t, err, "Failed to create the gRPC TraceService_ExportClient: %v", err)
	defer traceClientDoneFn()

	sLi := []*tracepb.Span{
		{TraceId: []byte("1234567890abcdef"), Status: &tracepb.Status{}},
		{TraceId: []byte("XXXXXXXXXXabcdef"), Status: &tracepb.Status{}},
	}
	ni := &commonpb.Node{
		Identifier:  &commonpb.ProcessIdentifier{Pid: 1},
		LibraryInfo: &commonpb.LibraryInfo{Language: commonpb.LibraryInfo_JAVA},
	}
	err = traceClient.Send(&agenttracepb.ExportTraceServiceRequest{Node: ni, Spans: sLi})
	require.NoError(t, err, "Failed to send the first message: %v", err)

	// Give it time to be sent over the wire, then exported.
	<-time.After(100 * time.Millisecond)

	// Examination time!
	resultsMapping := make(map[string][]*tracepb.Span)
	for _, td := range spanSink.AllTraces() {
		octds := internaldata.TraceDataToOC(td)
		for _, octd := range octds {
			resultsMapping[nodeToKey(octd.Node)] = append(resultsMapping[nodeToKey(octd.Node)], octd.Spans...)
		}
	}

	if g, w := len(resultsMapping), 1; g != w {
		t.Errorf("Results mapping: Got len(keys) %d Want %d", g, w)
	}

	// Check for the keys
	wantLengths := map[string]int{
		nodeToKey(ni): 2,
	}
	for key, wantLength := range wantLengths {
		gotLength := len(resultsMapping[key])
		if gotLength != wantLength {
			t.Errorf("Exported spans:: Key: %s\nGot length %d\nWant length %d", key, gotLength, wantLength)
		}
	}

	// And finally ensure that the protos' serializations are equivalent to the expected
	wantContents := map[string][]*tracepb.Span{
		nodeToKey(ni): sLi,
	}

	gotBlob, _ := json.Marshal(resultsMapping)
	wantBlob, _ := json.Marshal(wantContents)
	if !bytes.Equal(gotBlob, wantBlob) {
		t.Errorf("Unequal serialization results\nGot:\n\t%s\nWant:\n\t%s\n", gotBlob, wantBlob)
	}
}

// Helper functions from here on below
func makeTraceServiceClient(port int) (agenttracepb.TraceService_ExportClient, func(), error) {
	addr := fmt.Sprintf(":%d", port)
	cc, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	svc := agenttracepb.NewTraceServiceClient(cc)
	traceClient, err := svc.Export(context.Background())
	if err != nil {
		_ = cc.Close()
		return nil, nil, err
	}

	doneFn := func() { _ = cc.Close() }
	return traceClient, doneFn, nil
}

func nodeToKey(n *commonpb.Node) string {
	blob, _ := proto.Marshal(n)
	return string(blob)
}

func ocReceiverOnGRPCServer(t *testing.T, sr consumer.TracesConsumer) (int, func()) {
	ln, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	doneFnList := []func(){func() { require.NoError(t, ln.Close()) }}
	done := func() {
		for _, doneFn := range doneFnList {
			doneFn()
		}
	}

	_, port, err := testutil.HostPortFromAddr(ln.Addr())
	if err != nil {
		done()
		t.Fatalf("Failed to parse host:port from listener address: %s error: %v", ln.Addr(), err)
	}

	oci, err := New(receiverTagValue, sr)
	require.NoError(t, err, "Failed to create the Receiver: %v", err)

	// Now run it as a gRPC server
	srv := obsreport.GRPCServerWithObservabilityEnabled()
	agenttracepb.RegisterTraceServiceServer(srv, oci)
	go func() {
		_ = srv.Serve(ln)
	}()

	return port, done
}
