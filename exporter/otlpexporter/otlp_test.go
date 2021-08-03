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
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/otlpgrpc"
	"go.opentelemetry.io/collector/model/pdata"
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

type mockTracesReceiver struct {
	mockReceiver
	lastRequest pdata.Traces
}

func (r *mockTracesReceiver) Export(ctx context.Context, td pdata.Traces) (otlpgrpc.TracesResponse, error) {
	atomic.AddInt32(&r.requestCount, 1)
	atomic.AddInt32(&r.totalItems, int32(td.SpanCount()))
	r.mux.Lock()
	defer r.mux.Unlock()
	r.lastRequest = td
	r.metadata, _ = metadata.FromIncomingContext(ctx)
	return otlpgrpc.NewTracesResponse(), nil
}

func (r *mockTracesReceiver) GetLastRequest() pdata.Traces {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.lastRequest
}

func otlpTracesReceiverOnGRPCServer(ln net.Listener, useTLS bool) (*mockTracesReceiver, error) {
	sopts := []grpc.ServerOption{}

	if useTLS {
		_, currentFile, _, _ := runtime.Caller(0)
		basepath := filepath.Dir(currentFile)
		certpath := filepath.Join(basepath, "testdata/test_cert.pem")
		keypath := filepath.Join(basepath, "testdata/test_key.pem")

		creds, err := credentials.NewServerTLSFromFile(certpath, keypath)
		if err != nil {
			return nil, err
		}
		sopts = append(sopts, grpc.Creds(creds))
	}

	rcv := &mockTracesReceiver{
		mockReceiver: mockReceiver{
			srv: grpc.NewServer(sopts...),
		},
	}

	// Now run it as a gRPC server
	otlpgrpc.RegisterTracesServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv, nil
}

type mockLogsReceiver struct {
	mockReceiver
	lastRequest pdata.Logs
}

func (r *mockLogsReceiver) Export(ctx context.Context, ld pdata.Logs) (otlpgrpc.LogsResponse, error) {
	atomic.AddInt32(&r.requestCount, 1)
	atomic.AddInt32(&r.totalItems, int32(ld.LogRecordCount()))
	r.mux.Lock()
	defer r.mux.Unlock()
	r.lastRequest = ld
	r.metadata, _ = metadata.FromIncomingContext(ctx)
	return otlpgrpc.NewLogsResponse(), nil
}

func (r *mockLogsReceiver) GetLastRequest() pdata.Logs {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.lastRequest
}

func otlpLogsReceiverOnGRPCServer(ln net.Listener) *mockLogsReceiver {
	rcv := &mockLogsReceiver{
		mockReceiver: mockReceiver{
			srv: grpc.NewServer(),
		},
	}

	// Now run it as a gRPC server
	otlpgrpc.RegisterLogsServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
}

type mockMetricsReceiver struct {
	mockReceiver
	lastRequest pdata.Metrics
}

func (r *mockMetricsReceiver) Export(ctx context.Context, md pdata.Metrics) (otlpgrpc.MetricsResponse, error) {
	atomic.AddInt32(&r.requestCount, 1)
	atomic.AddInt32(&r.totalItems, int32(md.DataPointCount()))
	r.mux.Lock()
	defer r.mux.Unlock()
	r.lastRequest = md
	r.metadata, _ = metadata.FromIncomingContext(ctx)
	return otlpgrpc.NewMetricsResponse(), nil
}

func (r *mockMetricsReceiver) GetLastRequest() pdata.Metrics {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.lastRequest
}

func otlpMetricsReceiverOnGRPCServer(ln net.Listener) *mockMetricsReceiver {
	rcv := &mockMetricsReceiver{
		mockReceiver: mockReceiver{
			srv: grpc.NewServer(),
		},
	}

	// Now run it as a gRPC server
	otlpgrpc.RegisterMetricsServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
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
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: ln.Addr().String(),
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
		Headers: map[string]string{
			"header": "header-value",
		},
	}
	set := componenttest.NewNopExporterCreateSettings()
	exp, err := factory.CreateTracesExporter(context.Background(), set, cfg)
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
	td := pdata.NewTraces()
	assert.NoError(t, exp.ConsumeTraces(context.Background(), td))

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, 10*time.Second, 5*time.Millisecond)

	// Ensure it was received empty.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.totalItems))

	// A trace with 2 spans.
	td = testdata.GenerateTracesTwoSpansSameResource()

	err = exp.ConsumeTraces(context.Background(), td)
	assert.NoError(t, err)

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 1
	}, 10*time.Second, 5*time.Millisecond)

	expectedHeader := []string{"header-value"}

	// Verify received span.
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.totalItems))
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.requestCount))
	assert.EqualValues(t, td, rcv.GetLastRequest())

	require.EqualValues(t, rcv.GetMetadata().Get("header"), expectedHeader)
}

func TestSendTracesWhenEndpointHasHttpScheme(t *testing.T) {
	tests := []struct {
		name               string
		useTLS             bool
		scheme             string
		gRPCClientSettings configgrpc.GRPCClientSettings
	}{
		{
			name:               "Use https scheme",
			useTLS:             true,
			scheme:             "https://",
			gRPCClientSettings: configgrpc.GRPCClientSettings{},
		},
		{
			name:   "Use http scheme",
			useTLS: false,
			scheme: "http://",
			gRPCClientSettings: configgrpc.GRPCClientSettings{
				TLSSetting: configtls.TLSClientSetting{
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
			cfg.GRPCClientSettings = test.gRPCClientSettings
			cfg.GRPCClientSettings.Endpoint = test.scheme + ln.Addr().String()
			if test.useTLS {
				cfg.GRPCClientSettings.TLSSetting.InsecureSkipVerify = true
			}
			set := componenttest.NewNopExporterCreateSettings()
			exp, err := factory.CreateTracesExporter(context.Background(), set, cfg)
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
			td := pdata.NewTraces()
			assert.NoError(t, exp.ConsumeTraces(context.Background(), td))

			// Wait until it is received.
			assert.Eventually(t, func() bool {
				return atomic.LoadInt32(&rcv.requestCount) > 0
			}, 10*time.Second, 5*time.Millisecond)

			// Ensure it was received empty.
			assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.totalItems))
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
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: ln.Addr().String(),
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
		Headers: map[string]string{
			"header": "header-value",
		},
	}
	set := componenttest.NewNopExporterCreateSettings()
	exp, err := factory.CreateMetricsExporter(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	assert.NoError(t, exp.Start(context.Background(), host))

	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.requestCount))

	// Send empty metric.
	md := pdata.NewMetrics()
	assert.NoError(t, exp.ConsumeMetrics(context.Background(), md))

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, 10*time.Second, 5*time.Millisecond)

	// Ensure it was received empty.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.totalItems))

	// Send two metrics.
	md = testdata.GenerateMetricsTwoMetrics()

	err = exp.ConsumeMetrics(context.Background(), md)
	assert.NoError(t, err)

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 1
	}, 10*time.Second, 5*time.Millisecond)

	expectedHeader := []string{"header-value"}

	// Verify received metrics.
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.requestCount))
	assert.EqualValues(t, 4, atomic.LoadInt32(&rcv.totalItems))
	assert.EqualValues(t, md, rcv.GetLastRequest())

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
	set := componenttest.NewNopExporterCreateSettings()
	exp, err := factory.CreateTracesExporter(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	assert.NoError(t, exp.Start(context.Background(), host))

	// A trace with 2 spans.
	td := testdata.GenerateTracesTwoSpansSameResource()
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
	set := componenttest.NewNopExporterCreateSettings()
	exp, err := factory.CreateTracesExporter(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	defer func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	}()

	host := componenttest.NewNopHost()

	assert.NoError(t, exp.Start(context.Background(), host))

	// A trace with 2 spans.
	td := testdata.GenerateTracesTwoSpansSameResource()
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
		assert.NoError(t, ctx.Err())
	}
	cancel()
}

func startServerAndMakeRequest(t *testing.T, exp component.TracesExporter, td pdata.Traces, ln net.Listener) {
	rcv, _ := otlpTracesReceiverOnGRPCServer(ln, false)
	defer rcv.srv.GracefulStop()
	// Ensure that initially there is no data in the receiver.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.requestCount))

	// Clone the request and store as expected.
	expectedData := td.Clone()

	// Resend the request, this should succeed.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	assert.NoError(t, exp.ConsumeTraces(ctx, td))
	cancel()

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, 10*time.Second, 5*time.Millisecond)

	// Verify received span.
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.totalItems))
	assert.EqualValues(t, expectedData, rcv.GetLastRequest())
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
	set := componenttest.NewNopExporterCreateSettings()
	exp, err := factory.CreateLogsExporter(context.Background(), set, cfg)
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
	ld := pdata.NewLogs()
	assert.NoError(t, exp.ConsumeLogs(context.Background(), ld))

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 0
	}, 10*time.Second, 5*time.Millisecond)

	// Ensure it was received empty.
	assert.EqualValues(t, 0, atomic.LoadInt32(&rcv.totalItems))

	// A request with 2 log entries.
	ld = testdata.GenerateLogsTwoLogRecordsSameResource()

	err = exp.ConsumeLogs(context.Background(), ld)
	assert.NoError(t, err)

	// Wait until it is received.
	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&rcv.requestCount) > 1
	}, 10*time.Second, 5*time.Millisecond)

	// Verify received logs.
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.requestCount))
	assert.EqualValues(t, 2, atomic.LoadInt32(&rcv.totalItems))
	assert.EqualValues(t, ld, rcv.GetLastRequest())
}
