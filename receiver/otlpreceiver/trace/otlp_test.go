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

package trace

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
	collectortrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/testutil"
)

var _ collectortrace.TraceServiceServer = (*Receiver)(nil)

func TestExport(t *testing.T) {
	// given

	traceSink := new(consumertest.TracesSink)

	port, doneFn := otlpReceiverOnGRPCServer(t, traceSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeTraceServiceClient(port)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer traceClientDoneFn()

	// when

	unixnanos := uint64(12578940000000012345)
	traceID := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}
	spanID := [8]byte{8, 7, 6, 5, 4, 3, 2, 1}
	resourceSpans := []*otlptrace.ResourceSpans{
		{
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{
					Spans: []*otlptrace.Span{
						{
							TraceId:           data.NewTraceID(traceID),
							SpanId:            data.NewSpanID(spanID),
							Name:              "operationB",
							Kind:              otlptrace.Span_SPAN_KIND_SERVER,
							StartTimeUnixNano: unixnanos,
							EndTimeUnixNano:   unixnanos,
							Status:            otlptrace.Status{Message: "status-cancelled", Code: otlptrace.Status_STATUS_CODE_ERROR},
							TraceState:        "a=text,b=123",
						},
					},
				},
			},
		},
	}

	// Keep trace data to compare the test result against it
	// Clone needed because OTLP proto XXX_ fields are altered in the GRPC downstream
	traceData := pdata.TracesFromOtlp(resourceSpans).Clone()

	req := &collectortrace.ExportTraceServiceRequest{
		ResourceSpans: resourceSpans,
	}

	resp, err := traceClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export trace: %v", err)
	require.NotNil(t, resp, "The response is missing")

	// assert

	require.Equal(t, 1, len(traceSink.AllTraces()), "unexpected length: %v", len(traceSink.AllTraces()))

	assert.EqualValues(t, traceData, traceSink.AllTraces()[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	traceSink := new(consumertest.TracesSink)

	port, doneFn := otlpReceiverOnGRPCServer(t, traceSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeTraceServiceClient(port)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer traceClientDoneFn()

	resp, err := traceClient.Export(context.Background(), &collectortrace.ExportTraceServiceRequest{})
	assert.NoError(t, err, "Failed to export trace: %v", err)
	assert.NotNil(t, resp, "The response is missing")
}

func TestExport_ErrorConsumer(t *testing.T) {
	traceSink := new(consumertest.TracesSink)
	traceSink.SetConsumeError(fmt.Errorf("error"))

	port, doneFn := otlpReceiverOnGRPCServer(t, traceSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeTraceServiceClient(port)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer traceClientDoneFn()

	req := &collectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{
			{
				InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
					{
						Spans: []*otlptrace.Span{
							{
								Name: "operationB",
							},
						},
					},
				},
			},
		},
	}

	resp, err := traceClient.Export(context.Background(), req)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = error")
	assert.Nil(t, resp)
}

func makeTraceServiceClient(port int) (collectortrace.TraceServiceClient, func(), error) {
	addr := fmt.Sprintf(":%d", port)
	cc, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	metricsClient := collectortrace.NewTraceServiceClient(cc)

	doneFn := func() { _ = cc.Close() }
	return metricsClient, doneFn, nil
}

func otlpReceiverOnGRPCServer(t *testing.T, tc consumer.TracesConsumer) (int, func()) {
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

	r := New(receiverTagValue, tc)
	require.NoError(t, err)

	// Now run it as a gRPC server
	srv := obsreport.GRPCServerWithObservabilityEnabled()
	collectortrace.RegisterTraceServiceServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return port, done
}

func TestDeprecatedStatusCode(t *testing.T) {
	traceSink := new(consumertest.TracesSink)

	port, doneFn := otlpReceiverOnGRPCServer(t, traceSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeTraceServiceClient(port)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer traceClientDoneFn()

	// See specification for handling status code here:
	// https://github.com/open-telemetry/opentelemetry-proto/blob/59c488bfb8fb6d0458ad6425758b70259ff4a2bd/opentelemetry/proto/trace/v1/trace.proto#L231
	tests := []struct {
		sendCode           otlptrace.Status_StatusCode
		sendDeprecatedCode otlptrace.Status_DeprecatedStatusCode
		expectedRcvCode    otlptrace.Status_StatusCode
	}{
		{
			// If code==STATUS_CODE_UNSET then the value of `deprecated_code` is the
			//   carrier of the overall status according to these rules:
			//
			//     if deprecated_code==DEPRECATED_STATUS_CODE_OK then the receiver MUST interpret
			//     the overall status to be STATUS_CODE_UNSET.
			sendCode:           otlptrace.Status_STATUS_CODE_UNSET,
			sendDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_OK,
			expectedRcvCode:    otlptrace.Status_STATUS_CODE_UNSET,
		},
		{
			//     if deprecated_code!=DEPRECATED_STATUS_CODE_OK then the receiver MUST interpret
			//     the overall status to be STATUS_CODE_ERROR.
			sendCode:           otlptrace.Status_STATUS_CODE_UNSET,
			sendDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR,
			expectedRcvCode:    otlptrace.Status_STATUS_CODE_ERROR,
		},
		{
			//   If code!=STATUS_CODE_UNSET then the value of `deprecated_code` MUST be
			//   ignored, the `code` field is the sole carrier of the status.
			sendCode:           otlptrace.Status_STATUS_CODE_OK,
			sendDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_OK,
			expectedRcvCode:    otlptrace.Status_STATUS_CODE_OK,
		},
		{
			//   If code!=STATUS_CODE_UNSET then the value of `deprecated_code` MUST be
			//   ignored, the `code` field is the sole carrier of the status.
			sendCode:           otlptrace.Status_STATUS_CODE_OK,
			sendDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR,
			expectedRcvCode:    otlptrace.Status_STATUS_CODE_OK,
		},
		{
			//   If code!=STATUS_CODE_UNSET then the value of `deprecated_code` MUST be
			//   ignored, the `code` field is the sole carrier of the status.
			sendCode:           otlptrace.Status_STATUS_CODE_ERROR,
			sendDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_OK,
			expectedRcvCode:    otlptrace.Status_STATUS_CODE_ERROR,
		},
		{
			//   If code!=STATUS_CODE_UNSET then the value of `deprecated_code` MUST be
			//   ignored, the `code` field is the sole carrier of the status.
			sendCode:           otlptrace.Status_STATUS_CODE_ERROR,
			sendDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR,
			expectedRcvCode:    otlptrace.Status_STATUS_CODE_ERROR,
		},
	}

	for _, test := range tests {
		resourceSpans := []*otlptrace.ResourceSpans{
			{
				InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
					{
						Spans: []*otlptrace.Span{
							{
								Status: otlptrace.Status{
									Code:           test.sendCode,
									DeprecatedCode: test.sendDeprecatedCode,
								},
							},
						},
					},
				},
			},
		}

		req := &collectortrace.ExportTraceServiceRequest{
			ResourceSpans: resourceSpans,
		}

		traceSink.Reset()

		resp, err := traceClient.Export(context.Background(), req)
		require.NoError(t, err, "Failed to export trace: %v", err)
		require.NotNil(t, resp, "The response is missing")

		require.Equal(t, 1, len(traceSink.AllTraces()), "unexpected length: %v", len(traceSink.AllTraces()))

		rcvdStatus := traceSink.AllTraces()[0].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).Status()

		// Check that Code is as expected.
		assert.EqualValues(t, rcvdStatus.Code(), test.expectedRcvCode)

		// Check that DeprecatedCode is passed as is.
		assert.EqualValues(t, rcvdStatus.DeprecatedCode(), test.sendDeprecatedCode)
	}
}
