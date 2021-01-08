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

package otlpexporter

import (
	"context"
	"net"
	"sync"
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
	otlplogs "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlptraces "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/testutil"
)

type mockReceiver struct {
	srv          *grpc.Server
	requestCount int32
	totalItems   int32
	mux          sync.Mutex
	metadata     metadata.MD
}

func (r *mockReceiver) GetMetadata() metadata.MD {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.metadata
}

type mockTraceReceiver struct {
	mockReceiver
	lastRequest *otlptraces.ExportTraceServiceRequest
}

func (r *mockTraceReceiver) Export(
	ctx context.Context,
	req *otlptraces.ExportTraceServiceRequest,
) (*otlptraces.ExportTraceServiceResponse, error) {
	atomic.AddInt32(&r.requestCount, 1)
	spanCount := 0
	for _, rs := range req.ResourceSpans {
		for _, ils := range rs.InstrumentationLibrarySpans {
			spanCount += len(ils.Spans)
		}
	}
	atomic.AddInt32(&r.totalItems, int32(spanCount))
	r.mux.Lock()
	defer r.mux.Unlock()
	r.lastRequest = req
	r.metadata, _ = metadata.FromIncomingContext(ctx)
	return &otlptraces.ExportTraceServiceResponse{}, nil
}

func (r *mockTraceReceiver) GetLastRequest() *otlptraces.ExportTraceServiceRequest {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.lastRequest
}

func otlpTraceReceiverOnGRPCServer(ln net.Listener) *mockTraceReceiver {
	rcv := &mockTraceReceiver{
		mockReceiver: mockReceiver{
			srv: obsreport.GRPCServerWithObservabilityEnabled(),
		},
	}

	// Now run it as a gRPC server
	otlptraces.RegisterTraceServiceServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
}

type mockLogsReceiver struct {
	mockReceiver
	lastRequest *otlplogs.ExportLogsServiceRequest
}

func (r *mockLogsReceiver) Export(
	ctx context.Context,
	req *otlplogs.ExportLogsServiceRequest,
) (*otlplogs.ExportLogsServiceResponse, error) {
	atomic.AddInt32(&r.requestCount, 1)
	recordCount := 0
	for _, rs := range req.ResourceLogs {
		for _, il := range rs.InstrumentationLibraryLogs {
			recordCount += len(il.Logs)
		}
	}
	atomic.AddInt32(&r.totalItems, int32(recordCount))
	r.mux.Lock()
	defer r.mux.Unlock()
	r.lastRequest = req
	r.metadata, _ = metadata.FromIncomingContext(ctx)
	return &otlplogs.ExportLogsServiceResponse{}, nil
}

func (r *mockLogsReceiver) GetLastRequest() *otlplogs.ExportLogsServiceRequest {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.lastRequest
}

func otlpLogsReceiverOnGRPCServer(ln net.Listener) *mockLogsReceiver {
	rcv := &mockLogsReceiver{
		mockReceiver: mockReceiver{
			srv: obsreport.GRPCServerWithObservabilityEnabled(),
		},
	}

	// Now run it as a gRPC server
	otlplogs.RegisterLogsServiceServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
}

type mockMetricsReceiver struct {
	mockReceiver
	lastRequest *otlpmetrics.ExportMetricsServiceRequest
}

func (r *mockMetricsReceiver) Export(
	ctx context.Context,
	req *otlpmetrics.ExportMetricsServiceRequest,
) (*otlpmetrics.ExportMetricsServiceResponse, error) {
	atomic.AddInt32(&r.requestCount, 1)
	_, recordCount := pdata.MetricsFromOtlp(req.ResourceMetrics).MetricAndDataPointCount()
	atomic.AddInt32(&r.totalItems, int32(recordCount))
	r.mux.Lock()
	defer r.mux.Unlock()
	r.lastRequest = req
	r.metadata, _ = metadata.FromIncomingContext(ctx)
	return &otlpmetrics.ExportMetricsServiceResponse{}, nil
}

func (r *mockMetricsReceiver) GetLastRequest() *otlpmetrics.ExportMetricsServiceRequest {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.lastRequest
}

func otlpMetricsReceiverOnGRPCServer(ln net.Listener) *mockMetricsReceiver {
	rcv := &mockMetricsReceiver{
		mockReceiver: mockReceiver{
			srv: obsreport.GRPCServerWithObservabilityEnabled(),
		},
	}

	// Now run it as a gRPC server
	otlpmetrics.RegisterMetricsServiceServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
}

func TestSendTraces(t *testing.T) {
	// Start an OTLP-compatible receiver.
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv := otlpTraceReceiverOnGRPCServer(ln)
	// Also closes the connection.
	defer rcv.srv.GracefulStop()

	// Start an OTLP exporter and point to the receiver.
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: ln.Addr().String(),
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
		Headers: map[string]string{
			"header": "header-value",
		},
	}
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateTracesExporter(context.Background(), creationParams, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	assert.NoError(t, exp.Start(context.Background(), host))

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.requestCount))

	// Send empty trace.
	td := testdata.GenerateTraceDataEmpty()
	assert.NoError(t, exp.ConsumeTraces(context.Background(), td))

	// Wait until it is received.
	testutil.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, "receive a request")

	// Ensure it was received empty.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.totalItems))

	// A trace with 2 spans.
	td = testdata.GenerateTraceDataTwoSpansSameResource()

	expectedOTLPReq := &otlptraces.ExportTraceServiceRequest{
		ResourceSpans: testdata.GenerateTraceOtlpSameResourceTwoSpans(),
	}

	err = exp.ConsumeTraces(context.Background(), td)
	assert.NoError(t, err)

	// Wait until it is received.
	testutil.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 1
	}, "receive a request")

	expectedHeader := []string{"header-value"}

	// Verify received span.
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.totalItems))
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.requestCount))
	assert.EqualValues(t, expectedOTLPReq, rcv.GetLastRequest())

	require.EqualValues(t, rcv.GetMetadata().Get("header"), expectedHeader)
}

func TestSendMetrics(t *testing.T) {
	// Start an OTLP-compatible receiver.
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv := otlpMetricsReceiverOnGRPCServer(ln)
	// Also closes the connection.
	defer rcv.srv.GracefulStop()

	// Start an OTLP exporter and point to the receiver.
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: ln.Addr().String(),
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
		Headers: map[string]string{
			"header": "header-value",
		},
	}
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateMetricsExporter(context.Background(), creationParams, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	assert.NoError(t, exp.Start(context.Background(), host))

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.requestCount))

	// Send empty trace.
	md := testdata.GenerateMetricsEmpty()
	assert.NoError(t, exp.ConsumeMetrics(context.Background(), md))

	// Wait until it is received.
	testutil.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, "receive a request")

	// Ensure it was received empty.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.totalItems))

	// A trace with 2 spans.
	md = testdata.GenerateMetricsTwoMetrics()

	expectedOTLPReq := &otlpmetrics.ExportMetricsServiceRequest{
		ResourceMetrics: testdata.GenerateMetricsOtlpTwoMetrics(),
	}

	err = exp.ConsumeMetrics(context.Background(), md)
	assert.NoError(t, err)

	// Wait until it is received.
	testutil.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 1
	}, "receive a request")

	expectedHeader := []string{"header-value"}

	// Verify received metrics.
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.requestCount))
	assert.EqualValues(t, 4, atomic.LoadInt32(&rcv.totalItems))
	assert.EqualValues(t, expectedOTLPReq, rcv.GetLastRequest())

	require.EqualValues(t, rcv.GetMetadata().Get("header"), expectedHeader)
}

func TestSendTraceDataServerDownAndUp(t *testing.T) {
	// Find the addr, but don't start the server.
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	// Start an OTLP exporter and point to the receiver.
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	// Disable queuing to ensure that we execute the request when calling ConsumeTraces
	// otherwise we will not see the error.
	cfg.QueueSettings.Enabled = false
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: ln.Addr().String(),
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
		// Need to wait for every request blocking until either request timeouts or succeed.
		// Do not rely on external retry logic here, if that is intended set InitialInterval to 100ms.
		WaitForReady: true,
	}
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateTracesExporter(context.Background(), creationParams, cfg)
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
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: ln.Addr().String(),
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateTracesExporter(context.Background(), creationParams, cfg)
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

func startServerAndMakeRequest(t *testing.T, exp component.TracesExporter, td pdata.Traces, ln net.Listener) {
	rcv := otlpTraceReceiverOnGRPCServer(ln)
	defer rcv.srv.GracefulStop()
	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.requestCount))

	// Resend the request, this should succeed.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	assert.NoError(t, exp.ConsumeTraces(ctx, td))
	cancel()

	// Wait until it is received.
	testutil.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, "receive a request")

	expectedOTLPReq := &otlptraces.ExportTraceServiceRequest{
		ResourceSpans: testdata.GenerateTraceOtlpSameResourceTwoSpans(),
	}

	// Verify received span.
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.totalItems))
	assert.EqualValues(t, expectedOTLPReq, rcv.GetLastRequest())
}

func TestSendLogData(t *testing.T) {
	// Start an OTLP-compatible receiver.
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv := otlpLogsReceiverOnGRPCServer(ln)
	// Also closes the connection.
	defer rcv.srv.GracefulStop()

	// Start an OTLP exporter and point to the receiver.
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: ln.Addr().String(),
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateLogsExporter(context.Background(), creationParams, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	assert.NoError(t, exp.Start(context.Background(), host))

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.requestCount))

	// Send empty request.
	td := testdata.GenerateLogDataEmpty()
	assert.NoError(t, exp.ConsumeLogs(context.Background(), td))

	// Wait until it is received.
	testutil.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, "receive a request")

	// Ensure it was received empty.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.totalItems))

	// A request with 2 log entries.
	td = testdata.GenerateLogDataTwoLogsSameResource()

	expectedOTLPReq := &otlplogs.ExportLogsServiceRequest{
		ResourceLogs: testdata.GenerateLogOtlpSameResourceTwoLogs(),
	}

	err = exp.ConsumeLogs(context.Background(), td)
	assert.NoError(t, err)

	// Wait until it is received.
	testutil.WaitFor(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 1
	}, "receive a request")

	// Verify received logs.
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.requestCount))
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.totalItems))
	assert.EqualValues(t, expectedOTLPReq, rcv.GetLastRequest())
}
