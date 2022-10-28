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
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
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

	tests := []struct {
		settings GRPCClientSettings
		name     string
		host     component.Host
	}{
		{
			name: "test all with gzip compression",
			settings: GRPCClientSettings{
				Headers: map[string]string{
					"test": "test",
				},
				Endpoint:    "localhost:1234",
				Compression: configcompression.Gzip,
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
			},
			host: &mockHost{
				ext: map[config.ComponentID]component.Extension{
					config.NewComponentID("testauth"): &configauth.MockClientAuthenticator{},
				},
			},
		},
		{
			name: "test all with snappy compression",
			settings: GRPCClientSettings{
				Headers: map[string]string{
					"test": "test",
				},
				Endpoint:    "localhost:1234",
				Compression: configcompression.Snappy,
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
			},
			host: &mockHost{
				ext: map[config.ComponentID]component.Extension{
					config.NewComponentID("testauth"): &configauth.MockClientAuthenticator{},
				},
			},
		},
		{
			name: "test all with zstd compression",
			settings: GRPCClientSettings{
				Headers: map[string]string{
					"test": "test",
				},
				Endpoint:    "localhost:1234",
				Compression: configcompression.Zstd,
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
			},
			host: &mockHost{
				ext: map[config.ComponentID]component.Extension{
					config.NewComponentID("testauth"): &configauth.MockClientAuthenticator{},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts, err := test.settings.ToDialOptions(test.host, tt.TelemetrySettings)
			assert.NoError(t, err)
			assert.Len(t, opts, 9)
		})
	}
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
			config.NewComponentID("mock"): configauth.NewServerAuthenticator(),
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
		{
			err: "unsupported compression type \"zlib\"",
			settings: GRPCClientSettings{
				Endpoint: "localhost:1234",
				TLSSetting: configtls.TLSClientSetting{
					Insecure: true,
				},
				Compression: "zlib",
			},
			host: &mockHost{},
		},
		{
			err: "unsupported compression type \"deflate\"",
			settings: GRPCClientSettings{
				Endpoint: "localhost:1234",
				TLSSetting: configtls.TLSClientSetting{
					Insecure: true,
				},
				Compression: "deflate",
			},
			host: &mockHost{},
		},
		{
			err: "unsupported compression type \"bad\"",
			settings: GRPCClientSettings{
				Endpoint: "localhost:1234",
				TLSSetting: configtls.TLSClientSetting{
					Insecure: true,
				},
				Compression: "bad",
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
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
		},
		{
			name: "NoServerCertificates",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
		{
			name: "mTLS",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "ca.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "client.crt"),
					KeyFile:  filepath.Join("testdata", "client.key"),
				},
				ServerName: "localhost",
			},
		},
		{
			name: "NoClientCertificate",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "ca.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
		{
			name: "WrongClientCA",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "server.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "client.crt"),
					KeyFile:  filepath.Join("testdata", "client.key"),
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
			ptraceotlp.RegisterGRPCServer(s, &grpcTraceServer{})

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
			client := ptraceotlp.NewGRPCClient(grpcClientConn)
			ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
			resp, errResp := client.Export(ctx, ptraceotlp.NewExportRequest(), grpc.WaitForReady(true))
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
	ptraceotlp.RegisterGRPCServer(s, &grpcTraceServer{})

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
	client := ptraceotlp.NewGRPCClient(grpcClientConn)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
	resp, errResp := client.Export(ctx, ptraceotlp.NewExportRequest(), grpc.WaitForReady(true))
	assert.NoError(t, errResp)
	assert.NotNil(t, resp)
	cancelFunc()
	s.Stop()
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
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cl := client.FromContext(contextWithClient(tC.input, tC.doMetadata))
			assert.Equal(t, tC.expected, cl)
		})
	}
}

func TestStreamInterceptorEnhancesClient(t *testing.T) {
	// prepare
	inCtx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.IPAddr{IP: net.IPv4(1, 1, 1, 1)},
	})
	var outContext context.Context

	stream := &mockedStream{
		ctx: inCtx,
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		outContext = stream.Context()
		return nil
	}

	// test
	err := enhanceStreamWithClientInformation(false)(nil, stream, nil, handler)

	// verify
	assert.NoError(t, err)

	cl := client.FromContext(outContext)
	assert.Equal(t, "1.1.1.1", cl.Addr.String())
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
		desc   string
		tester func(context.Context, ptraceotlp.GRPCClient)
	}{
		{
			// we only have unary services, we don't have any clients we could use
			// to test with streaming services
			desc: "unary",
			tester: func(ctx context.Context, cl ptraceotlp.GRPCClient) {
				resp, errResp := cl.Export(ctx, ptraceotlp.NewExportRequest())
				require.NoError(t, errResp)
				require.NotNil(t, resp)
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			mock := &grpcTraceServer{}
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
				ptraceotlp.RegisterGRPCServer(srv, mock)

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

				cl := ptraceotlp.NewGRPCClient(grpcClientConn)
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
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		cl := client.FromContext(ctx)
		assert.Equal(t, "1.2.3.4", cl.Addr.String())
		return nil, nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))

	// test
	res, err := authUnaryServerInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler, authFunc)

	// verify
	assert.Nil(t, res)
	assert.NoError(t, err)
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
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		assert.FailNow(t, "the handler should not have been called on auth failure!")
		return nil, nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))

	// test
	res, err := authUnaryServerInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler, authFunc)

	// verify
	assert.Nil(t, res)
	assert.Equal(t, expectedErr, err)
	assert.True(t, authCalled)
}

func TestDefaultUnaryInterceptorMissingMetadata(t *testing.T) {
	// prepare
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		assert.FailNow(t, "the auth func should not have been called!")
		return context.Background(), nil
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		assert.FailNow(t, "the handler should not have been called!")
		return nil, nil
	}

	// test
	res, err := authUnaryServerInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler, authFunc)

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
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		// ensure that the client information is propagated down to the underlying stream
		cl := client.FromContext(stream.Context())
		assert.Equal(t, "1.2.3.4", cl.Addr.String())
		handlerCalled = true
		return nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))
	streamServer := &mockServerStream{
		ctx: ctx,
	}

	// test
	err := authStreamServerInterceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler, authFunc)

	// verify
	assert.NoError(t, err)
	assert.True(t, authCalled)
	assert.True(t, handlerCalled)
}

func TestDefaultStreamInterceptorAuthFailure(t *testing.T) {
	// prepare
	authCalled := false
	expectedErr := errors.New("not authenticated")
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		authCalled = true
		return context.Background(), expectedErr
	}
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		assert.FailNow(t, "the handler should not have been called on auth failure!")
		return nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))
	streamServer := &mockServerStream{
		ctx: ctx,
	}

	// test
	err := authStreamServerInterceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler, authFunc)

	// verify
	assert.Equal(t, expectedErr, err)
	assert.True(t, authCalled)
}

func TestDefaultStreamInterceptorMissingMetadata(t *testing.T) {
	// prepare
	authFunc := func(context.Context, map[string][]string) (context.Context, error) {
		assert.FailNow(t, "the auth func should not have been called!")
		return context.Background(), nil
	}
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		assert.FailNow(t, "the handler should not have been called!")
		return nil
	}
	streamServer := &mockServerStream{
		ctx: context.Background(),
	}

	// test
	err := authStreamServerInterceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler, authFunc)

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
	recordedContext context.Context
}

func (gts *grpcTraceServer) Export(ctx context.Context, _ ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	gts.recordedContext = ctx
	return ptraceotlp.NewExportResponse(), nil
}

// tempSocketName provides a temporary Unix socket name for testing.
func tempSocketName(t *testing.T) string {
	tmpfile, err := os.CreateTemp("", "sock")
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
