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

package otlpexporter

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	otlptracecol "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/trace/v1"
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configgrpc"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
)

type MockReceiver struct {
	requestCount   int32
	totalSpanCount int32
	lastRequest    *otlptracecol.ExportTraceServiceRequest
}

func (r *MockReceiver) Export(
	ctx context.Context,
	req *otlptracecol.ExportTraceServiceRequest,
) (*otlptracecol.ExportTraceServiceResponse, error) {
	atomic.AddInt32(&r.requestCount, 1)
	spanCount := 0
	for _, rs := range req.ResourceSpans {
		for _, ils := range rs.InstrumentationLibrarySpans {
			spanCount += len(ils.Spans)
		}
	}
	atomic.AddInt32(&r.totalSpanCount, int32(spanCount))
	r.lastRequest = req
	return &otlptracecol.ExportTraceServiceResponse{}, nil
}

func otlpReceiverOnGRPCServer(t *testing.T) (r *MockReceiver, port int, done func()) {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

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

	r = &MockReceiver{}
	require.NoError(t, err, "Failed to create the Receiver: %v", err)

	// Now run it as a gRPC server
	srv := observability.GRPCServerWithObservabilityEnabled()
	otlptracecol.RegisterTraceServiceServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return r, port, done
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

func TestSendData(t *testing.T) {
	// Start an OTLP-compatible receiver.
	rcv, port, done := otlpReceiverOnGRPCServer(t)
	defer done()

	// Start an OTLP exporter and point to the receiver.

	endpoint := fmt.Sprintf("localhost:%d", port)

	config := Config{
		GRPCSettings: configgrpc.GRPCSettings{
			Endpoint: endpoint,
		},
	}

	factory := &Factory{}
	exp, err := factory.CreateTraceExporter(zap.NewNop(), &config)
	assert.Nil(t, err)
	require.NotNil(t, exp)
	defer exp.Shutdown()

	host := component.NewMockHost()

	err = exp.Start(host)
	assert.NoError(t, err)

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, rcv.requestCount)

	// Send empty trace.
	td := consumerdata.TraceData{}
	exp.ConsumeTraceData(context.Background(), td)

	// Wait until it is received.
	testutils.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, "receive a request")

	// Ensure it was received empty.
	assert.EqualValues(t, 0, rcv.totalSpanCount)

	// Prepare non-empty trace data.
	unixnanos := uint64(12578940000000012345)

	traceID, err := base64.StdEncoding.DecodeString("SEhaOVO7YSQ=")
	assert.NoError(t, err)

	spanID, err := base64.StdEncoding.DecodeString("QuHicGYRg4U=")
	assert.NoError(t, err)

	// A trace with 1 span.
	td = consumerdata.TraceData{
		Node: &occommon.Node{},
		Resource: &ocresource.Resource{
			Labels: map[string]string{
				"key1": "value1",
			},
		},
		Spans: []*octrace.Span{
			{
				TraceId:   traceID,
				SpanId:    spanID,
				Name:      &octrace.TruncatableString{Value: "operationB"},
				Kind:      octrace.Span_SERVER,
				StartTime: internal.UnixNanoToTimestamp(data.TimestampUnixNano(unixnanos)),
				EndTime:   internal.UnixNanoToTimestamp(data.TimestampUnixNano(unixnanos)),
				TimeEvents: &octrace.Span_TimeEvents{
					TimeEvent: []*octrace.Span_TimeEvent{
						{
							Time: internal.UnixNanoToTimestamp(data.TimestampUnixNano(unixnanos)),
							Value: &octrace.Span_TimeEvent_Annotation_{
								Annotation: &octrace.Span_TimeEvent_Annotation{
									Description: &octrace.TruncatableString{Value: "event1"},
									Attributes: &octrace.Span_Attributes{
										AttributeMap: map[string]*octrace.AttributeValue{
											"eventattr1": {
												Value: &octrace.AttributeValue_StringValue{
													StringValue: &octrace.TruncatableString{Value: "eventattrval1"},
												},
											},
										},
										DroppedAttributesCount: 4,
									},
								},
							},
						},
					},
					DroppedMessageEventsCount: 2,
				},
				Links: &octrace.Span_Links{
					Link: []*octrace.Span_Link{
						{
							TraceId: traceID,
							SpanId:  spanID,
						},
					},
				},
				Attributes: &octrace.Span_Attributes{
					DroppedAttributesCount: 1,
				},
				Status: &octrace.Status{Message: "status-cancelled", Code: 1},
				Tracestate: &octrace.Span_Tracestate{
					Entries: []*octrace.Span_Tracestate_Entry{
						{
							Key:   "a",
							Value: "text",
						},
						{
							Key:   "b",
							Value: "123",
						},
					},
				},
			},
		},
		SourceFormat: "otlp_trace",
	}

	expectedOTLPReq := &otlptracecol.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{
			{
				Resource: &otlpresource.Resource{
					Attributes: []*otlpcommon.AttributeKeyValue{
						{
							Key:         "key1",
							StringValue: "value1",
						},
					},
				},
				InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
					{
						Spans: []*otlptrace.Span{
							{
								TraceId:           traceID,
								SpanId:            spanID,
								Name:              "operationB",
								Kind:              otlptrace.Span_SERVER,
								StartTimeUnixNano: unixnanos,
								EndTimeUnixNano:   unixnanos,
								Events: []*otlptrace.Span_Event{
									{
										TimeUnixNano: unixnanos,
										Name:         "event1",
										Attributes: []*otlpcommon.AttributeKeyValue{
											{
												Key:         "eventattr1",
												Type:        otlpcommon.AttributeKeyValue_STRING,
												StringValue: "eventattrval1",
											},
										},
										DroppedAttributesCount: 4,
									},
								},
								Links: []*otlptrace.Span_Link{
									{
										TraceId: traceID,
										SpanId:  spanID,
									},
								},
								DroppedAttributesCount: 1,
								DroppedEventsCount:     2,
								Status:                 &otlptrace.Status{Message: "status-cancelled", Code: otlptrace.Status_Cancelled},
								TraceState:             "a=text,b=123",
							},
						},
					},
				},
			},
		},
	}

	err = exp.ConsumeTraceData(context.Background(), td)
	assert.NoError(t, err)

	// Wait until it is received.
	testutils.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 1
	}, "receive a request")

	// Verify received span.
	assert.EqualValues(t, 1, rcv.totalSpanCount)
	assert.EqualValues(t, expectedOTLPReq, rcv.lastRequest)
}
