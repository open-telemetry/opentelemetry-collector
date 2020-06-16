// Copyright The OpenTelemetry Authors
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
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer/pdata"
	otlptracecol "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
	otlplogs "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/observability"
	"go.opentelemetry.io/collector/testutils"
)

type mockTraceReceiver struct {
	srv            *grpc.Server
	requestCount   int32
	totalSpanCount int32
	lastRequest    *otlptracecol.ExportTraceServiceRequest
	metadata       metadata.MD
}

func (r *mockTraceReceiver) Export(
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
	r.metadata, _ = metadata.FromIncomingContext(ctx)
	return &otlptracecol.ExportTraceServiceResponse{}, nil
}

func otlpTraceReceiverOnGRPCServer(ln net.Listener) *mockTraceReceiver {
	rcv := &mockTraceReceiver{}

	// Now run it as a gRPC server
	rcv.srv = observability.GRPCServerWithObservabilityEnabled()
	otlptracecol.RegisterTraceServiceServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
}

type mockLogsReceiver struct {
	srv                 *grpc.Server
	requestCount        int32
	totalLogRecordCount int32
	lastRequest         *otlplogs.ExportLogServiceRequest
}

func (r *mockLogsReceiver) Export(ctx context.Context, req *otlplogs.ExportLogServiceRequest) (*otlplogs.ExportLogServiceResponse, error) {
	atomic.AddInt32(&r.requestCount, 1)
	recordCount := 0
	for _, rs := range req.ResourceLogs {
		recordCount += len(rs.Logs)
	}
	atomic.AddInt32(&r.totalLogRecordCount, int32(recordCount))
	r.lastRequest = req
	return &otlplogs.ExportLogServiceResponse{}, nil
}

func otlpLogsReceiverOnGRPCServer(ln net.Listener) *mockLogsReceiver {
	rcv := &mockLogsReceiver{}

	// Now run it as a gRPC server
	rcv.srv = observability.GRPCServerWithObservabilityEnabled()
	otlplogs.RegisterLogServiceServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
}

func TestSendTraceData(t *testing.T) {
	// Start an OTLP-compatible receiver.
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv := otlpTraceReceiverOnGRPCServer(ln)
	// Also closes the connection.
	defer rcv.srv.GracefulStop()

	// Start an OTLP exporter and point to the receiver.
	config := Config{
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: ln.Addr().String(),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
			Headers: map[string]string{
				"header": "header-value",
			},
		},
	}

	factory := &Factory{}
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateTraceExporter(context.Background(), creationParams, &config)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	assert.NoError(t, exp.Start(context.Background(), host))

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, rcv.requestCount)

	// Send empty trace.
	td := testdata.GenerateTraceDataEmpty()
	assert.NoError(t, exp.ConsumeTraces(context.Background(), td))

	// Wait until it is received.
	testutils.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, "receive a request")

	// Ensure it was received empty.
	assert.EqualValues(t, 0, rcv.totalSpanCount)

	// A trace with 2 spans.
	td = testdata.GenerateTraceDataTwoSpansSameResource()

	expectedOTLPReq := &otlptracecol.ExportTraceServiceRequest{
		ResourceSpans: testdata.GenerateTraceOtlpSameResourceTwoSpans(),
	}

	err = exp.ConsumeTraces(context.Background(), td)
	assert.NoError(t, err)

	// Wait until it is received.
	testutils.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 1
	}, "receive a request")

	expectedHeader := []string{"header-value"}

	// Verify received span.
	assert.EqualValues(t, 2, rcv.totalSpanCount)
	assert.EqualValues(t, expectedOTLPReq, rcv.lastRequest)

	require.EqualValues(t, rcv.metadata.Get("header"), expectedHeader)
}

func TestSendTraceDataServerDownAndUp(t *testing.T) {
	// Find the addr, but don't start the server.
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	// Start an OTLP exporter and point to the receiver.
	config := Config{
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: ln.Addr().String(),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
		},
	}

	factory := &Factory{}
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateTraceExporter(context.Background(), creationParams, &config)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	assert.NoError(t, exp.Start(context.Background(), host))

	// A trace with 2 spans.
	td := testdata.GenerateTraceDataTwoSpansSameResource()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	assert.Error(t, exp.ConsumeTraces(ctx, td))
	assert.EqualValues(t, context.DeadlineExceeded, ctx.Err())
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	assert.Error(t, exp.ConsumeTraces(ctx, td))
	assert.EqualValues(t, context.DeadlineExceeded, ctx.Err())
	cancel()

	startServerAndMakeRequest(t, exp, td, ln)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	assert.Error(t, exp.ConsumeTraces(ctx, td))
	assert.EqualValues(t, context.DeadlineExceeded, ctx.Err())
	cancel()

	// First call to startServerAndMakeRequest closed the connection. There is a race condition here that the
	// port may be reused, if this gets flaky rethink what to do.
	ln, err = net.Listen("tcp", ln.Addr().String())
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	startServerAndMakeRequest(t, exp, td, ln)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	assert.Error(t, exp.ConsumeTraces(ctx, td))
	assert.EqualValues(t, context.DeadlineExceeded, ctx.Err())
	cancel()
}

func TestSendTraceDataServerStartWhileRequest(t *testing.T) {
	// Find the addr, but don't start the server.
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	// Start an OTLP exporter and point to the receiver.
	config := Config{
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: ln.Addr().String(),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
		},
	}

	factory := &Factory{}
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateTraceExporter(context.Background(), creationParams, &config)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	assert.NoError(t, exp.Start(context.Background(), host))

	// A trace with 2 spans.
	td := testdata.GenerateTraceDataTwoSpansSameResource()
	done := make(chan bool, 1)
	defer close(done)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	go func() {
		assert.NoError(t, exp.ConsumeTraces(ctx, td))
		done <- true
	}()

	time.Sleep(2 * time.Second)
	rcv := otlpTraceReceiverOnGRPCServer(ln)
	defer rcv.srv.GracefulStop()
	// Wait until one of the conditions below triggers.
	select {
	case <-ctx.Done():
		t.Fail()
	case <-done:
		assert.NoError(t, ctx.Err())
	}
	cancel()
}

func startServerAndMakeRequest(t *testing.T, exp component.TraceExporter, td pdata.Traces, ln net.Listener) {
	rcv := otlpTraceReceiverOnGRPCServer(ln)
	defer rcv.srv.GracefulStop()
	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, rcv.requestCount)

	// Resend the request, this should succeed.
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	assert.NoError(t, exp.ConsumeTraces(ctx, td))
	cancel()

	// Wait until it is received.
	testutils.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, "receive a request")

	expectedOTLPReq := &otlptracecol.ExportTraceServiceRequest{
		ResourceSpans: testdata.GenerateTraceOtlpSameResourceTwoSpans(),
	}

	// Verify received span.
	assert.EqualValues(t, 2, rcv.totalSpanCount)
	assert.EqualValues(t, expectedOTLPReq, rcv.lastRequest)
}

func TestSendLogData(t *testing.T) {
	// Start an OTLP-compatible receiver.
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv := otlpLogsReceiverOnGRPCServer(ln)
	// Also closes the connection.
	defer rcv.srv.GracefulStop()

	// Start an OTLP exporter and point to the receiver.
	config := Config{
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: ln.Addr().String(),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
		},
	}

	factory := &Factory{}
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateLogExporter(context.Background(), creationParams, &config)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	assert.NoError(t, exp.Start(context.Background(), host))

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, rcv.requestCount)

	// Send empty request.
	td := testdata.GenerateLogDataEmpty()
	assert.NoError(t, exp.ConsumeLogs(context.Background(), td))

	// Wait until it is received.
	testutils.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, "receive a request")

	// Ensure it was received empty.
	assert.EqualValues(t, 0, rcv.totalLogRecordCount)

	// A request with 2 log entries.
	td = testdata.GenerateLogDataTwoLogsSameResource()

	expectedOTLPReq := &otlplogs.ExportLogServiceRequest{
		ResourceLogs: testdata.GenerateLogOtlpSameResourceTwoLogs(),
	}

	err = exp.ConsumeLogs(context.Background(), td)
	assert.NoError(t, err)

	// Wait until it is received.
	testutils.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 1
	}, "receive a request")

	// Verify received span.
	assert.EqualValues(t, 2, rcv.totalLogRecordCount)
	assert.EqualValues(t, expectedOTLPReq, rcv.lastRequest)
}
