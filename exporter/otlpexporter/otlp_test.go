// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter

import (
	"context"
	"net"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/pdata/testdata"
)

type mockReceiver struct {
	srv          *grpc.Server
	requestCount *atomic.Int64
	totalItems   *atomic.Int64
	mux          sync.Mutex
	metadata     metadata.MD
	exportError  error
}

func (r *mockReceiver) getMetadata() metadata.MD {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.metadata
}

func (r *mockReceiver) setExportError(err error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.exportError = err
}

var _ ptraceotlp.GRPCServer = &mockTracesReceiver{}

type mockTracesReceiver struct {
	ptraceotlp.UnimplementedGRPCServer
	mockReceiver
	exportResponse func() ptraceotlp.ExportResponse
	lastRequest    ptrace.Traces
}

func (r *mockTracesReceiver) Export(ctx context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	r.requestCount.Add(1)
	td := req.Traces()
	r.totalItems.Add(int64(td.SpanCount()))
	r.mux.Lock()
	defer r.mux.Unlock()
	r.lastRequest = td
	r.metadata, _ = metadata.FromIncomingContext(ctx)
	return r.exportResponse(), r.exportError
}

func (r *mockTracesReceiver) getLastRequest() ptrace.Traces {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.lastRequest
}

func (r *mockTracesReceiver) setExportResponse(fn func() ptraceotlp.ExportResponse) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.exportResponse = fn
}

func otlpTracesReceiverOnGRPCServer(ln net.Listener, useTLS bool) (*mockTracesReceiver, error) {
	sopts := []grpc.ServerOption{}

	if useTLS {
		_, currentFile, _, _ := runtime.Caller(0)
		basepath := filepath.Dir(currentFile)
		certpath := filepath.Join(basepath, filepath.Join("testdata", "test_cert.pem"))
		keypath := filepath.Join(basepath, filepath.Join("testdata", "test_key.pem"))

		creds, err := credentials.NewServerTLSFromFile(certpath, keypath)
		if err != nil {
			return nil, err
		}
		sopts = append(sopts, grpc.Creds(creds))
	}

	rcv := &mockTracesReceiver{
		mockReceiver: mockReceiver{
			srv:          grpc.NewServer(sopts...),
			requestCount: new(atomic.Int64),
			totalItems:   new(atomic.Int64),
		},
		exportResponse: ptraceotlp.NewExportResponse,
	}

	// Now run it as a gRPC server
	ptraceotlp.RegisterGRPCServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv, nil
}

var _ plogotlp.GRPCServer = &mockLogsReceiver{}

type mockLogsReceiver struct {
	plogotlp.UnimplementedGRPCServer
	mockReceiver
	exportResponse func() plogotlp.ExportResponse
	lastRequest    plog.Logs
}

func (r *mockLogsReceiver) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	r.requestCount.Add(1)
	ld := req.Logs()
	r.totalItems.Add(int64(ld.LogRecordCount()))
	r.mux.Lock()
	defer r.mux.Unlock()
	r.lastRequest = ld
	r.metadata, _ = metadata.FromIncomingContext(ctx)
	return r.exportResponse(), r.exportError
}

func (r *mockLogsReceiver) getLastRequest() plog.Logs {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.lastRequest
}

func (r *mockLogsReceiver) setExportResponse(fn func() plogotlp.ExportResponse) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.exportResponse = fn
}

func otlpLogsReceiverOnGRPCServer(ln net.Listener) *mockLogsReceiver {
	rcv := &mockLogsReceiver{
		mockReceiver: mockReceiver{
			srv:          grpc.NewServer(),
			requestCount: new(atomic.Int64),
			totalItems:   new(atomic.Int64),
		},
		exportResponse: plogotlp.NewExportResponse,
	}

	// Now run it as a gRPC server
	plogotlp.RegisterGRPCServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
}

var _ pmetricotlp.GRPCServer = &mockMetricsReceiver{}

type mockMetricsReceiver struct {
	pmetricotlp.UnimplementedGRPCServer
	mockReceiver
	exportResponse func() pmetricotlp.ExportResponse
	lastRequest    pmetric.Metrics
}

func (r *mockMetricsReceiver) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	md := req.Metrics()
	r.requestCount.Add(1)
	r.totalItems.Add(int64(md.DataPointCount()))
	r.mux.Lock()
	defer r.mux.Unlock()
	r.lastRequest = md
	r.metadata, _ = metadata.FromIncomingContext(ctx)
	return r.exportResponse(), r.exportError
}

func (r *mockMetricsReceiver) getLastRequest() pmetric.Metrics {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.lastRequest
}

func (r *mockMetricsReceiver) setExportResponse(fn func() pmetricotlp.ExportResponse) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.exportResponse = fn
}

func otlpMetricsReceiverOnGRPCServer(ln net.Listener) *mockMetricsReceiver {
	rcv := &mockMetricsReceiver{
		mockReceiver: mockReceiver{
			srv:          grpc.NewServer(),
			requestCount: new(atomic.Int64),
			totalItems:   new(atomic.Int64),
		},
		exportResponse: pmetricotlp.NewExportResponse,
	}

	// Now run it as a gRPC server
	pmetricotlp.RegisterGRPCServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
}

type mockProfilesReceiver struct {
	pprofileotlp.UnimplementedGRPCServer
	mockReceiver
	exportResponse func() pprofileotlp.ExportResponse
	lastRequest    pprofile.Profiles
}

func (r *mockProfilesReceiver) Export(ctx context.Context, req pprofileotlp.ExportRequest) (pprofileotlp.ExportResponse, error) {
	r.requestCount.Add(1)
	td := req.Profiles()
	r.totalItems.Add(int64(td.SampleCount()))
	r.mux.Lock()
	defer r.mux.Unlock()
	r.lastRequest = td
	r.metadata, _ = metadata.FromIncomingContext(ctx)
	return r.exportResponse(), r.exportError
}

func (r *mockProfilesReceiver) getLastRequest() pprofile.Profiles {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.lastRequest
}

func (r *mockProfilesReceiver) setExportResponse(fn func() pprofileotlp.ExportResponse) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.exportResponse = fn
}

func otlpProfilesReceiverOnGRPCServer(ln net.Listener, useTLS bool) (*mockProfilesReceiver, error) {
	sopts := []grpc.ServerOption{}

	if useTLS {
		_, currentFile, _, _ := runtime.Caller(0)
		basepath := filepath.Dir(currentFile)
		certpath := filepath.Join(basepath, filepath.Join("testdata", "test_cert.pem"))
		keypath := filepath.Join(basepath, filepath.Join("testdata", "test_key.pem"))

		creds, err := credentials.NewServerTLSFromFile(certpath, keypath)
		if err != nil {
			return nil, err
		}
		sopts = append(sopts, grpc.Creds(creds))
	}

	rcv := &mockProfilesReceiver{
		mockReceiver: mockReceiver{
			requestCount: &atomic.Int64{},
			totalItems:   &atomic.Int64{},
			srv:          grpc.NewServer(sopts...),
		},
		exportResponse: pprofileotlp.NewExportResponse,
	}

	// Now run it as a gRPC server
	pprofileotlp.RegisterGRPCServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv, nil
}

func TestSendTraces(t *testing.T) {
	// Start an OTLP-compatible receiver.
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv, _ := otlpTracesReceiverOnGRPCServer(ln, false)
	// Also closes the connection.
	defer rcv.srv.GracefulStop()

	// Start an OTLP exporter and point to the receiver.
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	// Disable queuing to ensure that we execute the request when calling ConsumeTraces
	// otherwise we will not see any errors.
	cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	cfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: ln.Addr().String(),
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
		Headers: configopaque.MapList{
			{Name: "header", Value: "header-value"},
		},
	}
	set := exportertest.NewNopSettings(factory.Type())
	set.BuildInfo.Description = "Collector"
	set.BuildInfo.Version = "1.2.3test"

	// For testing the "Partial success" warning.
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)

	exp, err := factory.CreateTraces(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)

	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, rcv.requestCount.Load())

	// Send empty trace.
	td := ptrace.NewTraces()
	require.NoError(t, exp.ConsumeTraces(context.Background(), td))

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return rcv.requestCount.Load() > 0
	}, 10*time.Second, 5*time.Millisecond)

	// Ensure it was received empty.
	assert.EqualValues(t, 0, rcv.totalItems.Load())

	// A trace with 2 spans.
	td = testdata.GenerateTraces(2)

	err = exp.ConsumeTraces(context.Background(), td)
	require.NoError(t, err)

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return rcv.requestCount.Load() > 1
	}, 10*time.Second, 5*time.Millisecond)

	expectedHeader := []string{"header-value"}

	// Verify received span.
	assert.EqualValues(t, 2, rcv.totalItems.Load())
	assert.EqualValues(t, 2, rcv.requestCount.Load())
	assert.Equal(t, td, rcv.getLastRequest())

	md := rcv.getMetadata()
	require.Equal(t, expectedHeader, md.Get("header"))
	require.Len(t, md.Get("User-Agent"), 1)
	require.Contains(t, md.Get("User-Agent")[0], "Collector/1.2.3test")

	// Return partial success
	rcv.setExportResponse(func() ptraceotlp.ExportResponse {
		response := ptraceotlp.NewExportResponse()
		partialSuccess := response.PartialSuccess()
		partialSuccess.SetErrorMessage("Some spans were not ingested")
		partialSuccess.SetRejectedSpans(1)

		return response
	})

	// A request with 2 Trace entries.
	td = testdata.GenerateTraces(2)

	err = exp.ConsumeTraces(context.Background(), td)
	require.NoError(t, err)
	assert.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), 1)
	assert.Contains(t, observed.FilterLevelExact(zap.WarnLevel).All()[0].Message, "Partial success")
}

func TestSendTracesWhenEndpointHasHttpScheme(t *testing.T) {
	tests := []struct {
		name               string
		useTLS             bool
		scheme             string
		gRPCClientSettings configgrpc.ClientConfig
	}{
		{
			name:               "Use https scheme",
			useTLS:             true,
			scheme:             "https://",
			gRPCClientSettings: configgrpc.ClientConfig{},
		},
		{
			name:   "Use http scheme",
			useTLS: false,
			scheme: "http://",
			gRPCClientSettings: configgrpc.ClientConfig{
				TLS: configtls.ClientConfig{
					Insecure: true,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Start an OTLP-compatible receiver.
			ln, err := net.Listen("tcp", "localhost:")
			require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
			rcv, err := otlpTracesReceiverOnGRPCServer(ln, test.useTLS)
			require.NoError(t, err, "Failed to start mock OTLP receiver")
			// Also closes the connection.
			defer rcv.srv.GracefulStop()

			// Start an OTLP exporter and point to the receiver.
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig().(*Config)
			cfg.ClientConfig = test.gRPCClientSettings
			cfg.ClientConfig.Endpoint = test.scheme + ln.Addr().String()
			if test.useTLS {
				cfg.ClientConfig.TLS.InsecureSkipVerify = true
			}
			set := exportertest.NewNopSettings(factory.Type())
			exp, err := factory.CreateTraces(context.Background(), set, cfg)
			require.NoError(t, err)
			require.NotNil(t, exp)

			defer func() {
				assert.NoError(t, exp.Shutdown(context.Background()))
			}()

			host := componenttest.NewNopHost()
			require.NoError(t, exp.Start(context.Background(), host))

			// Ensure that initially there is no data in the receiver.
			assert.EqualValues(t, 0, rcv.requestCount.Load())

			// Send empty trace.
			td := ptrace.NewTraces()
			require.NoError(t, exp.ConsumeTraces(context.Background(), td))

			// Wait until it is received.
			assert.Eventually(t, func() bool {
				return rcv.requestCount.Load() > 0
			}, 10*time.Second, 5*time.Millisecond)

			// Ensure it was received empty.
			assert.EqualValues(t, 0, rcv.totalItems.Load())
		})
	}
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
	// Disable queuing to ensure that we execute the request when calling ConsumeMetrics
	// otherwise we will not see any errors.
	cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	cfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: ln.Addr().String(),
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
		Headers: configopaque.MapList{
			{Name: "header", Value: "header-value"},
		},
	}
	set := exportertest.NewNopSettings(factory.Type())
	set.BuildInfo.Description = "Collector"
	set.BuildInfo.Version = "1.2.3test"

	// For testing the "Partial success" warning.
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)

	exp, err := factory.CreateMetrics(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	require.NoError(t, exp.Start(context.Background(), host))

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, rcv.requestCount.Load())

	// Send empty metric.
	md := pmetric.NewMetrics()
	require.NoError(t, exp.ConsumeMetrics(context.Background(), md))

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return rcv.requestCount.Load() > 0
	}, 10*time.Second, 5*time.Millisecond)

	// Ensure it was received empty.
	assert.EqualValues(t, 0, rcv.totalItems.Load())

	// Send two metrics.
	md = testdata.GenerateMetrics(2)

	err = exp.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return rcv.requestCount.Load() > 1
	}, 10*time.Second, 5*time.Millisecond)

	expectedHeader := []string{"header-value"}

	// Verify received metrics.
	assert.EqualValues(t, 2, rcv.requestCount.Load())
	assert.EqualValues(t, 4, rcv.totalItems.Load())
	assert.Equal(t, md, rcv.getLastRequest())

	mdata := rcv.getMetadata()
	require.Equal(t, expectedHeader, mdata.Get("header"))
	require.Len(t, mdata.Get("User-Agent"), 1)
	require.Contains(t, mdata.Get("User-Agent")[0], "Collector/1.2.3test")

	st := status.New(codes.InvalidArgument, "Invalid argument")
	rcv.setExportError(st.Err())

	// Send two metrics..
	md = testdata.GenerateMetrics(2)

	err = exp.ConsumeMetrics(context.Background(), md)
	require.Error(t, err)

	rcv.setExportError(nil)

	// Return partial success
	rcv.setExportResponse(func() pmetricotlp.ExportResponse {
		response := pmetricotlp.NewExportResponse()
		partialSuccess := response.PartialSuccess()
		partialSuccess.SetErrorMessage("Some data points were not ingested")
		partialSuccess.SetRejectedDataPoints(1)

		return response
	})

	// Send two metrics.
	md = testdata.GenerateMetrics(2)
	require.NoError(t, exp.ConsumeMetrics(context.Background(), md))
	assert.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), 1)
	assert.Contains(t, observed.FilterLevelExact(zap.WarnLevel).All()[0].Message, "Partial success")
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
	cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	cfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: ln.Addr().String(),
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
		// Need to wait for every request blocking until either request timeouts or succeed.
		// Do not rely on external retry logic here, if that is intended set InitialInterval to 100ms.
		WaitForReady: true,
	}
	set := exportertest.NewNopSettings(factory.Type())
	exp, err := factory.CreateTraces(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	require.NoError(t, exp.Start(context.Background(), host))

	// A trace with 2 spans.
	td := testdata.GenerateTraces(2)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	require.Error(t, exp.ConsumeTraces(ctx, td))
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	require.Error(t, exp.ConsumeTraces(ctx, td))
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	cancel()

	startServerAndMakeRequest(t, exp, td, ln)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	require.Error(t, exp.ConsumeTraces(ctx, td))
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	cancel()

	// First call to startServerAndMakeRequest closed the connection. There is a race condition here that the
	// port may be reused, if this gets flaky rethink what to do.
	ln, err = net.Listen("tcp", ln.Addr().String())
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	startServerAndMakeRequest(t, exp, td, ln)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	require.Error(t, exp.ConsumeTraces(ctx, td))
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	cancel()
}

func TestSendTraceDataServerStartWhileRequest(t *testing.T) {
	// Find the addr, but don't start the server.
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	// Start an OTLP exporter and point to the receiver.
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: ln.Addr().String(),
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	}
	set := exportertest.NewNopSettings(factory.Type())
	exp, err := factory.CreateTraces(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	require.NoError(t, exp.Start(context.Background(), host))

	// A trace with 2 spans.
	td := testdata.GenerateTraces(2)
	done := make(chan bool, 1)
	defer close(done)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	go func() {
		assert.NoError(t, exp.ConsumeTraces(ctx, td))
		done <- true
	}()

	time.Sleep(2 * time.Second)
	rcv, _ := otlpTracesReceiverOnGRPCServer(ln, false)
	defer rcv.srv.GracefulStop()
	// Wait until one of the conditions below triggers.
	select {
	case <-ctx.Done():
		t.Fail()
	case <-done:
		require.NoError(t, ctx.Err())
	}
	cancel()
}

func TestSendTracesOnResourceExhaustion(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err)
	rcv, _ := otlpTracesReceiverOnGRPCServer(ln, false)
	rcv.setExportError(status.Error(codes.ResourceExhausted, "resource exhausted"))
	defer rcv.srv.GracefulStop()

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.RetryConfig.InitialInterval = 0
	cfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: ln.Addr().String(),
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	}
	set := exportertest.NewNopSettings(factory.Type())
	exp, err := factory.CreateTraces(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)

	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))

	assert.EqualValues(t, 0, rcv.requestCount.Load())

	td := ptrace.NewTraces()
	require.NoError(t, exp.ConsumeTraces(context.Background(), td))

	assert.Never(t, func() bool {
		return rcv.requestCount.Load() > 1
	}, 1*time.Second, 5*time.Millisecond, "Should not retry if RetryInfo is not included into status details by the server.")

	rcv.requestCount.Swap(0)

	st := status.New(codes.ResourceExhausted, "resource exhausted")
	st, _ = st.WithDetails(&errdetails.RetryInfo{
		RetryDelay: durationpb.New(100 * time.Millisecond),
	})
	rcv.setExportError(st.Err())

	require.NoError(t, exp.ConsumeTraces(context.Background(), td))

	assert.Eventually(t, func() bool {
		return rcv.requestCount.Load() > 1
	}, 10*time.Second, 5*time.Millisecond, "Should retry if RetryInfo is included into status details by the server.")
}

func startServerAndMakeRequest(t *testing.T, exp exporter.Traces, td ptrace.Traces, ln net.Listener) {
	rcv, _ := otlpTracesReceiverOnGRPCServer(ln, false)
	defer rcv.srv.GracefulStop()
	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, rcv.requestCount.Load())

	// Clone the request and store as expected.
	expectedData := ptrace.NewTraces()
	td.CopyTo(expectedData)

	// Resend the request, this should succeed.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	require.NoError(t, exp.ConsumeTraces(ctx, td))
	cancel()

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return rcv.requestCount.Load() > 0
	}, 10*time.Second, 5*time.Millisecond)

	// Verify received span.
	assert.EqualValues(t, 2, rcv.totalItems.Load())
	assert.Equal(t, expectedData, rcv.getLastRequest())
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
	// Disable queuing to ensure that we execute the request when calling ConsumeLogs
	// otherwise we will not see any errors.
	cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	cfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: ln.Addr().String(),
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	}
	set := exportertest.NewNopSettings(factory.Type())
	set.BuildInfo.Description = "Collector"
	set.BuildInfo.Version = "1.2.3test"

	// For testing the "Partial success" warning.
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)

	exp, err := factory.CreateLogs(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	require.NoError(t, exp.Start(context.Background(), host))

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, rcv.requestCount.Load())

	// Send empty request.
	ld := plog.NewLogs()
	require.NoError(t, exp.ConsumeLogs(context.Background(), ld))

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return rcv.requestCount.Load() > 0
	}, 10*time.Second, 5*time.Millisecond)

	// Ensure it was received empty.
	assert.EqualValues(t, 0, rcv.totalItems.Load())

	// A request with 2 log entries.
	ld = testdata.GenerateLogs(2)

	err = exp.ConsumeLogs(context.Background(), ld)
	require.NoError(t, err)

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return rcv.requestCount.Load() > 1
	}, 10*time.Second, 5*time.Millisecond)

	// Verify received logs.
	assert.EqualValues(t, 2, rcv.requestCount.Load())
	assert.EqualValues(t, 2, rcv.totalItems.Load())
	assert.Equal(t, ld, rcv.getLastRequest())

	md := rcv.getMetadata()
	require.Len(t, md.Get("User-Agent"), 1)
	require.Contains(t, md.Get("User-Agent")[0], "Collector/1.2.3test")

	st := status.New(codes.InvalidArgument, "Invalid argument")
	rcv.setExportError(st.Err())

	// A request with 2 log entries.
	ld = testdata.GenerateLogs(2)

	err = exp.ConsumeLogs(context.Background(), ld)
	require.Error(t, err)

	rcv.setExportError(nil)

	// Return partial success
	rcv.setExportResponse(func() plogotlp.ExportResponse {
		response := plogotlp.NewExportResponse()
		partialSuccess := response.PartialSuccess()
		partialSuccess.SetErrorMessage("Some log records were not ingested")
		partialSuccess.SetRejectedLogRecords(1)

		return response
	})

	// A request with 2 log entries.
	ld = testdata.GenerateLogs(2)

	err = exp.ConsumeLogs(context.Background(), ld)
	require.NoError(t, err)
	assert.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), 1)
	assert.Contains(t, observed.FilterLevelExact(zap.WarnLevel).All()[0].Message, "Partial success")
}

func TestSendProfiles(t *testing.T) {
	// Start an OTLP-compatible receiver.
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv, _ := otlpProfilesReceiverOnGRPCServer(ln, false)
	// Also closes the connection.
	defer rcv.srv.GracefulStop()

	// Start an OTLP exporter and point to the receiver.
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	// Disable queuing to ensure that we execute the request when calling ConsumeProfiles
	// otherwise we will not see any errors.
	cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	cfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: ln.Addr().String(),
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
		Headers: configopaque.MapList{
			{Name: "header", Value: "header-value"},
		},
	}
	set := exportertest.NewNopSettings(factory.Type())
	set.BuildInfo.Description = "Collector"
	set.BuildInfo.Version = "1.2.3test"

	// For testing the "Partial success" warning.
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)

	exp, err := factory.(xexporter.Factory).CreateProfiles(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)

	defer func() {
		require.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, rcv.requestCount.Load())

	// Send empty profile.
	td := pprofile.NewProfiles()
	require.NoError(t, exp.ConsumeProfiles(context.Background(), td))

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return rcv.requestCount.Load() > 0
	}, 10*time.Second, 5*time.Millisecond)

	// Ensure it was received empty.
	assert.EqualValues(t, 0, rcv.totalItems.Load())

	// A request with 2 profiles.
	td = testdata.GenerateProfiles(2)

	err = exp.ConsumeProfiles(context.Background(), td)
	require.NoError(t, err)

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return rcv.requestCount.Load() > 1
	}, 10*time.Second, 5*time.Millisecond)

	expectedHeader := []string{"header-value"}

	// Verify received span.
	assert.EqualValues(t, 2, rcv.totalItems.Load())
	assert.EqualValues(t, 2, rcv.requestCount.Load())
	assert.Equal(t, td, rcv.getLastRequest())

	md := rcv.getMetadata()
	require.Equal(t, expectedHeader, md.Get("header"))
	require.Len(t, md.Get("User-Agent"), 1)
	require.Contains(t, md.Get("User-Agent")[0], "Collector/1.2.3test")

	// Return partial success
	rcv.setExportResponse(func() pprofileotlp.ExportResponse {
		response := pprofileotlp.NewExportResponse()
		partialSuccess := response.PartialSuccess()
		partialSuccess.SetErrorMessage("Some spans were not ingested")
		partialSuccess.SetRejectedProfiles(1)

		return response
	})

	// A request with 2 Profile entries.
	td = testdata.GenerateProfiles(2)

	err = exp.ConsumeProfiles(context.Background(), td)
	require.NoError(t, err)
	assert.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), 1)
	assert.Contains(t, observed.FilterLevelExact(zap.WarnLevel).All()[0].Message, "Partial success")
}

func TestSendProfilesWhenEndpointHasHttpScheme(t *testing.T) {
	tests := []struct {
		name               string
		useTLS             bool
		scheme             string
		gRPCClientSettings configgrpc.ClientConfig
	}{
		{
			name:               "Use https scheme",
			useTLS:             true,
			scheme:             "https://",
			gRPCClientSettings: configgrpc.ClientConfig{},
		},
		{
			name:   "Use http scheme",
			useTLS: false,
			scheme: "http://",
			gRPCClientSettings: configgrpc.ClientConfig{
				TLS: configtls.ClientConfig{
					Insecure: true,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Start an OTLP-compatible receiver.
			ln, err := net.Listen("tcp", "localhost:")
			require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
			rcv, err := otlpProfilesReceiverOnGRPCServer(ln, test.useTLS)
			require.NoError(t, err, "Failed to start mock OTLP receiver")
			// Also closes the connection.
			defer rcv.srv.GracefulStop()

			// Start an OTLP exporter and point to the receiver.
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig().(*Config)
			cfg.ClientConfig = test.gRPCClientSettings
			cfg.ClientConfig.Endpoint = test.scheme + ln.Addr().String()
			if test.useTLS {
				cfg.ClientConfig.TLS.InsecureSkipVerify = true
			}
			set := exportertest.NewNopSettings(factory.Type())
			exp, err := factory.(xexporter.Factory).CreateProfiles(context.Background(), set, cfg)
			require.NoError(t, err)
			require.NotNil(t, exp)

			defer func() {
				require.NoError(t, exp.Shutdown(context.Background()))
			}()

			host := componenttest.NewNopHost()
			require.NoError(t, exp.Start(context.Background(), host))

			// Ensure that initially there is no data in the receiver.
			assert.EqualValues(t, 0, rcv.requestCount.Load())

			// Send empty profile.
			td := pprofile.NewProfiles()
			require.NoError(t, exp.ConsumeProfiles(context.Background(), td))

			// Wait until it is received.
			assert.Eventually(t, func() bool {
				return rcv.requestCount.Load() > 0
			}, 10*time.Second, 5*time.Millisecond)

			// Ensure it was received empty.
			assert.EqualValues(t, 0, rcv.totalItems.Load())
		})
	}
}
