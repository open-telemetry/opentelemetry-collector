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

package configgrpc

import (
	"context"
	"io/ioutil"
	"net"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	pb_testproto "github.com/grpc-ecosystem/go-grpc-middleware/testing/testproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/model/otlpgrpc"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

func TestDefaultGrpcClientSettings(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	gcs := &GRPCClientSettings{
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	opts, err := gcs.ToDialOptions(componenttest.NewNopHost(), tt.TelemetrySettings)
	assert.NoError(t, err)
	assert.Len(t, opts, 3)
}

func TestAllGrpcClientSettings(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	gcs := &GRPCClientSettings{
		Headers: map[string]string{
			"test": "test",
		},
		Endpoint:    "localhost:1234",
		Compression: "gzip",
		TLSSetting: configtls.TLSClientSetting{
			Insecure: false,
		},
		Keepalive: &KeepaliveClientConfig{
			Time:                time.Second,
			Timeout:             time.Second,
			PermitWithoutStream: true,
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		WaitForReady:    true,
		BalancerName:    "round_robin",
		Auth:            &configauth.Authentication{AuthenticatorID: config.NewComponentID("testauth")},
	}

	host := &mockHost{
		ext: map[config.ComponentID]component.Extension{
			config.NewComponentID("testauth"): &configauth.MockClientAuthenticator{},
		},
	}

	opts, err := gcs.ToDialOptions(host, tt.TelemetrySettings)
	assert.NoError(t, err)
	assert.Len(t, opts, 9)
}

func TestDefaultGrpcServerSettings(t *testing.T) {
	gss := &GRPCServerSettings{}
	opts, err := gss.ToServerOption(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	_ = grpc.NewServer(opts...)

	assert.NoError(t, err)
	assert.Len(t, opts, 2)
}

func TestAllGrpcServerSettingsExceptAuth(t *testing.T) {
	gss := &GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  "localhost:1234",
			Transport: "tcp",
		},
		TLSSetting: &configtls.TLSServerSetting{
			TLSSetting:   configtls.TLSSetting{},
			ClientCAFile: "",
		},
		MaxRecvMsgSizeMiB:    1,
		MaxConcurrentStreams: 1024,
		ReadBufferSize:       1024,
		WriteBufferSize:      1024,
		Keepalive: &KeepaliveServerConfig{
			ServerParameters: &KeepaliveServerParameters{
				MaxConnectionIdle:     time.Second,
				MaxConnectionAge:      time.Second,
				MaxConnectionAgeGrace: time.Second,
				Time:                  time.Second,
				Timeout:               time.Second,
			},
			EnforcementPolicy: &KeepaliveEnforcementPolicy{
				MinTime:             time.Second,
				PermitWithoutStream: true,
			},
		},
	}
	opts, err := gss.ToServerOption(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	_ = grpc.NewServer(opts...)

	assert.NoError(t, err)
	assert.Len(t, opts, 9)
}

func TestGrpcServerAuthSettings(t *testing.T) {
	gss := &GRPCServerSettings{}

	// sanity check
	_, err := gss.ToServerOption(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	// test
	gss.Auth = &configauth.Authentication{
		AuthenticatorID: config.NewComponentID("mock"),
	}
	host := &mockHost{
		ext: map[config.ComponentID]component.Extension{
			config.NewComponentID("mock"): &configauth.MockServerAuthenticator{},
		},
	}
	opts, err := gss.ToServerOption(host, componenttest.NewNopTelemetrySettings())
	_ = grpc.NewServer(opts...)

	// verify
	assert.NoError(t, err)
	assert.NotNil(t, opts)
}

func TestGRPCClientSettingsError(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	tests := []struct {
		settings GRPCClientSettings
		err      string
		host     component.Host
	}{
		{
			err: "^failed to load TLS config: failed to load CA CertPool: failed to load CA /doesnt/exist:",
			settings: GRPCClientSettings{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSSetting: configtls.TLSClientSetting{
					TLSSetting: configtls.TLSSetting{
						CAFile: "/doesnt/exist",
					},
					Insecure:   false,
					ServerName: "",
				},
				Keepalive: nil,
			},
		},
		{
			err: "^failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither",
			settings: GRPCClientSettings{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSSetting: configtls.TLSClientSetting{
					TLSSetting: configtls.TLSSetting{
						CertFile: "/doesnt/exist",
					},
					Insecure:   false,
					ServerName: "",
				},
				Keepalive: nil,
			},
		},
		{
			err: "invalid balancer_name: test",
			settings: GRPCClientSettings{
				Headers: map[string]string{
					"test": "test",
				},
				Endpoint:    "localhost:1234",
				Compression: "gzip",
				TLSSetting: configtls.TLSClientSetting{
					Insecure: false,
				},
				Keepalive: &KeepaliveClientConfig{
					Time:                time.Second,
					Timeout:             time.Second,
					PermitWithoutStream: true,
				},
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				WaitForReady:    true,
				BalancerName:    "test",
			},
		},
		{
			err: "failed to resolve authenticator \"doesntexist\": authenticator not found",
			settings: GRPCClientSettings{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorID: config.NewComponentID("doesntexist")},
			},
			host: &mockHost{ext: map[config.ComponentID]component.Extension{}},
		},
		{
			err: "no extensions configuration available",
			settings: GRPCClientSettings{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorID: config.NewComponentID("doesntexist")},
			},
			host: &mockHost{},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			opts, err := test.settings.ToDialOptions(test.host, tt.TelemetrySettings)
			assert.Nil(t, opts)
			assert.Error(t, err)
			assert.Regexp(t, test.err, err)
		})
	}
}

func TestUseSecure(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	gcs := &GRPCClientSettings{
		Headers:     nil,
		Endpoint:    "",
		Compression: "",
		TLSSetting:  configtls.TLSClientSetting{},
		Keepalive:   nil,
	}
	dialOpts, err := gcs.ToDialOptions(componenttest.NewNopHost(), tt.TelemetrySettings)
	assert.NoError(t, err)
	assert.Len(t, dialOpts, 3)
}

func TestGRPCServerSettingsError(t *testing.T) {
	tests := []struct {
		settings GRPCServerSettings
		err      string
	}{
		{
			err: "^failed to load TLS config: failed to load CA CertPool: failed to load CA /doesnt/exist:",
			settings: GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  "127.0.0.1:1234",
					Transport: "tcp",
				},
				TLSSetting: &configtls.TLSServerSetting{
					TLSSetting: configtls.TLSSetting{
						CAFile: "/doesnt/exist",
					},
				},
			},
		},
		{
			err: "^failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither",
			settings: GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  "127.0.0.1:1234",
					Transport: "tcp",
				},
				TLSSetting: &configtls.TLSServerSetting{
					TLSSetting: configtls.TLSSetting{
						CertFile: "/doesnt/exist",
					},
				},
			},
		},
		{
			err: "^failed to load TLS config: failed to load client CA CertPool: failed to load CA /doesnt/exist:",
			settings: GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  "127.0.0.1:1234",
					Transport: "tcp",
				},
				TLSSetting: &configtls.TLSServerSetting{
					ClientCAFile: "/doesnt/exist",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			opts, err := test.settings.ToServerOption(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
			_ = grpc.NewServer(opts...)

			assert.Regexp(t, test.err, err)
		})
	}
}

func TestGRPCServerSettings_ToListener_Error(t *testing.T) {
	settings := GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  "127.0.0.1:1234567",
			Transport: "tcp",
		},
		TLSSetting: &configtls.TLSServerSetting{
			TLSSetting: configtls.TLSSetting{
				CertFile: "/doesnt/exist",
			},
		},
		Keepalive: nil,
	}
	_, err := settings.ToListener()
	assert.Error(t, err)
}

func TestGetGRPCCompressionKey(t *testing.T) {
	if GetGRPCCompressionKey("gzip") != CompressionGzip {
		t.Error("gzip is marked as supported but returned unsupported")
	}

	if GetGRPCCompressionKey("Gzip") != CompressionGzip {
		t.Error("Capitalization of CompressionGzip should not matter")
	}

	if GetGRPCCompressionKey("snappy") != CompressionSnappy {
		t.Error("snappy is marked as supported but returned unsupported")
	}

	if GetGRPCCompressionKey("Snappy") != CompressionSnappy {
		t.Error("Capitalization of CompressionSnappy should not matter")
	}

	if GetGRPCCompressionKey("zstd") != CompressionZstd {
		t.Error("zstd is marked as supported but returned unsupported")
	}

	if GetGRPCCompressionKey("Zstd") != CompressionZstd {
		t.Error("Capitalization of CompressionZstd should not matter")
	}

	if GetGRPCCompressionKey("badType") != CompressionUnsupported {
		t.Error("badType is not supported but was returned as supported")
	}
}

func TestHttpReception(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	tests := []struct {
		name           string
		tlsServerCreds *configtls.TLSServerSetting
		tlsClientCreds *configtls.TLSClientSetting
		hasError       bool
	}{
		{
			name:           "noTLS",
			tlsServerCreds: nil,
			tlsClientCreds: &configtls.TLSClientSetting{
				Insecure: true,
			},
		},
		{
			name: "TLS",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "server.crt"),
					KeyFile:  path.Join(".", "testdata", "server.key"),
				},
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: path.Join(".", "testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
		},
		{
			name: "NoServerCertificates",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: path.Join(".", "testdata", "ca.crt"),
				},
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: path.Join(".", "testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
		{
			name: "mTLS",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "server.crt"),
					KeyFile:  path.Join(".", "testdata", "server.key"),
				},
				ClientCAFile: path.Join(".", "testdata", "ca.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "client.crt"),
					KeyFile:  path.Join(".", "testdata", "client.key"),
				},
				ServerName: "localhost",
			},
		},
		{
			name: "NoClientCertificate",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "server.crt"),
					KeyFile:  path.Join(".", "testdata", "server.key"),
				},
				ClientCAFile: path.Join(".", "testdata", "ca.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: path.Join(".", "testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
		{
			name: "WrongClientCA",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "server.crt"),
					KeyFile:  path.Join(".", "testdata", "server.key"),
				},
				ClientCAFile: path.Join(".", "testdata", "server.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "client.crt"),
					KeyFile:  path.Join(".", "testdata", "client.key"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
	}
	// prepare

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gss := &GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  "localhost:0",
					Transport: "tcp",
				},
				TLSSetting: test.tlsServerCreds,
			}
			ln, err := gss.ToListener()
			assert.NoError(t, err)
			opts, err := gss.ToServerOption(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
			assert.NoError(t, err)
			s := grpc.NewServer(opts...)
			otlpgrpc.RegisterTracesServer(s, &grpcTraceServer{})

			go func() {
				_ = s.Serve(ln)
			}()

			gcs := &GRPCClientSettings{
				Endpoint:   ln.Addr().String(),
				TLSSetting: *test.tlsClientCreds,
			}
			clientOpts, errClient := gcs.ToDialOptions(componenttest.NewNopHost(), tt.TelemetrySettings)
			assert.NoError(t, errClient)
			grpcClientConn, errDial := grpc.Dial(gcs.Endpoint, clientOpts...)
			assert.NoError(t, errDial)
			client := otlpgrpc.NewTracesClient(grpcClientConn)
			ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
			resp, errResp := client.Export(ctx, otlpgrpc.NewTracesRequest(), grpc.WaitForReady(true))
			if test.hasError {
				assert.Error(t, errResp)
			} else {
				assert.NoError(t, errResp)
				assert.NotNil(t, resp)
			}
			cancelFunc()
			s.Stop()
		})
	}
}

func TestReceiveOnUnixDomainSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on windows")
	}
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	socketName := tempSocketName(t)
	gss := &GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  socketName,
			Transport: "unix",
		},
	}
	ln, err := gss.ToListener()
	assert.NoError(t, err)
	opts, err := gss.ToServerOption(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	assert.NoError(t, err)
	s := grpc.NewServer(opts...)
	otlpgrpc.RegisterTracesServer(s, &grpcTraceServer{})

	go func() {
		_ = s.Serve(ln)
	}()

	gcs := &GRPCClientSettings{
		Endpoint: "unix://" + ln.Addr().String(),
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	clientOpts, errClient := gcs.ToDialOptions(componenttest.NewNopHost(), tt.TelemetrySettings)
	assert.NoError(t, errClient)
	grpcClientConn, errDial := grpc.Dial(gcs.Endpoint, clientOpts...)
	assert.NoError(t, errDial)
	client := otlpgrpc.NewTracesClient(grpcClientConn)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
	resp, errResp := client.Export(ctx, otlpgrpc.NewTracesRequest(), grpc.WaitForReady(true))
	assert.NoError(t, errResp)
	assert.NotNil(t, resp)
	cancelFunc()
	s.Stop()
}

func TestContextWithClient(t *testing.T) {
	testCases := []struct {
		desc     string
		input    context.Context
		expected client.Info
	}{
		{
			desc:     "no peer information, empty client",
			input:    context.Background(),
			expected: client.Info{},
		},
		{
			desc: "existing client with IP, no peer information",
			input: client.NewContext(context.Background(), client.Info{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 4),
				},
			}),
			expected: client.Info{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 4),
				},
			},
		},
		{
			desc: "empty client, with peer information",
			input: peer.NewContext(context.Background(), &peer.Peer{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 4),
				},
			}),
			expected: client.Info{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 4),
				},
			},
		},
		{
			desc: "existing client, existing IP gets overridden with peer information",
			input: peer.NewContext(client.NewContext(context.Background(), client.Info{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 4),
				},
			}), &peer.Peer{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 5),
				},
			}),
			expected: client.Info{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 5),
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cl := client.FromContext(contextWithClient(tC.input))
			assert.Equal(t, tC.expected, cl)
		})
	}
}

func TestClientInfoInterceptors(t *testing.T) {
	testCases := []struct {
		desc   string
		tester func(context.Context, pb_testproto.TestServiceClient)
	}{
		{
			desc: "stream",
			tester: func(ctx context.Context, cl pb_testproto.TestServiceClient) {
				stream, err := cl.PingList(ctx, &pb_testproto.PingRequest{})
				require.NoError(t, err)

				_, err = stream.Recv()
				require.NoError(t, err)
			},
		},
		{
			desc: "unary",
			tester: func(ctx context.Context, cl pb_testproto.TestServiceClient) {
				resp, errResp := cl.Ping(ctx, &pb_testproto.PingRequest{})
				require.NoError(t, errResp)
				require.NotNil(t, resp)
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			mock := &pingService{}
			var l net.Listener

			// prepare the server
			{
				gss := &GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "localhost:0",
						Transport: "tcp",
					},
				}
				opts, err := gss.ToServerOption(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
				require.NoError(t, err)
				srv := grpc.NewServer(opts...)
				pb_testproto.RegisterTestServiceServer(srv, mock)

				defer srv.Stop()

				l, err = gss.ToListener()
				require.NoError(t, err)

				go func() {
					_ = srv.Serve(l)
				}()
			}

			// prepare the client and execute a RPC
			{
				gcs := &GRPCClientSettings{
					Endpoint: l.Addr().String(),
					TLSSetting: configtls.TLSClientSetting{
						Insecure: true,
					},
				}

				tt, err := obsreporttest.SetupTelemetry()
				require.NoError(t, err)
				defer func() {
					require.NoError(t, tt.Shutdown(context.Background()))
				}()

				clientOpts, errClient := gcs.ToDialOptions(componenttest.NewNopHost(), tt.TelemetrySettings)
				require.NoError(t, errClient)

				grpcClientConn, errDial := grpc.Dial(gcs.Endpoint, clientOpts...)
				require.NoError(t, errDial)

				cl := pb_testproto.NewTestServiceClient(grpcClientConn)
				ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancelFunc()

				// test
				tC.tester(ctx, cl)
			}

			// verify
			cl := client.FromContext(mock.recordedContext)

			// the client address is something like 127.0.0.1:41086
			assert.Contains(t, cl.Addr.String(), "127.0.0.1")
		})
	}
}

type grpcTraceServer struct {
}

func (gts *grpcTraceServer) Export(ctx context.Context, _ otlpgrpc.TracesRequest) (otlpgrpc.TracesResponse, error) {
	return otlpgrpc.NewTracesResponse(), nil
}

// pingService comes from the grpc-middleware project and is made available to test
// interceptors and other gRPC middleware code. We use it here as it provides a
// streaming interface, which we don't have for OTLP services.
type pingService struct {
	recordedContext context.Context
}

func (s *pingService) PingEmpty(ctx context.Context, _ *pb_testproto.Empty) (*pb_testproto.PingResponse, error) {
	s.recordedContext = ctx
	return &pb_testproto.PingResponse{}, nil
}

func (s *pingService) Ping(ctx context.Context, _ *pb_testproto.PingRequest) (*pb_testproto.PingResponse, error) {
	s.recordedContext = ctx
	return &pb_testproto.PingResponse{}, nil
}

func (s *pingService) PingError(ctx context.Context, _ *pb_testproto.PingRequest) (*pb_testproto.Empty, error) {
	s.recordedContext = ctx
	return &pb_testproto.Empty{}, nil
}

func (s *pingService) PingList(req *pb_testproto.PingRequest, stream pb_testproto.TestService_PingListServer) error {
	s.recordedContext = stream.Context()
	return stream.Send(&pb_testproto.PingResponse{})
}

func (s *pingService) PingStream(stream pb_testproto.TestService_PingStreamServer) error {
	return nil
}

// tempSocketName provides a temporary Unix socket name for testing.
func tempSocketName(t *testing.T) string {
	tmpfile, err := ioutil.TempFile("", "sock")
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())
	socket := tmpfile.Name()
	require.NoError(t, os.Remove(socket))
	return socket
}

type mockHost struct {
	component.Host
	ext map[config.ComponentID]component.Extension
}

func (nh *mockHost) GetExtensions() map[config.ComponentID]component.Extension {
	return nh.ext
}
