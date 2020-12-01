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

package ocmetrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	metricSink := new(consumertest.MetricsSink)

	port, doneFn := ocReceiverOnGRPCServer(t, metricSink)
	defer doneFn()

	address := fmt.Sprintf("localhost:%d", port)
	expFactory := opencensusexporter.NewFactory()
	expCfg := expFactory.CreateDefaultConfig().(*opencensusexporter.Config)
	expCfg.GRPCClientSettings.TLSSetting.Insecure = true
	expCfg.Endpoint = address
	expCfg.WaitForReady = true
	oce, err := expFactory.CreateMetricsExporter(context.Background(), component.ExporterCreateParams{Logger: zap.NewNop()}, expCfg)
	require.NoError(t, err)
	err = oce.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	defer func() {
		require.NoError(t, oce.Shutdown(context.Background()))
	}()

	md := testdata.GenerateMetricsOneMetric()
	assert.NoError(t, oce.ConsumeMetrics(context.Background(), md))

	testutil.WaitFor(t, func() bool {
		return len(metricSink.AllMetrics()) != 0
	})
	gotMetrics := metricSink.AllMetrics()
	require.Len(t, gotMetrics, 1)
	assert.Equal(t, md, gotMetrics[0])
}

// Issue #43. Export should support node multiplexing.
// The goal is to ensure that Receiver can always support
// a passthrough mode where it initiates Export normally by firstly
// receiving the initiator node. However ti should still be able to
// accept nodes from downstream sources, but if a node isn't specified in
// an exportMetrics request, assume it is from the last received and non-nil node.
func TestExportMultiplexing(t *testing.T) {
	metricSink := new(consumertest.MetricsSink)

	port, doneFn := ocReceiverOnGRPCServer(t, metricSink)
	defer doneFn()

	metricsClient, metricsClientDoneFn, err := makeMetricsServiceClient(port)
	require.NoError(t, err, "Failed to create the gRPC MetricsService_ExportClient: %v", err)
	defer metricsClientDoneFn()

	// Step 1) The initiation.
	initiatingNode := &commonpb.Node{
		Identifier: &commonpb.ProcessIdentifier{
			Pid:      1,
			HostName: "multiplexer",
		},
		LibraryInfo: &commonpb.LibraryInfo{Language: commonpb.LibraryInfo_JAVA},
	}

	err = metricsClient.Send(&agentmetricspb.ExportMetricsServiceRequest{Node: initiatingNode})
	require.NoError(t, err, "Failed to send the initiating message: %v", err)

	// Step 1a) Send some metrics without a node, they should be registered as coming from the initiating node.
	mLi := []*metricspb.Metric{makeMetric(1)}
	err = metricsClient.Send(&agentmetricspb.ExportMetricsServiceRequest{Node: nil, Metrics: mLi})
	require.NoError(t, err, "Failed to send the proxied message from app1: %v", err)

	// Step 2) Send a "proxied" metrics message from app1 with "node1"
	node1 := &commonpb.Node{
		Identifier:  &commonpb.ProcessIdentifier{Pid: 9489, HostName: "nodejs-host"},
		LibraryInfo: &commonpb.LibraryInfo{Language: commonpb.LibraryInfo_NODE_JS},
	}
	mL1 := []*metricspb.Metric{makeMetric(2)}
	err = metricsClient.Send(&agentmetricspb.ExportMetricsServiceRequest{Node: node1, Metrics: mL1})
	require.NoError(t, err, "Failed to send the proxied message from app1: %v", err)

	// Step 3) Send a metrics message without a node but with metrics: this
	// should be registered as belonging to the last used node i.e. "node1".
	mLn1 := []*metricspb.Metric{makeMetric(3)}
	err = metricsClient.Send(&agentmetricspb.ExportMetricsServiceRequest{Node: nil, Metrics: mLn1})
	require.NoError(t, err, "Failed to send the proxied message without a node: %v", err)

	// Step 4) Send a metrics message from a differently proxied node "node2" from app2
	node2 := &commonpb.Node{
		Identifier:  &commonpb.ProcessIdentifier{Pid: 7752, HostName: "golang-host"},
		LibraryInfo: &commonpb.LibraryInfo{Language: commonpb.LibraryInfo_GO_LANG},
	}
	mL2 := []*metricspb.Metric{makeMetric(4)}
	err = metricsClient.Send(&agentmetricspb.ExportMetricsServiceRequest{Node: node2, Metrics: mL2})
	require.NoError(t, err, "Failed to send the proxied message from app2: %v", err)

	// Step 5a) Send a metrics message without a node but with metrics: this
	// should be registered as belonging to the last used node i.e. "node2".
	mLn2a := []*metricspb.Metric{makeMetric(5)}
	err = metricsClient.Send(&agentmetricspb.ExportMetricsServiceRequest{Node: nil, Metrics: mLn2a})
	require.NoError(t, err, "Failed to send the proxied message without a node: %v", err)

	// Step 5b)
	mLn2b := []*metricspb.Metric{makeMetric(6)}
	err = metricsClient.Send(&agentmetricspb.ExportMetricsServiceRequest{Node: nil, Metrics: mLn2b})
	require.NoError(t, err, "Failed to send the proxied message without a node: %v", err)
	// Give the process sometime to send data over the wire and perform batching
	<-time.After(150 * time.Millisecond)

	// Examination time!
	resultsMapping := make(map[string][]*metricspb.Metric)
	for _, md := range metricSink.AllMetrics() {
		ocmds := internaldata.MetricsToOC(md)
		for _, ocmd := range ocmds {
			resultsMapping[nodeToKey(ocmd.Node)] = append(resultsMapping[nodeToKey(ocmd.Node)], ocmd.Metrics...)
		}
	}

	// First things first, we expect exactly 3 unique keys
	// 1. Initiating Node
	// 2. Node 1
	// 3. Node 2
	if g, w := len(resultsMapping), 3; g != w {
		t.Errorf("Got %d keys in the results map; Wanted exactly %d\n\nResultsMapping: %+v\n", g, w, resultsMapping)
	}

	// Want metric counts
	wantMetricCounts := map[string]int{
		nodeToKey(initiatingNode): 1,
		nodeToKey(node1):          2,
		nodeToKey(node2):          3,
	}
	for key, wantMetricCounts := range wantMetricCounts {
		gotMetricCounts := len(resultsMapping[key])
		if gotMetricCounts != wantMetricCounts {
			t.Errorf("Key=%q gotMetricCounts %d wantMetricCounts %d", key, gotMetricCounts, wantMetricCounts)
		}
	}

	// Now ensure that the exported metrics match up exactly with
	// the nodes and the last seen node expectation/behavior.
	// (or at least their serialized equivalents match up)
	wantContents := map[string][]*metricspb.Metric{
		nodeToKey(initiatingNode): mLi,
		nodeToKey(node1):          append(mL1, mLn1...),
		nodeToKey(node2):          append(mL2, append(mLn2a, mLn2b...)...),
	}

	gotBlob, _ := json.Marshal(resultsMapping)
	wantBlob, _ := json.Marshal(wantContents)
	if !bytes.Equal(gotBlob, wantBlob) {
		t.Errorf("Unequal serialization results\nGot:\n\t%s\nWant:\n\t%s\n", gotBlob, wantBlob)
	}
}

// The first message without a Node MUST be rejected and teardown the connection.
// See https://github.com/census-instrumentation/opencensus-service/issues/53
func TestExportProtocolViolations_nodelessFirstMessage(t *testing.T) {
	metricSink := new(consumertest.MetricsSink)

	port, doneFn := ocReceiverOnGRPCServer(t, metricSink)
	defer doneFn()

	metricsClient, metricsClientDoneFn, err := makeMetricsServiceClient(port)
	require.NoError(t, err, "Failed to create the gRPC MetricsService_ExportClient: %v", err)
	defer metricsClientDoneFn()

	// Send a Nodeless first message
	err = metricsClient.Send(&agentmetricspb.ExportMetricsServiceRequest{Node: nil})
	require.NoError(t, err, "Unexpectedly failed to send the first message: %v", err)

	longDuration := 2 * time.Second
	testDone := make(chan bool, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		// Our insurance policy to ensure that this test doesn't hang
		// forever and should quickly report if/when we regress.
		select {
		case <-testDone:
			t.Log("Test ended early enough")
		case <-time.After(longDuration):
			metricsClientDoneFn()
			t.Errorf("Test took too long (%s) and is likely still hanging so this is a regression", longDuration)
		}
		wg.Done()
	}()

	// Now the response should return an error and should have been torn down
	// regardless of the number of times after invocation below, or any attempt
	// to send the proper/corrective data should be rejected.
	for i := 0; i < 10; i++ {
		recv, err := metricsClient.Recv()
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
		if err = metricsClient.Send(&agentmetricspb.ExportMetricsServiceRequest{Node: n1}); err == nil {
			t.Errorf("Iteration #%d: Unexpectedly succeeded in sending a message upstream. Connection must be in terminal state", i)
		} else if g, w := err, io.EOF; g != w {
			t.Errorf("Iteration #%d:\nGot error %q\nWant error %q", i, g, w)
		}
	}

	close(testDone)
	wg.Wait()
}

// If the first message is valid (has a non-nil Node) and has metrics, those
// metrics should be received and NEVER discarded.
// See https://github.com/census-instrumentation/opencensus-service/issues/51
func TestExportProtocolConformation_metricsInFirstMessage(t *testing.T) {
	// This test used to be flaky on Windows. Skip if errors pop up again

	metricSink := new(consumertest.MetricsSink)

	port, doneFn := ocReceiverOnGRPCServer(t, metricSink)
	defer doneFn()

	metricsClient, metricsClientDoneFn, err := makeMetricsServiceClient(port)
	require.NoError(t, err, "Failed to create the gRPC MetricsService_ExportClient: %v", err)
	defer metricsClientDoneFn()

	mLi := []*metricspb.Metric{makeMetric(10), makeMetric(11)}
	ni := &commonpb.Node{
		Identifier:  &commonpb.ProcessIdentifier{Pid: 1},
		LibraryInfo: &commonpb.LibraryInfo{Language: commonpb.LibraryInfo_JAVA},
	}
	err = metricsClient.Send(&agentmetricspb.ExportMetricsServiceRequest{Node: ni, Metrics: mLi})
	require.NoError(t, err, "Failed to send the first message: %v", err)

	// Give it time to be sent over the wire, then exported.
	<-time.After(100 * time.Millisecond)

	// Examination time!
	resultsMapping := make(map[string][]*metricspb.Metric)
	for _, md := range metricSink.AllMetrics() {
		ocmds := internaldata.MetricsToOC(md)
		for _, ocmd := range ocmds {
			resultsMapping[nodeToKey(ocmd.Node)] = append(resultsMapping[nodeToKey(ocmd.Node)], ocmd.Metrics...)
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
			t.Errorf("Exported metrics:: Key: %s\nGot length %d\nWant length %d", key, gotLength, wantLength)
		}
	}

	// And finally ensure that the protos' serializations are equivalent to the expected
	wantContents := map[string][]*metricspb.Metric{
		nodeToKey(ni): mLi,
	}

	gotBlob, _ := json.Marshal(resultsMapping)
	wantBlob, _ := json.Marshal(wantContents)
	if !bytes.Equal(gotBlob, wantBlob) {
		t.Errorf("Unequal serialization results\nGot:\n\t%s\nWant:\n\t%s\n", gotBlob, wantBlob)
	}
}

// Helper functions from here on below
func makeMetricsServiceClient(port int) (agentmetricspb.MetricsService_ExportClient, func(), error) {
	addr := fmt.Sprintf(":%d", port)
	cc, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	svc := agentmetricspb.NewMetricsServiceClient(cc)
	metricsClient, err := svc.Export(context.Background())
	if err != nil {
		_ = cc.Close()
		return nil, nil, err
	}

	doneFn := func() { _ = cc.Close() }
	return metricsClient, doneFn, nil
}

func nodeToKey(n *commonpb.Node) string {
	blob, _ := proto.Marshal(n)
	return string(blob)
}

func ocReceiverOnGRPCServer(t *testing.T, sr consumer.MetricsConsumer) (int, func()) {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	doneFnList := []func(){func() { ln.Close() }}
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
	agentmetricspb.RegisterMetricsServiceServer(srv, oci)
	go func() {
		_ = srv.Serve(ln)
	}()

	return port, done
}

func makeMetric(val int) *metricspb.Metric {
	key := &metricspb.LabelKey{
		Key: fmt.Sprintf("%s%d", "key", val),
	}
	value := &metricspb.LabelValue{
		Value:    fmt.Sprintf("%s%d", "value", val),
		HasValue: true,
	}

	descriptor := &metricspb.MetricDescriptor{
		Name:        fmt.Sprintf("%s%d", "metric_descriptort_", val),
		Description: "metric descriptor",
		Unit:        "1",
		Type:        metricspb.MetricDescriptor_GAUGE_INT64,
		LabelKeys:   []*metricspb.LabelKey{key},
	}

	now := time.Now().UTC()
	point := &metricspb.Point{
		Timestamp: timestamppb.New(now.Add(20 * time.Second)),
		Value: &metricspb.Point_Int64Value{
			Int64Value: int64(val),
		},
	}

	ts := &metricspb.TimeSeries{
		StartTimestamp: timestamppb.New(now.Add(-10 * time.Second)),
		LabelValues:    []*metricspb.LabelValue{value},
		Points:         []*metricspb.Point{point},
	}

	return &metricspb.Metric{
		MetricDescriptor: descriptor,
		Timeseries:       []*metricspb.TimeSeries{ts},
	}
}
