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

package ocinterceptor_test

import (
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/grpc"

	"contrib.go.opencensus.io/exporter/ocagent"
	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/interceptor/opencensus"
	"github.com/census-instrumentation/opencensus-service/spanreceiver"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/tracestate"
)

func TestOCInterceptor_endToEnd(t *testing.T) {
	sappender := newSpanAppender()

	_, port, doneFn := ocInterceptorOnGRPCServer(t, sappender, ocinterceptor.WithSpanBufferPeriod(100*time.Millisecond))
	defer doneFn()

	// Now the opencensus-agent exporter.
	oce, err := ocagent.NewExporter(ocagent.WithPort(uint16(port)), ocagent.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to create the ocagent-exporter: %v", err)
	}

	trace.RegisterExporter(oce)

	defer func() {
		oce.Stop()
		trace.UnregisterExporter(oce)
	}()

	now := time.Now().UTC()
	clientSpanData := &trace.SpanData{
		StartTime: now.Add(-10 * time.Second),
		EndTime:   now.Add(20 * time.Second),
		SpanContext: trace.SpanContext{
			TraceID:      trace.TraceID{0x4F, 0x4E, 0x4D, 0x4C, 0x4B, 0x4A, 0x49, 0x48, 0x47, 0x46, 0x45, 0x44, 0x43, 0x42, 0x41},
			SpanID:       trace.SpanID{0x7F, 0x7E, 0x7D, 0x7C, 0x7B, 0x7A, 0x79, 0x78},
			TraceOptions: trace.TraceOptions(0x01),
		},
		ParentSpanID: trace.SpanID{0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37},
		Name:         "ClientSpan",
		Status:       trace.Status{Code: trace.StatusCodeInternal, Message: "Blocked by firewall"},
		SpanKind:     trace.SpanKindClient,
	}

	serverSpanData := &trace.SpanData{
		StartTime: now.Add(-5 * time.Second),
		EndTime:   now.Add(10 * time.Second),
		SpanContext: trace.SpanContext{
			TraceID:      trace.TraceID{0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E},
			SpanID:       trace.SpanID{0xF0, 0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7},
			TraceOptions: trace.TraceOptions(0x01),
			Tracestate:   &tracestate.Tracestate{},
		},
		ParentSpanID: trace.SpanID{0x38, 0x39, 0x3A, 0x3B, 0x3C, 0x3D, 0x3E, 0x3F},
		Name:         "ServerSpan",
		Status:       trace.Status{Code: trace.StatusCodeOK, Message: "OK"},
		SpanKind:     trace.SpanKindServer,
		Links: []trace.Link{
			{
				TraceID: trace.TraceID{0x4F, 0x4E, 0x4D, 0x4C, 0x4B, 0x4A, 0x49, 0x48, 0x47, 0x46, 0x45, 0x44, 0x43, 0x42, 0x41, 0x40},
				SpanID:  trace.SpanID{0x7F, 0x7E, 0x7D, 0x7C, 0x7B, 0x7A, 0x79, 0x78},
				Type:    trace.LinkTypeParent,
			},
		},
	}

	oce.ExportSpan(serverSpanData)
	oce.ExportSpan(clientSpanData)
	// Give them some time to be exported.
	<-time.After(100 * time.Millisecond)

	oce.Flush()

	// Give them some time to be exported.
	<-time.After(150 * time.Millisecond)

	// Now span inspection and verification time!
	var gotSpans []*tracepb.Span
	sappender.forEachEntry(func(_ *commonpb.Node, spans []*tracepb.Span) {
		gotSpans = append(gotSpans, spans...)
	})

	wantSpans := []*tracepb.Span{
		{
			TraceId:      serverSpanData.TraceID[:],
			SpanId:       serverSpanData.SpanID[:],
			ParentSpanId: serverSpanData.ParentSpanID[:],
			Name:         &tracepb.TruncatableString{Value: "ServerSpan"},
			Kind:         tracepb.Span_SERVER,
			StartTime:    timeToTimestamp(serverSpanData.StartTime),
			EndTime:      timeToTimestamp(serverSpanData.EndTime),
			Status:       &tracepb.Status{Code: int32(serverSpanData.Status.Code), Message: serverSpanData.Status.Message},
			Tracestate:   &tracepb.Span_Tracestate{},
			Links: &tracepb.Span_Links{
				Link: []*tracepb.Span_Link{
					{
						TraceId: []byte{0x4F, 0x4E, 0x4D, 0x4C, 0x4B, 0x4A, 0x49, 0x48, 0x47, 0x46, 0x45, 0x44, 0x43, 0x42, 0x41, 0x40},
						SpanId:  []byte{0x7F, 0x7E, 0x7D, 0x7C, 0x7B, 0x7A, 0x79, 0x78},
						Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
					},
				},
			},
		},
		{
			TraceId:      clientSpanData.TraceID[:],
			SpanId:       clientSpanData.SpanID[:],
			ParentSpanId: clientSpanData.ParentSpanID[:],
			Name:         &tracepb.TruncatableString{Value: "ClientSpan"},
			Kind:         tracepb.Span_CLIENT,
			StartTime:    timeToTimestamp(clientSpanData.StartTime),
			EndTime:      timeToTimestamp(clientSpanData.EndTime),
			Status:       &tracepb.Status{Code: int32(clientSpanData.Status.Code), Message: clientSpanData.Status.Message},
		},
	}

	if g, w := len(gotSpans), len(wantSpans); g != w {
		t.Errorf("SpanCount: got %d want %d", g, w)
	}

	if !reflect.DeepEqual(gotSpans, wantSpans) {
		gotBlob, _ := json.MarshalIndent(gotSpans, "", "  ")
		wantBlob, _ := json.MarshalIndent(wantSpans, "", "  ")
		t.Errorf("GotSpans:\n%s\nWantSpans:\n%s", gotBlob, wantBlob)
	}
}

// Helper functions from here on down.
type spanAppender struct {
	sync.RWMutex
	spansPerNode map[*commonpb.Node][]*tracepb.Span
}

func newSpanAppender() *spanAppender {
	return &spanAppender{spansPerNode: make(map[*commonpb.Node][]*tracepb.Span)}
}

var _ spanreceiver.SpanReceiver = (*spanAppender)(nil)

func (sa *spanAppender) ReceiveSpans(node *commonpb.Node, spans ...*tracepb.Span) (*spanreceiver.Acknowledgement, error) {
	sa.Lock()
	defer sa.Unlock()

	sa.spansPerNode[node] = append(sa.spansPerNode[node], spans...)

	return &spanreceiver.Acknowledgement{SavedSpans: uint64(len(spans))}, nil
}

func ocInterceptorOnGRPCServer(t *testing.T, sr spanreceiver.SpanReceiver, opts ...ocinterceptor.OCOption) (oci *ocinterceptor.OCInterceptor, port int, done func()) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to find an available address to run the gRPC server: %v", err)
	}

	doneFnList := []func(){func() { ln.Close() }}
	done = func() {
		for _, doneFn := range doneFnList {
			doneFn()
		}
	}

	_, port, err = hostPortFromAddr(ln.Addr())
	if err != nil {
		done()
		t.Fatalf("Failed to parse host:port from listener address: %s error: %v", ln.Addr(), err)
	}

	if err != nil {
		done()
		t.Fatalf("Failed to create new agent: %v", err)
	}

	oci, err = ocinterceptor.New(sr, opts...)
	if err != nil {
		t.Fatalf("Failed to create the OCInterceptor: %v", err)
	}

	// Now run it as a gRPC server
	srv := grpc.NewServer()
	agenttracepb.RegisterTraceServiceServer(srv, oci)
	go func() {
		_ = srv.Serve(ln)
	}()

	return oci, port, done
}

func hostPortFromAddr(addr net.Addr) (host string, port int, err error) {
	addrStr := addr.String()
	sepIndex := strings.LastIndex(addrStr, ":")
	if sepIndex < 0 {
		return "", -1, errors.New("failed to parse host:port")
	}
	host, portStr := addrStr[:sepIndex], addrStr[sepIndex+1:]
	port, err = strconv.Atoi(portStr)
	return host, port, err
}

func (sa *spanAppender) forEachEntry(fn func(*commonpb.Node, []*tracepb.Span)) {
	sa.RLock()
	defer sa.RUnlock()

	for node, spans := range sa.spansPerNode {
		fn(node, spans)
	}
}

func timeToTimestamp(t time.Time) *timestamp.Timestamp {
	nanoTime := t.UnixNano()
	return &timestamp.Timestamp{
		Seconds: nanoTime / 1e9,
		Nanos:   int32(nanoTime % 1e9),
	}
}
