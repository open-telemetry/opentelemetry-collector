// Copyright 2019, OpenTelemetry Authors
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

package ocagent

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	"github.com/open-telemetry/opentelemetry-service/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-service/internal/testutils"
	"github.com/open-telemetry/opentelemetry-service/receiver/opencensusreceiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/receivertest"
)

func TestNewExporter_invokeStartThenStopManyTimes(t *testing.T) {
	ma := runMockAgent(t)
	defer ma.stop()

	exp, err := NewExporter(WithInsecure(),
		WithReconnectionPeriod(50*time.Millisecond),
		WithAddress(ma.address))
	if err != nil {
		t.Fatal("Surprisingly connected with a bad port")
	}
	defer exp.Stop()

	// Invoke Start numerous times, should return errAlreadyStarted
	for i := 0; i < 10; i++ {
		if err := exp.Start(); err == nil || !strings.Contains(err.Error(), "already started") {
			t.Errorf("#%d unexpected Start error: %v", i, err)
		}
	}

	exp.Stop()
	// Invoke Stop numerous times
	for i := 0; i < 10; i++ {
		if err := exp.Stop(); err == nil || !strings.Contains(err.Error(), "not started") {
			t.Errorf(`#%d got error (%v) expected a "not started error"`, i, err)
		}
	}
}

// This test takes a long time to run: to skip it, run tests using: -short
func TestNewExporter_agentOnBadConnection(t *testing.T) {
	if testing.Short() {
		t.Skipf("Skipping this long running test")
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to grab an available port: %v", err)
	}
	// Firstly close the "agent's" channel: optimistically this address won't get reused ASAP
	// However, our goal of closing it is to simulate an unavailable connection
	ln.Close()

	_, agentPortStr, _ := net.SplitHostPort(ln.Addr().String())

	address := fmt.Sprintf("localhost:%s", agentPortStr)
	exp, err := NewExporter(WithInsecure(),
		WithReconnectionPeriod(50*time.Millisecond),
		WithAddress(address))
	if err != nil {
		t.Fatalf("Despite an indefinite background reconnection, got error: %v", err)
	}
	defer exp.Stop()
}

func TestNewExporter_withAddress(t *testing.T) {
	ma := runMockAgent(t)
	defer ma.stop()

	exp, err := NewUnstartedExporter(
		WithInsecure(),
		WithReconnectionPeriod(50*time.Millisecond),
		WithAddress(ma.address))
	if err != nil {
		t.Fatal("Surprisingly connected with a bad port")
	}
	defer exp.Stop()

	if err := exp.Start(); err != nil {
		t.Fatalf("Unexpected Start error: %v", err)
	}
}

func TestExporter_ExportTraceServiceRequest(t *testing.T) {
	addr := testutils.GetAvailableLocalAddress(t)

	exp, err := NewExporter(WithAddress(addr), WithInsecure())
	if err != nil {
		t.Fatalf("cannot create exporter: %v", err)
	}

	ste := new(exportertest.SinkTraceExporter)
	rcv, err := opencensusreceiver.New(addr, ste, nil)
	if err != nil {
		t.Fatalf("cannot create receiver: %v", err)
	}

	mh := receivertest.NewMockHost()
	err = rcv.StartTraceReception(mh)
	if err != nil {
		t.Fatalf("cannot start receiver: %v", err)
	}
	defer rcv.StopTraceReception()

	// nil and empty batch should not cause error
	err = exp.ExportTraceServiceRequest(nil)
	if err != nil {
		t.Fatalf("nil request should not generate error %v", err)
	}
	err = exp.ExportTraceServiceRequest(&agenttracepb.ExportTraceServiceRequest{})
	if err != nil {
		t.Fatalf("empty request should not generate error %v", err)
	}

	extd := &agenttracepb.ExportTraceServiceRequest{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "test-svc"},
		},
		Spans: []*tracepb.Span{
			{Name: &tracepb.TruncatableString{Value: "00"}},
			{Name: &tracepb.TruncatableString{Value: "01"}},
		},
	}
	err = exp.ExportTraceServiceRequest(extd)
	if err != nil {
		t.Fatalf("failed to send data: %v", err)
	}

	// TODO: use a synchronization mechanism instead of time.After.
	<-time.After(100 * time.Millisecond)

	tds := ste.AllTraces()
	if len(tds) != 1 {
		t.Fatalf("want 1 trace data element, got %d", len(tds))
	}
	if len(tds[0].Spans) != len(extd.Spans) {
		t.Fatalf("want to receive %d spans, got %d", len(extd.Spans), len(tds[0].Spans))
	}
}

func TestExporter_ExportTraceServiceRequest_disconnected(t *testing.T) {
	addr := testutils.GetAvailableLocalAddress(t)

	exp, err := NewExporter(WithAddress(addr), WithInsecure())
	if err != nil {
		t.Fatalf("cannot create exporter: %v", err)
	}

	// nil and empty batch should not cause error, since no attempt should be
	// performed.
	err = exp.ExportTraceServiceRequest(nil)
	if err != nil {
		t.Fatalf("nil request should not generate error %v", err)
	}
	err = exp.ExportTraceServiceRequest(&agenttracepb.ExportTraceServiceRequest{})
	if err != nil {
		t.Fatalf("empty request should not generate error %v", err)
	}

	extd := &agenttracepb.ExportTraceServiceRequest{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "test-svc"},
		},
		Spans: []*tracepb.Span{
			{Name: &tracepb.TruncatableString{Value: "00"}},
			{Name: &tracepb.TruncatableString{Value: "01"}},
		},
	}
	err = exp.ExportTraceServiceRequest(extd)
	if err == nil {
		t.Fatalf("failed to send data: %v", err)
	}
}
