// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
	"go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

var (
	_ extension.Extension  = (*mockAuthServer)(nil)
	_ extensionauth.Server = (*mockAuthServer)(nil)
)

type mockAuthServer struct {
	component.StartFunc
	component.ShutdownFunc
	extensionauth.ServerAuthenticateFunc
}

func newMockAuthServer(auth func(ctx context.Context, sources map[string][]string) (context.Context, error)) extensionauth.Server {
	return &mockAuthServer{ServerAuthenticateFunc: auth}
}

func TestNewDefaultKeepaliveClientConfig(t *testing.T) {
	expectedKeepaliveClientConfig := KeepaliveClientConfig{
		Time:    time.Second * 10,
		Timeout: time.Second * 10,
	}
	keepaliveClientConfig := NewDefaultKeepaliveClientConfig()
	assert.Equal(t, expectedKeepaliveClientConfig, keepaliveClientConfig)
}

func TestNewDefaultClientConfig(t *testing.T) {
	keepalive := NewDefaultKeepaliveClientConfig()
	expected := ClientConfig{
		TLS:          configtls.NewDefaultClientConfig(),
		Keepalive:    configoptional.Some(keepalive),
		BalancerName: BalancerName(),
	}

	result := NewDefaultClientConfig()

	assert.Equal(t, expected, result)
}

func TestNewDefaultKeepaliveServerParameters(t *testing.T) {
	expectedParams := KeepaliveServerParameters{}
	params := NewDefaultKeepaliveServerParameters()

	assert.Equal(t, expectedParams, params)
}

func TestNewDefaultKeepaliveEnforcementPolicy(t *testing.T) {
	expectedPolicy := KeepaliveEnforcementPolicy{}

	policy := NewDefaultKeepaliveEnforcementPolicy()

	assert.Equal(t, expectedPolicy, policy)
}

func TestNewDefaultKeepaliveServerConfig(t *testing.T) {
	expected := KeepaliveServerConfig{
		ServerParameters:  configoptional.Some(NewDefaultKeepaliveServerParameters()),
		EnforcementPolicy: configoptional.Some(NewDefaultKeepaliveEnforcementPolicy()),
	}
	result := NewDefaultKeepaliveServerConfig()
	assert.Equal(t, expected, result)
}

func TestNewDefaultServerConfig(t *testing.T) {
	expected := ServerConfig{
		Keepalive: configoptional.Some(NewDefaultKeepaliveServerConfig()),
		NetAddr: confignet.AddrConfig{
			Transport: confignet.TransportTypeTCP,
		},
	}

	result := NewDefaultServerConfig()

	assert.Equal(t, expected, result)
}

var (
	testAuthID    = component.MustNewID("testauth")
	mockID        = component.MustNewID("mock")
	doesntExistID = component.MustNewID("doesntexist")
)

func TestDefaultGrpcClientSettings(t *testing.T) {
	cc := &ClientConfig{
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	}
	opts, err := cc.getGrpcDialOptions(context.Background(), nil, componenttest.NewNopTelemetrySettings(), []ToClientConnOption{})
	require.NoError(t, err)
	/* Expecting 2 DialOptions:
	 * - WithTransportCredentials (TLS)
	 * - WithStatsHandler (always, for self-telemetry)
	 */
	assert.Len(t, opts, 2)
}

func TestGrpcClientExtraOption(t *testing.T) {
	cc := &ClientConfig{
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	}
	extraOpt := grpc.WithUserAgent("test-agent")
	opts, err := cc.getGrpcDialOptions(
		context.Background(),
		nil,
		componenttest.NewNopTelemetrySettings(),
		[]ToClientConnOption{WithGrpcDialOption(extraOpt)},
	)
	require.NoError(t, err)
	/* Expecting 3 DialOptions:
	 * - WithTransportCredentials (TLS)
	 * - WithStatsHandler (always, for self-telemetry)
	 * - extraOpt
	 */
	assert.Len(t, opts, 3)
	assert.Equal(t, opts[2], extraOpt)
}

func TestAllGrpcClientSettings(t *testing.T) {
	tests := []struct {
		settings   ClientConfig
		name       string
		extensions map[component.ID]component.Component
	}{
		{
			name: "test all with gzip compression",
			settings: ClientConfig{
				Headers: configopaque.MapList{
					{Name: "test", Value: "test"},
				},
				Endpoint:    "localhost:1234",
				Compression: configcompression.TypeGzip,
				TLS: configtls.ClientConfig{
					Insecure: false,
				},
				Keepalive: configoptional.Some(KeepaliveClientConfig{
					Time:                time.Second,
					Timeout:             time.Second,
					PermitWithoutStream: true,
				}),
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				WaitForReady:    true,
				BalancerName:    "round_robin",
				Authority:       "pseudo-authority",
				Auth:            configoptional.Some(configauth.Config{AuthenticatorID: testAuthID}),
			},
			extensions: map[component.ID]component.Component{
				testAuthID: extensionauthtest.NewNopClient(),
			},
		},
		{
			name: "test all with snappy compression",
			settings: ClientConfig{
				Headers: configopaque.MapList{
					{Name: "test", Value: "test"},
				},
				Endpoint:    "localhost:1234",
				Compression: configcompression.TypeSnappy,
				TLS: configtls.ClientConfig{
					Insecure: false,
				},
				Keepalive: configoptional.Some(KeepaliveClientConfig{
					Time:                time.Second,
					Timeout:             time.Second,
					PermitWithoutStream: true,
				}),
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				WaitForReady:    true,
				BalancerName:    "round_robin",
				Authority:       "pseudo-authority",
				Auth:            configoptional.Some(configauth.Config{AuthenticatorID: testAuthID}),
			},
			extensions: map[component.ID]component.Component{
				testAuthID: extensionauthtest.NewNopClient(),
			},
		},
		{
			name: "test all with zstd compression",
			settings: ClientConfig{
				Headers: configopaque.MapList{
					{Name: "test", Value: "test"},
				},
				Endpoint:    "localhost:1234",
				Compression: configcompression.TypeZstd,
				TLS: configtls.ClientConfig{
					Insecure: false,
				},
				Keepalive: configoptional.Some(KeepaliveClientConfig{
					Time:                time.Second,
					Timeout:             time.Second,
					PermitWithoutStream: true,
				}),
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				WaitForReady:    true,
				BalancerName:    "round_robin",
				Authority:       "pseudo-authority",
				Auth:            configoptional.Some(configauth.Config{AuthenticatorID: testAuthID}),
			},
			extensions: map[component.ID]component.Component{
				testAuthID: extensionauthtest.NewNopClient(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts, err := test.settings.getGrpcDialOptions(context.Background(), test.extensions, componenttest.NewNopTelemetrySettings(), []ToClientConnOption{})
			require.NoError(t, err)
			/* Expecting 11 DialOptions:
			 * - WithDefaultCallOptions (Compression)
			 * - WithTransportCredentials (TLS)
			 * - WithDefaultServiceConfig (BalancerName)
			 * - WithAuthority (Authority)
			 * - WithStatsHandler (always, for self-telemetry)
			 * - WithReadBufferSize (ReadBufferSize)
			 * - WithWriteBufferSize (WriteBufferSize)
			 * - WithKeepaliveParams (Keepalive)
			 * - WithPerRPCCredentials (Auth)
			 * - WithUnaryInterceptor/WithStreamInterceptor (Headers)
			 */
			assert.Len(t, opts, 11)
		})
	}
}

func TestSanitizeEndpoint(t *testing.T) {
	cfg := NewDefaultClientConfig()
	cfg.Endpoint = "dns://authority/backend.example.com:4317"
	assert.Equal(t, "authority/backend.example.com:4317", cfg.sanitizedEndpoint())
	cfg.Endpoint = "dns:///backend.example.com:4317"
	assert.Equal(t, "backend.example.com:4317", cfg.sanitizedEndpoint())
	cfg.Endpoint = "dns:////backend.example.com:4317"
	assert.Equal(t, "/backend.example.com:4317", cfg.sanitizedEndpoint())
}

func TestValidateEndpoint(t *testing.T) {
	cfg := NewDefaultClientConfig()
	cfg.Endpoint = "dns://authority/backend.example.com:4317"
	assert.NoError(t, cfg.Validate())
	cfg.Endpoint = "unix:///my/unix/socket.sock"
	assert.NoError(t, cfg.Validate())
}

func TestHeaders(t *testing.T) {
	traceServer := &grpcTraceServer{}
	server, addr := traceServer.startTestServer(t, configoptional.Some(ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "localhost:0",
			Transport: confignet.TransportTypeTCP,
		},
	}))
	defer server.Stop()

	// Create client and send request to server with headers
	resp, errResp := sendTestRequest(t, ClientConfig{
		Endpoint: addr,
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
		Headers: configopaque.MapList{
			{Name: "testheader", Value: "testvalue"},
		},
	})
	require.NoError(t, errResp)
	assert.NotNil(t, resp)

	// Check received headers
	md, ok := metadata.FromIncomingContext(traceServer.recordedContext)
	require.True(t, ok)
	assert.Equal(t, []string{"testvalue"}, md.Get("testheader"))
}

func TestDefaultGrpcServerSettings(t *testing.T) {
	gss := &ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint: "0.0.0.0:1234",
		},
	}
	opts, err := gss.getGrpcServerOptions(context.Background(), nil, componenttest.NewNopTelemetrySettings(), []ToServerOption{})
	require.NoError(t, err)
	assert.Len(t, opts, 3)
}

func TestGrpcServerExtraOption(t *testing.T) {
	gss := &ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint: "0.0.0.0:1234",
		},
	}
	extraOpt := grpc.ConnectionTimeout(1_000_000_000)
	opts, err := gss.getGrpcServerOptions(
		context.Background(),
		nil,
		componenttest.NewNopTelemetrySettings(),
		[]ToServerOption{WithGrpcServerOption(extraOpt)},
	)
	require.NoError(t, err)
	assert.Len(t, opts, 4)
	assert.Equal(t, opts[3], extraOpt)
}

func TestGrpcServerValidate(t *testing.T) {
	tests := []struct {
		gss *ServerConfig
		err string
	}{
		{
			gss: &ServerConfig{
				MaxRecvMsgSizeMiB: -1,
				NetAddr: confignet.AddrConfig{
					Endpoint: "0.0.0.0:1234",
				},
			},
			err: "invalid max_recv_msg_size_mib value",
		},
		{
			gss: &ServerConfig{
				MaxRecvMsgSizeMiB: 9223372036854775807,
				NetAddr: confignet.AddrConfig{
					Endpoint: "0.0.0.0:1234",
				},
			},
			err: "invalid max_recv_msg_size_mib value",
		},
		{
			gss: &ServerConfig{
				ReadBufferSize: -1,
				NetAddr: confignet.AddrConfig{
					Endpoint: "0.0.0.0:1234",
				},
			},
			err: "invalid read_buffer_size value",
		},
		{
			gss: &ServerConfig{
				WriteBufferSize: -1,
				NetAddr: confignet.AddrConfig{
					Endpoint: "0.0.0.0:1234",
				},
			},
			err: "invalid write_buffer_size value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.err, func(t *testing.T) {
			err := tt.gss.Validate()
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.err)
		})
	}
}

func TestAllGrpcServerSettingsExceptAuth(t *testing.T) {
	gss := &ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "localhost:1234",
			Transport: confignet.TransportTypeTCP,
		},
		TLS: configoptional.Some(configtls.ServerConfig{
			Config:       configtls.Config{},
			ClientCAFile: "",
		}),
		MaxRecvMsgSizeMiB:    1,
		MaxConcurrentStreams: 1024,
		ReadBufferSize:       1024,
		WriteBufferSize:      1024,
		Keepalive: configoptional.Some(KeepaliveServerConfig{
			ServerParameters: configoptional.Some(KeepaliveServerParameters{
				MaxConnectionIdle:     time.Second,
				MaxConnectionAge:      time.Second,
				MaxConnectionAgeGrace: time.Second,
				Time:                  time.Second,
				Timeout:               time.Second,
			}),
			EnforcementPolicy: configoptional.Some(KeepaliveEnforcementPolicy{
				MinTime:             time.Second,
				PermitWithoutStream: true,
			}),
		}),
	}
	opts, err := gss.getGrpcServerOptions(context.Background(), nil, componenttest.NewNopTelemetrySettings(), []ToServerOption{})
	require.NoError(t, err)
	assert.Len(t, opts, 10)
}

func TestGrpcServerAuthSettings(t *testing.T) {
	gss := &ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint: "0.0.0.0:1234",
		},
	}
	gss.Auth = configoptional.Some(configauth.Config{
		AuthenticatorID: mockID,
	})

	extensions := map[component.ID]component.Component{
		mockID: extensionauthtest.NewNopServer(),
	}
	srv, err := gss.ToServer(context.Background(), extensions, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	assert.NotNil(t, srv)
}

func TestGrpcClientConfigInvalidBalancer(t *testing.T) {
	settings := ClientConfig{
		Headers: configopaque.MapList{
			{Name: "test", Value: "test"},
		},
		Endpoint:    "localhost:1234",
		Compression: "gzip",
		TLS: configtls.ClientConfig{
			Insecure: false,
		},
		Keepalive: configoptional.Some(KeepaliveClientConfig{
			Time:                time.Second,
			Timeout:             time.Second,
			PermitWithoutStream: true,
		}),
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		WaitForReady:    true,
		BalancerName:    "test",
	}
	assert.ErrorContains(t, settings.Validate(), "invalid balancer_name: test")
}

func TestGRPCClientSettingsError(t *testing.T) {
	tests := []struct {
		settings   ClientConfig
		err        string
		extensions map[component.ID]component.Component
	}{
		{
			err: "failed to load TLS config: failed to load CA CertPool File: failed to load cert /doesnt/exist:",
			settings: ClientConfig{
				Headers:     nil,
				Endpoint:    "localhost:1234",
				Compression: "",
				TLS: configtls.ClientConfig{
					Config: configtls.Config{
						CAFile: "/doesnt/exist",
					},
					Insecure:   false,
					ServerName: "",
				},
			},
		},
		{
			err: "failed to load TLS config: failed to load TLS cert and key: for auth via TLS, provide both certificate and key, or neither",
			settings: ClientConfig{
				Headers:     nil,
				Endpoint:    "localhost:1234",
				Compression: "",
				TLS: configtls.ClientConfig{
					Config: configtls.Config{
						CertFile: "/doesnt/exist",
					},
					Insecure:   false,
					ServerName: "",
				},
			},
		},
		{
			err: "failed to resolve authenticator \"doesntexist\": authenticator not found",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: doesntExistID}),
			},
			extensions: map[component.ID]component.Component{},
		},
		{
			err: "authentication was configured but this component or its host does not support extensions",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: doesntExistID}),
			},
		},
		{
			err: "unsupported compression type \"zlib\"",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: true,
				},
				Compression: "zlib",
			},
		},
		{
			err: "unsupported compression type \"deflate\"",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: true,
				},
				Compression: "deflate",
			},
		},
		{
			err: "unsupported compression type \"bad\"",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: true,
				},
				Compression: "bad",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			require.NoError(t, test.settings.Validate())
			_, err := test.settings.ToClientConn(context.Background(), test.extensions, componenttest.NewNopTelemetrySettings())
			require.Error(t, err)
			assert.ErrorContains(t, err, test.err)
		})
	}
}

func TestUseSecure(t *testing.T) {
	cc := &ClientConfig{
		Headers:     nil,
		Endpoint:    "",
		Compression: "",
		TLS:         configtls.ClientConfig{},
	}
	dialOpts, err := cc.getGrpcDialOptions(context.Background(), nil, componenttest.NewNopTelemetrySettings(), []ToClientConnOption{})
	require.NoError(t, err)
	assert.Len(t, dialOpts, 2)
}

func TestGRPCServerSettingsError(t *testing.T) {
	tests := []struct {
		settings ServerConfig
		err      string
	}{
		{
			err: "failed to load TLS config: failed to load CA CertPool File: failed to load cert /doesnt/exist:",
			settings: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "127.0.0.1:1234",
					Transport: confignet.TransportTypeTCP,
				},
				TLS: configoptional.Some(configtls.ServerConfig{
					Config: configtls.Config{
						CAFile: "/doesnt/exist",
					},
				}),
			},
		},
		{
			err: "failed to load TLS config: failed to load TLS cert and key: for auth via TLS, provide both certificate and key, or neither",
			settings: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "127.0.0.1:1234",
					Transport: confignet.TransportTypeTCP,
				},
				TLS: configoptional.Some(configtls.ServerConfig{
					Config: configtls.Config{
						CertFile: "/doesnt/exist",
					},
				}),
			},
		},
		{
			err: "failed to load client CA CertPool: failed to load CA /doesnt/exist:",
			settings: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "127.0.0.1:1234",
					Transport: confignet.TransportTypeTCP,
				},
				TLS: configoptional.Some(configtls.ServerConfig{
					ClientCAFile: "/doesnt/exist",
				}),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			_, err := test.settings.ToServer(context.Background(), nil, componenttest.NewNopTelemetrySettings())
			assert.ErrorContains(t, err, test.err)
		})
	}
}

func TestGRPCServerSettings_ToListener_Error(t *testing.T) {
	settings := ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "127.0.0.1:1234567",
			Transport: confignet.TransportTypeTCP,
		},
	}
	_, err := settings.NetAddr.Listen(context.Background())
	assert.Error(t, err)
}

func TestHttpReception(t *testing.T) {
	tests := []struct {
		name           string
		tlsServerCreds configoptional.Optional[configtls.ServerConfig]
		tlsClientCreds configoptional.Optional[configtls.ClientConfig]
		hasError       bool
	}{
		{
			name:           "noTLS",
			tlsServerCreds: configoptional.None[configtls.ServerConfig](),
			tlsClientCreds: configoptional.Some(configtls.ClientConfig{
				Insecure: true,
			}),
		},
		{
			name: "TLS",
			tlsServerCreds: configoptional.Some(configtls.ServerConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
			}),
			tlsClientCreds: configoptional.Some(configtls.ClientConfig{
				Config: configtls.Config{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			}),
		},
		{
			name: "NoServerCertificates",
			tlsServerCreds: configoptional.Some(configtls.ServerConfig{
				Config: configtls.Config{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
			}),
			tlsClientCreds: configoptional.Some(configtls.ClientConfig{
				Config: configtls.Config{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			}),
			hasError: true,
		},
		{
			name: "mTLS",
			tlsServerCreds: configoptional.Some(configtls.ServerConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "ca.crt"),
			}),
			tlsClientCreds: configoptional.Some(configtls.ClientConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "client.crt"),
					KeyFile:  filepath.Join("testdata", "client.key"),
				},
				ServerName: "localhost",
			}),
		},
		{
			name: "NoClientCertificate",
			tlsServerCreds: configoptional.Some(configtls.ServerConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "ca.crt"),
			}),
			tlsClientCreds: configoptional.Some(configtls.ClientConfig{
				Config: configtls.Config{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			}),
			hasError: true,
		},
		{
			name: "WrongClientCA",
			tlsServerCreds: configoptional.Some(configtls.ServerConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "server.crt"),
			}),
			tlsClientCreds: configoptional.Some(configtls.ClientConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "client.crt"),
					KeyFile:  filepath.Join("testdata", "client.key"),
				},
				ServerName: "localhost",
			}),
			hasError: true,
		},
	}
	// prepare

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, addr := (&grpcTraceServer{}).startTestServer(t, configoptional.Some(ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				TLS: test.tlsServerCreds,
			}))
			defer s.Stop()

			resp, errResp := sendTestRequest(t, ClientConfig{
				Endpoint: addr,
				TLS:      *test.tlsClientCreds.Get(),
			})
			if test.hasError {
				require.Error(t, errResp)
			} else {
				require.NoError(t, errResp)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestReceiveOnUnixDomainSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on windows")
	}

	socketName := tempSocketName(t)
	srv, addr := (&grpcTraceServer{}).startTestServer(t, configoptional.Some(ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  socketName,
			Transport: confignet.TransportTypeUnix,
		},
	}))
	defer srv.Stop()

	resp, errResp := sendTestRequest(t, ClientConfig{
		Endpoint: "unix://" + addr,
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	})
	require.NoError(t, errResp)
	assert.NotNil(t, resp)
}

func TestContextWithClient(t *testing.T) {
	testCases := []struct {
		desc       string
		input      context.Context
		doMetadata bool
		expected   client.Info
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
		{
			desc: "existing client with metadata",
			input: client.NewContext(context.Background(), client.Info{
				Metadata: client.NewMetadata(map[string][]string{"test-metadata-key": {"test-value"}}),
			}),
			doMetadata: true,
			expected: client.Info{
				Metadata: client.NewMetadata(map[string][]string{"test-metadata-key": {"test-value"}}),
			},
		},
		{
			desc: "existing client with metadata in context",
			input: metadata.NewIncomingContext(
				client.NewContext(context.Background(), client.Info{}),
				metadata.Pairs("test-metadata-key", "test-value"),
			),
			doMetadata: true,
			expected: client.Info{
				Metadata: client.NewMetadata(map[string][]string{"test-metadata-key": {"test-value"}}),
			},
		},
		{
			desc: "existing client with metadata in context, no metadata processing",
			input: metadata.NewIncomingContext(
				client.NewContext(context.Background(), client.Info{}),
				metadata.Pairs("test-metadata-key", "test-value"),
			),
			expected: client.Info{},
		},
		{
			desc: "existing client with Host and metadata",
			input: metadata.NewIncomingContext(
				client.NewContext(context.Background(), client.Info{}),
				metadata.Pairs("test-metadata-key", "test-value", ":authority", "localhost:55443"),
			),
			doMetadata: true,
			expected: client.Info{
				Metadata: client.NewMetadata(map[string][]string{"test-metadata-key": {"test-value"}, ":authority": {"localhost:55443"}, "Host": {"localhost:55443"}}),
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			cl := client.FromContext(contextWithClient(tt.input, tt.doMetadata))
			assert.Equal(t, tt.expected, cl)
		})
	}
}

func TestStreamInterceptorEnhancesClient(t *testing.T) {
	// prepare
	inCtx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.IPAddr{IP: net.IPv4(1, 1, 1, 1)},
	})

	stream := &mockedStream{
		ctx: inCtx,
	}

	var handlerCalled bool
	handler := func(_ any, stream grpc.ServerStream) error {
		handlerCalled = true
		cl := client.FromContext(stream.Context())
		assert.Equal(t, "1.1.1.1", cl.Addr.String())
		return nil
	}

	// test
	err := enhanceStreamWithClientInformation(false)(nil, stream, nil, handler)

	// verify
	require.NoError(t, err)
	assert.True(t, handlerCalled, "the handler should have been called")
}

type mockedStream struct {
	ctx context.Context
	grpc.ServerStream
}

func (ms *mockedStream) Context() context.Context {
	return ms.ctx
}

func TestClientInfoInterceptors(t *testing.T) {
	testCases := []struct {
		name   string
		tester func(ptraceotlp.ExportResponse, error)
	}{
		{
			// we only have unary services, we don't have any clients we could use
			// to test with streaming services
			name: "unary",
			tester: func(resp ptraceotlp.ExportResponse, errResp error) {
				require.NoError(t, errResp)
				require.NotNil(t, resp)
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &grpcTraceServer{}
			var addr string

			// prepare the server
			{
				var srv *grpc.Server
				srv, addr = mock.startTestServer(t, configoptional.Some(ServerConfig{
					NetAddr: confignet.AddrConfig{
						Endpoint:  "localhost:0",
						Transport: confignet.TransportTypeTCP,
					},
				}))
				defer srv.Stop()
			}

			// prepare the client and execute a RPC
			{
				resp, errResp := sendTestRequest(t, ClientConfig{
					Endpoint: addr,
					TLS: configtls.ClientConfig{
						Insecure: true,
					},
				})

				// test
				tt.tester(resp, errResp)
			}

			// verify
			cl := client.FromContext(mock.recordedContext)

			// the client address is something like 127.0.0.1:41086
			assert.Contains(t, cl.Addr.String(), "127.0.0.1")
		})
	}
}

func TestDefaultUnaryInterceptorAuthSucceeded(t *testing.T) {
	// prepare
	handlerCalled := false
	authCalled := false
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		authCalled = true
		ctx := client.NewContext(context.Background(), client.Info{
			Addr: &net.IPAddr{IP: net.IPv4(1, 2, 3, 4)},
		})

		return ctx, nil
	}
	handler := func(ctx context.Context, _ any) (any, error) {
		handlerCalled = true
		cl := client.FromContext(ctx)
		assert.Equal(t, "1.2.3.4", cl.Addr.String())
		return nil, nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))
	interceptor := authUnaryServerInterceptor(newMockAuthServer(authFunc))

	// test
	res, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	// verify
	assert.Nil(t, res)
	require.NoError(t, err)
	assert.True(t, authCalled)
	assert.True(t, handlerCalled)
}

func TestDefaultUnaryInterceptorAuthFailure(t *testing.T) {
	// prepare
	authCalled := false
	expectedErr := errors.New("not authenticated")
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		authCalled = true
		return context.Background(), expectedErr
	}
	handler := func(context.Context, any) (any, error) {
		assert.FailNow(t, "the handler should not have been called on auth failure!")
		return nil, nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))
	interceptor := authUnaryServerInterceptor(newMockAuthServer(authFunc))

	// test
	res, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	// verify
	assert.Nil(t, res)
	require.ErrorContains(t, err, expectedErr.Error())
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.True(t, authCalled)
}

func TestDefaultUnaryInterceptorAuthFailureWithStatusErr(t *testing.T) {
	// prepare
	authCalled := false
	expectedStatusErr := status.New(codes.Unavailable, "unavailable")
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		authCalled = true
		return context.Background(), expectedStatusErr.Err()
	}
	handler := func(context.Context, any) (any, error) {
		assert.FailNow(t, "the handler should not have been called on auth failure!")
		return nil, nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))
	interceptor := authUnaryServerInterceptor(newMockAuthServer(authFunc))

	// test
	res, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	// verify
	assert.Nil(t, res)
	require.ErrorContains(t, err, expectedStatusErr.Err().Error())
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.True(t, authCalled)
}

func TestDefaultUnaryInterceptorMissingMetadata(t *testing.T) {
	// prepare
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		assert.FailNow(t, "the auth func should not have been called!")
		return context.Background(), nil
	}
	handler := func(context.Context, any) (any, error) {
		assert.FailNow(t, "the handler should not have been called!")
		return nil, nil
	}
	interceptor := authUnaryServerInterceptor(newMockAuthServer(authFunc))

	// test
	res, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)

	// verify
	assert.Nil(t, res)
	assert.Equal(t, errMetadataNotFound, err)
}

func TestDefaultStreamInterceptorAuthSucceeded(t *testing.T) {
	// prepare
	handlerCalled := false
	authCalled := false
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		authCalled = true
		ctx := client.NewContext(context.Background(), client.Info{
			Addr: &net.IPAddr{IP: net.IPv4(1, 2, 3, 4)},
		})
		return ctx, nil
	}
	handler := func(_ any, stream grpc.ServerStream) error {
		// ensure that the client information is propagated down to the underlying stream
		cl := client.FromContext(stream.Context())
		assert.Equal(t, "1.2.3.4", cl.Addr.String())
		handlerCalled = true
		return nil
	}
	streamServer := &mockServerStream{
		ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data")),
	}
	interceptor := authStreamServerInterceptor(newMockAuthServer(authFunc))

	// test
	err := interceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler)

	// verify
	require.NoError(t, err)
	assert.True(t, authCalled)
	assert.True(t, handlerCalled)
}

func TestDefaultStreamInterceptorAuthFailureWithStatusErr(t *testing.T) {
	// prepare
	authCalled := false
	expectedStatusErr := status.New(codes.Unavailable, "unavailable")
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		authCalled = true
		return context.Background(), expectedStatusErr.Err()
	}
	handler := func(any, grpc.ServerStream) error {
		assert.FailNow(t, "the handler should not have been called on auth failure!")
		return nil
	}
	streamServer := &mockServerStream{
		ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data")),
	}
	interceptor := authStreamServerInterceptor(newMockAuthServer(authFunc))

	// test
	err := interceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler)

	// verify
	require.ErrorContains(t, err, expectedStatusErr.Err().Error()) // unfortunately, grpc errors don't wrap the original ones
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.True(t, authCalled)
}

func TestDefaultStreamInterceptorAuthFailure(t *testing.T) {
	// prepare
	authCalled := false
	expectedErr := errors.New("not authenticated")
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		authCalled = true
		return context.Background(), expectedErr
	}
	handler := func(any, grpc.ServerStream) error {
		assert.FailNow(t, "the handler should not have been called on auth failure!")
		return nil
	}
	streamServer := &mockServerStream{
		ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data")),
	}
	interceptor := authStreamServerInterceptor(newMockAuthServer(authFunc))

	// test
	err := interceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler)

	// verify
	require.ErrorContains(t, err, expectedErr.Error()) // unfortunately, grpc errors don't wrap the original ones
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.True(t, authCalled)
}

func TestDefaultStreamInterceptorMissingMetadata(t *testing.T) {
	// prepare
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		assert.FailNow(t, "the auth func should not have been called!")
		return context.Background(), nil
	}
	handler := func(any, grpc.ServerStream) error {
		assert.FailNow(t, "the handler should not have been called!")
		return nil
	}
	streamServer := &mockServerStream{
		ctx: context.Background(),
	}
	interceptor := authStreamServerInterceptor(newMockAuthServer(authFunc))

	// test
	err := interceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler)

	// verify
	assert.Equal(t, errMetadataNotFound, err)
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

type grpcTraceServer struct {
	ptraceotlp.UnimplementedGRPCServer
	recordedContext context.Context
}

func (gts *grpcTraceServer) Export(ctx context.Context, _ ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	gts.recordedContext = ctx
	return ptraceotlp.NewExportResponse(), nil
}

func (gts *grpcTraceServer) startTestServer(t *testing.T, gss configoptional.Optional[ServerConfig]) (*grpc.Server, string) {
	return gts.startTestServerWithExtensions(t, gss, nil)
}

func (gts *grpcTraceServer) startTestServerWithExtensions(t *testing.T, gss configoptional.Optional[ServerConfig], extensions map[component.ID]component.Component, opts ...ToServerOption) (*grpc.Server, string) {
	listener, err := gss.Get().NetAddr.Listen(context.Background())
	require.NoError(t, err)
	server, err := gss.Get().ToServer(context.Background(), extensions, componenttest.NewNopTelemetrySettings(), opts...)
	require.NoError(t, err)
	ptraceotlp.RegisterGRPCServer(server, gts)
	go func() {
		_ = server.Serve(listener)
	}()
	return server, listener.Addr().String()
}

func (gts *grpcTraceServer) startTestServerWithExtensionsError(_ *testing.T, gss ServerConfig, extensions map[component.ID]component.Component, opts ...ToServerOption) (*grpc.Server, error) {
	listener, err := gss.NetAddr.Listen(context.Background())
	if err != nil {
		return nil, err
	}
	defer listener.Close()

	server, err := gss.ToServer(context.Background(), extensions, componenttest.NewNopTelemetrySettings(), opts...)
	if err != nil {
		return nil, err
	}

	ptraceotlp.RegisterGRPCServer(server, gts)
	return server, nil
}

// sendTestRequest issues a ptraceotlp export request and captures metadata.
func sendTestRequest(t *testing.T, cc ClientConfig) (ptraceotlp.ExportResponse, error) {
	return sendTestRequestWithExtensions(t, cc, nil)
}

// sendTestRequestWithExtensions is similar to sendTestRequest but allows specifying the host
func sendTestRequestWithExtensions(t *testing.T, cc ClientConfig, extensions map[component.ID]component.Component) (ptraceotlp.ExportResponse, error) {
	grpcClientConn, errClient := cc.ToClientConn(context.Background(), extensions, componenttest.NewNopTelemetrySettings())
	require.NoError(t, errClient)
	defer func() { assert.NoError(t, grpcClientConn.Close()) }()
	c := ptraceotlp.NewGRPCClient(grpcClientConn)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
	resp, errResp := c.Export(ctx, ptraceotlp.NewExportRequest(), grpc.WaitForReady(true))
	cancelFunc()
	return resp, errResp
}

// tempSocketName provides a temporary Unix socket name for testing.
func tempSocketName(t *testing.T) string {
	// The socket path length limit on macOS is 104 characters. Using `os.TempDir` to produce a shorter file path (#12639)
	tmpfile, err := os.CreateTemp(os.TempDir(), "sock")
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())
	socket := tmpfile.Name()
	require.NoError(t, os.Remove(socket))
	return socket
}
