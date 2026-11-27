// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

// contextKey is a private type for keys defined in this test.
type contextKey int

// Key for the slice of middleware names in the context.
const middlewareCallsKey contextKey = 0

// getMiddlewareCalls retrieves the middleware calls from context or returns an empty slice.
func getMiddlewareCalls(ctx context.Context) []string {
	calls, ok := ctx.Value(middlewareCallsKey).([]string)
	if !ok {
		return []string{}
	}
	return calls
}

// testServerMiddleware is a test implementation of configmiddleware.Middleware
type testServerMiddleware struct {
	extension.Extension
	extensionmiddleware.GetGRPCServerOptionsFunc
}

func newTestServerMiddleware(name string) extension.Extension {
	return &testServerMiddleware{
		Extension: extensionmiddlewaretest.NewNop(),
		GetGRPCServerOptionsFunc: func(_ context.Context) ([]grpc.ServerOption, error) {
			return []grpc.ServerOption{grpc.ChainUnaryInterceptor(
				func(
					ctx context.Context,
					req any, _ *grpc.UnaryServerInfo,
					handler grpc.UnaryHandler,
				) (any, error) {
					ctx = context.WithValue(ctx, middlewareCallsKey, append(getMiddlewareCalls(ctx), name))
					return handler(ctx, req)
				},
			)}, nil
		},
	}
}

func TestGrpcServerUnaryInterceptor(t *testing.T) {
	// Register two test extensions
	extensions := map[component.ID]component.Component{
		component.MustNewID("test1"): newTestServerMiddleware("test1"),
		component.MustNewID("test2"): newTestServerMiddleware("test2"),
	}

	// Setup the server with both middleware options
	server := &grpcTraceServer{}
	var addr string

	// Create the server with middleware interceptors
	{
		var srv *grpc.Server
		srv, addr = server.startTestServerWithExtensions(t, configoptional.Some(ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  "localhost:0",
				Transport: confignet.TransportTypeTCP,
			},
			Middlewares: []configmiddleware.Config{
				newTestMiddlewareConfig("test1"),
				newTestMiddlewareConfig("test2"),
			},
		}), extensions)
		defer srv.Stop()
	}

	// Send a request to trigger the interceptors
	resp, errResp := sendTestRequest(t, ClientConfig{
		Endpoint: addr,
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	})
	require.NoError(t, errResp)
	require.NotNil(t, resp)

	// Verify interceptors were called in the correct order
	assert.Equal(t, []string{"test1", "test2"}, getMiddlewareCalls(server.recordedContext))
}

// TestServerMiddlewareToServerErrors tests failure cases for the ToServer method
// specifically related to middleware resolution and API calls.
func TestServerMiddlewareToServerErrors(t *testing.T) {
	tests := []struct {
		name       string
		extensions map[component.ID]component.Component
		config     ServerConfig
		errText    string
	}{
		{
			name:       "extension_not_found",
			extensions: map[component.ID]component.Component{},
			config: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("nonexistent"),
					},
				},
			},
			errText: "failed to resolve middleware \"nonexistent\": middleware not found",
		},
		{
			name: "get_server_options_fails",
			extensions: map[component.ID]component.Component{
				component.MustNewID("errormw"): extensionmiddlewaretest.NewErr(errors.New("get server options failed")),
			},
			config: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("errormw"),
					},
				},
			},
			errText: "get server options failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test creating the server with middleware errors
			server := &grpcTraceServer{}
			srv, err := server.startTestServerWithExtensionsError(t, tc.config, tc.extensions)
			if srv != nil {
				srv.Stop()
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}

// testPreDecodeMiddleware is a test middleware extension that checks unary RPCs
// before their request message is decoded.
type testPreDecodeMiddleware struct {
	extension.Extension
	extensionmiddleware.GetGRPCServerOptionsFunc
	extensionmiddleware.GetGRPCServerPreDecodeCheckFunc
}

func newTestPreDecodeMiddleware(check extensionmiddleware.GRPCPreDecodeCheckFunc) extension.Extension {
	return &testPreDecodeMiddleware{
		Extension: extensionmiddlewaretest.NewNop(),
		GetGRPCServerPreDecodeCheckFunc: func(context.Context) (extensionmiddleware.GRPCPreDecodeCheckFunc, error) {
			return check, nil
		},
	}
}

// preDecodeCallRecorder records the pre-decode checks and authentications in the order they run.
type preDecodeCallRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *preDecodeCallRecorder) record(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, name)
}

func (r *preDecodeCallRecorder) get() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// exportCompressedTraces sends a request whose body is large enough to be compressed
// with countingGzip, so tests can tell whether the server decompressed it.
func exportCompressedTraces(t *testing.T, addr string, md metadata.MD) error {
	countingGzipOnce.Do(func() {
		encoding.RegisterCompressor(countingGzip{Compressor: encoding.GetCompressor("gzip")})
	})
	conn, err := (&ClientConfig{
		Endpoint: addr,
		TLS:      configtls.ClientConfig{Insecure: true},
	}).ToClientConn(context.Background(), nil, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	defer func() { assert.NoError(t, conn.Close()) }()

	ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), 5*time.Second)
	defer cancel()
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName(strings.Repeat("a", 1024))
	_, err = ptraceotlp.NewGRPCClient(conn).Export(ctx, ptraceotlp.NewExportRequestFromTraces(td),
		grpc.UseCompressor(countingGzipName), grpc.WaitForReady(true))
	return err
}

// TestServerPreDecodeChecks verifies that the pre-decode checks of the middlewares run in
// the configured order, and that the first error rejects the RPC before its request is
// decompressed.
func TestServerPreDecodeChecks(t *testing.T) {
	rejectErr := status.Error(codes.ResourceExhausted, "rejected")

	tests := []struct {
		name      string
		rejectAt  string
		wantCalls []string
		wantCode  codes.Code
	}{
		{
			name:      "all_checks_pass",
			wantCalls: []string{"check1", "check2"},
			wantCode:  codes.OK,
		},
		{
			name:      "first_check_rejects",
			rejectAt:  "check1",
			wantCalls: []string{"check1"},
			wantCode:  codes.ResourceExhausted,
		},
		{
			name:      "second_check_rejects",
			rejectAt:  "check2",
			wantCalls: []string{"check1", "check2"},
			wantCode:  codes.ResourceExhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &preDecodeCallRecorder{}
			newCheck := func(name string) extensionmiddleware.GRPCPreDecodeCheckFunc {
				return func(ctx context.Context) error {
					recorder.record(name)
					assert.NotNil(t, client.FromContext(ctx).Addr, "the check context should carry the client address")
					if name == tt.rejectAt {
						return rejectErr
					}
					return nil
				}
			}
			extensions := map[component.ID]component.Component{
				component.MustNewID("check1"): newTestPreDecodeMiddleware(newCheck("check1")),
				component.MustNewID("check2"): newTestPreDecodeMiddleware(newCheck("check2")),
			}

			mock := &grpcTraceServer{}
			srv, addr := mock.startTestServerWithExtensions(t, configoptional.Some(ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Middlewares: []configmiddleware.Config{
					newTestMiddlewareConfig("check1"),
					newTestMiddlewareConfig("check2"),
				},
			}), extensions)
			defer srv.Stop()

			countingGzipDecompresses.Store(0)
			err := exportCompressedTraces(t, addr, nil)

			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Equal(t, tt.wantCalls, recorder.get())
			if tt.wantCode == codes.OK {
				assert.NotNil(t, mock.recordedContext)
				assert.Positive(t, countingGzipDecompresses.Load())
			} else {
				assert.Nil(t, mock.recordedContext, "the rejected RPC reached the service")
				assert.Zero(t, countingGzipDecompresses.Load(), "the rejected request was decompressed")
			}
		})
	}
}

// TestServerPreDecodeChecksRunAfterAuth verifies that the pre-decode checks see the context
// returned by the authenticator, and do not run for RPCs that fail authentication.
func TestServerPreDecodeChecksRunAfterAuth(t *testing.T) {
	type authKey struct{}

	tests := []struct {
		name      string
		md        metadata.MD
		wantCalls []string
		wantCode  codes.Code
	}{
		{
			name:      "authenticated",
			md:        metadata.Pairs("authorization", "ok"),
			wantCalls: []string{"auth", "check"},
			wantCode:  codes.OK,
		},
		{
			name:      "not_authenticated",
			wantCalls: []string{"auth"},
			wantCode:  codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &preDecodeCallRecorder{}
			extensions := map[component.ID]component.Component{
				mockID: struct {
					component.StartFunc
					component.ShutdownFunc
					extensionauth.Server
				}{
					Server: newMockAuthServer(func(ctx context.Context, headers map[string][]string) (context.Context, error) {
						recorder.record("auth")
						if len(headers["authorization"]) == 0 {
							return nil, errors.New("not authenticated")
						}
						return context.WithValue(ctx, authKey{}, true), nil
					}),
				},
				component.MustNewID("check"): newTestPreDecodeMiddleware(func(ctx context.Context) error {
					recorder.record("check")
					assert.Equal(t, true, ctx.Value(authKey{}), "the check context should come from the authenticator")
					return nil
				}),
			}

			mock := &grpcTraceServer{}
			srv, addr := mock.startTestServerWithExtensions(t, configoptional.Some(ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Auth:        configoptional.Some(configauth.Config{AuthenticatorID: mockID}),
				Middlewares: []configmiddleware.Config{newTestMiddlewareConfig("check")},
			}), extensions)
			defer srv.Stop()

			err := exportCompressedTraces(t, addr, tt.md)

			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Equal(t, tt.wantCalls, recorder.get())
		})
	}
}

func TestServiceRegistrarPreDecodeChecks(t *testing.T) {
	sc := ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "localhost:0",
			Transport: confignet.TransportTypeTCP,
		},
		Middlewares: []configmiddleware.Config{newTestMiddlewareConfig("mw")},
	}

	tests := []struct {
		name          string
		middleware    extension.Extension
		wantServer    bool
		wantErrString string
	}{
		{
			name:       "middleware_without_pre_decode_check",
			middleware: newTestServerMiddleware("mw"),
			wantServer: true,
		},
		{
			name:       "nil_pre_decode_check",
			middleware: newTestPreDecodeMiddleware(nil),
			wantServer: true,
		},
		{
			name: "pre_decode_check",
			middleware: newTestPreDecodeMiddleware(func(context.Context) error {
				return nil
			}),
		},
		{
			name: "pre_decode_check_error",
			middleware: &testPreDecodeMiddleware{
				Extension: extensionmiddlewaretest.NewNop(),
				GetGRPCServerPreDecodeCheckFunc: func(context.Context) (extensionmiddleware.GRPCPreDecodeCheckFunc, error) {
					return nil, errors.New("check failed")
				},
			},
			wantErrString: "failed to get gRPC server pre-decode check from middleware: check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extensions := map[component.ID]component.Component{
				component.MustNewID("mw"): tt.middleware,
			}
			srv, err := sc.ToServer(context.Background(), extensions, componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)
			defer srv.Stop()

			registrar, err := sc.ServiceRegistrar(context.Background(), extensions, srv)
			if tt.wantErrString != "" {
				require.EqualError(t, err, tt.wantErrString)
				return
			}
			require.NoError(t, err)
			if tt.wantServer {
				assert.Same(t, srv, registrar)
			} else {
				assert.IsType(t, &preDecodeCheckRegistrar{}, registrar)
			}
		})
	}
}

// TestServerPreDecodeCheckContext verifies that the pre-decode checks see the same client
// information as the service, which gets it from the interceptors.
func TestServerPreDecodeCheckContext(t *testing.T) {
	for _, includeMetadata := range []bool{false, true} {
		t.Run(fmt.Sprintf("include_metadata_%t", includeMetadata), func(t *testing.T) {
			checkInfo := make(chan client.Info, 1)
			extensions := map[component.ID]component.Component{
				component.MustNewID("check"): newTestPreDecodeMiddleware(func(ctx context.Context) error {
					checkInfo <- client.FromContext(ctx)
					return nil
				}),
			}

			mock := &grpcTraceServer{}
			srv, addr := mock.startTestServerWithExtensions(t, configoptional.Some(ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				IncludeMetadata: includeMetadata,
				Middlewares:     []configmiddleware.Config{newTestMiddlewareConfig("check")},
			}), extensions)
			defer srv.Stop()

			require.NoError(t, exportCompressedTraces(t, addr, metadata.Pairs("x-tenant", "acme")))

			got := <-checkInfo
			want := client.FromContext(mock.recordedContext)
			assert.NotNil(t, got.Addr)
			assert.Equal(t, want.Addr, got.Addr)
			assert.Equal(t, want.Metadata.Get("x-tenant"), got.Metadata.Get("x-tenant"))
			if includeMetadata {
				assert.Equal(t, []string{"acme"}, got.Metadata.Get("x-tenant"))
			} else {
				assert.Empty(t, got.Metadata.Get("x-tenant"))
			}
		})
	}
}

// TestServerPreDecodeChecksMultipleMethods verifies that the pre-decode checks apply to
// every unary method of a service, and not to its streaming methods.
func TestServerPreDecodeChecksMultipleMethods(t *testing.T) {
	var checkCalls atomic.Int64
	extensions := map[component.ID]component.Component{
		component.MustNewID("check"): newTestPreDecodeMiddleware(func(context.Context) error {
			checkCalls.Add(1)
			return status.Error(codes.ResourceExhausted, "rejected")
		}),
	}
	sc := ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "localhost:0",
			Transport: confignet.TransportTypeTCP,
		},
		Middlewares: []configmiddleware.Config{newTestMiddlewareConfig("check")},
	}
	listener, err := sc.NetAddr.Listen(context.Background())
	require.NoError(t, err)
	srv, err := sc.ToServer(context.Background(), extensions, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	registrar, err := sc.ServiceRegistrar(context.Background(), extensions, srv)
	require.NoError(t, err)
	// The health service has two unary methods and a streaming one.
	healthpb.RegisterHealthServer(registrar, health.NewServer())
	go func() { _ = srv.Serve(listener) }()
	defer srv.Stop()

	conn, err := (&ClientConfig{
		Endpoint: listener.Addr().String(),
		TLS:      configtls.ClientConfig{Insecure: true},
	}).ToClientConn(context.Background(), nil, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	defer func() { assert.NoError(t, conn.Close()) }()
	healthClient := healthpb.NewHealthClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = healthClient.Check(ctx, &healthpb.HealthCheckRequest{}, grpc.WaitForReady(true))
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	_, err = healthClient.List(ctx, &healthpb.HealthListRequest{})
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	assert.EqualValues(t, 2, checkCalls.Load())

	stream, err := healthClient.Watch(ctx, &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	resp, err := stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.GetStatus())
	assert.EqualValues(t, 2, checkCalls.Load(), "the streaming RPC was checked")
}
