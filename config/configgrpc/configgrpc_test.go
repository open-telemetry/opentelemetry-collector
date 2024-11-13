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
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension/auth"
	"go.opentelemetry.io/collector/extension/auth/authtest"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

func TestNewDefaultKeepaliveClientConfig(t *testing.T) {
	expectedKeepaliveClientConfig := &KeepaliveClientConfig{
		Time:    time.Second * 10,
		Timeout: time.Second * 10,
	}
	keepaliveClientConfig := NewDefaultKeepaliveClientConfig()
	assert.Equal(t, expectedKeepaliveClientConfig, keepaliveClientConfig)
}

func TestNewDefaultClientConfig(t *testing.T) {
	expected := &ClientConfig{
		TLSSetting:   configtls.NewDefaultClientConfig(),
		Keepalive:    NewDefaultKeepaliveClientConfig(),
		Auth:         configauth.NewDefaultAuthentication(),
		BalancerName: BalancerName(),
	}

	result := NewDefaultClientConfig()

	assert.Equal(t, expected, result)
}
func TestNewDefaultKeepaliveServerParameters(t *testing.T) {
	expectedParams := &KeepaliveServerParameters{}
	params := NewDefaultKeepaliveServerParameters()

	assert.Equal(t, expectedParams, params)
}
func TestNewDefaultKeepaliveEnforcementPolicy(t *testing.T) {
	expectedPolicy := &KeepaliveEnforcementPolicy{}

	policy := NewDefaultKeepaliveEnforcementPolicy()

	assert.Equal(t, expectedPolicy, policy)
}

func TestNewDefaultKeepaliveServerConfig(t *testing.T) {
	expected := &KeepaliveServerConfig{
		ServerParameters:  NewDefaultKeepaliveServerParameters(),
		EnforcementPolicy: NewDefaultKeepaliveEnforcementPolicy(),
	}
	result := NewDefaultKeepaliveServerConfig()
	assert.Equal(t, expected, result)
}

func TestNewDefaultServerConfig(t *testing.T) {
	expected := &ServerConfig{
		Keepalive: NewDefaultKeepaliveServerConfig(),
		Auth:      configauth.NewDefaultAuthentication(),
	}

	result := NewDefaultServerConfig()

	assert.Equal(t, expected, result)
}

var (
	componentID   = component.MustNewID("component")
	testAuthID    = component.MustNewID("testauth")
	mockID        = component.MustNewID("mock")
	doesntExistID = component.MustNewID("doesntexist")
)

func TestDefaultGrpcClientSettings(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(componentID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })
	tlsClientConfig := configtls.NewDefaultClientConfig()
	tlsClientConfig.Insecure = true
	gcs := &ClientConfig{
		TLSSetting: tlsClientConfig,
	}
	opts, err := gcs.getGrpcDialOptions(context.Background(), componenttest.NewNopHost(), tt.TelemetrySettings(), []ToClientConnOption{})
	require.NoError(t, err)
	assert.Len(t, opts, 2)
}

func TestGrpcClientExtraOption(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(componentID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })
	tlsClientConfig := configtls.NewDefaultClientConfig()
	tlsClientConfig.Insecure = true
	gcs := &ClientConfig{
		TLSSetting: tlsClientConfig,
	}
	extraOpt := grpc.WithUserAgent("test-agent")
	opts, err := gcs.getGrpcDialOptions(
		context.Background(),
		componenttest.NewNopHost(),
		tt.TelemetrySettings(),
		[]ToClientConnOption{WithGrpcDialOption(extraOpt)},
	)
	require.NoError(t, err)
	assert.Len(t, opts, 3)
	assert.Equal(t, opts[2], extraOpt)
}

func TestAllGrpcClientSettings(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(componentID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })
	tlsClientConfig := configtls.NewDefaultClientConfig()
	tests := []struct {
		settings ClientConfig
		name     string
		host     component.Host
	}{
		{
			name: "test all with gzip compression",
			settings: ClientConfig{
				Headers: map[string]configopaque.String{
					"test": "test",
				},
				Endpoint:    "localhost:1234",
				Compression: configcompression.TypeGzip,
				TLSSetting:  tlsClientConfig,
				Keepalive: &KeepaliveClientConfig{
					Time:                time.Second,
					Timeout:             time.Second,
					PermitWithoutStream: true,
				},
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				WaitForReady:    true,
				BalancerName:    "round_robin",
				Authority:       "pseudo-authority",
				Auth:            &configauth.Authentication{AuthenticatorID: testAuthID},
			},
			host: &mockHost{
				ext: map[component.ID]component.Component{
					testAuthID: &authtest.MockClient{},
				},
			},
		},
		{
			name: "test all with snappy compression",
			settings: ClientConfig{
				Headers: map[string]configopaque.String{
					"test": "test",
				},
				Endpoint:    "localhost:1234",
				Compression: configcompression.TypeSnappy,
				TLSSetting:  tlsClientConfig,
				Keepalive: &KeepaliveClientConfig{
					Time:                time.Second,
					Timeout:             time.Second,
					PermitWithoutStream: true,
				},
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				WaitForReady:    true,
				BalancerName:    "round_robin",
				Authority:       "pseudo-authority",
				Auth:            &configauth.Authentication{AuthenticatorID: testAuthID},
			},
			host: &mockHost{
				ext: map[component.ID]component.Component{
					testAuthID: &authtest.MockClient{},
				},
			},
		},
		{
			name: "test all with zstd compression",
			settings: ClientConfig{
				Headers: map[string]configopaque.String{
					"test": "test",
				},
				Endpoint:    "localhost:1234",
				Compression: configcompression.TypeZstd,
				TLSSetting:  tlsClientConfig,
				Keepalive: &KeepaliveClientConfig{
					Time:                time.Second,
					Timeout:             time.Second,
					PermitWithoutStream: true,
				},
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				WaitForReady:    true,
				BalancerName:    "round_robin",
				Authority:       "pseudo-authority",
				Auth:            &configauth.Authentication{AuthenticatorID: testAuthID},
			},
			host: &mockHost{
				ext: map[component.ID]component.Component{
					testAuthID: &authtest.MockClient{},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts, err := test.settings.getGrpcDialOptions(context.Background(), test.host, tt.TelemetrySettings(), []ToClientConnOption{})
			require.NoError(t, err)
			assert.Len(t, opts, 9)
		})
	}
}

func TestDefaultGrpcServerSettings(t *testing.T) {
	gss := &ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint: "0.0.0.0:1234",
		},
	}
	opts, err := gss.getGrpcServerOptions(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings(), []ToServerOption{})
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
		componenttest.NewNopHost(),
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
	tlsServerConfig := configtls.NewDefaultServerConfig()
	gss := &ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "localhost:1234",
			Transport: confignet.TransportTypeTCP,
		},
		TLSSetting:           &tlsServerConfig,
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
	opts, err := gss.getGrpcServerOptions(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings(), []ToServerOption{})
	require.NoError(t, err)
	assert.Len(t, opts, 10)
}

func TestGrpcServerAuthSettings(t *testing.T) {
	gss := &ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint: "0.0.0.0:1234",
		},
	}
	gss.Auth = &configauth.Authentication{
		AuthenticatorID: mockID,
	}
	host := &mockHost{
		ext: map[component.ID]component.Component{
			mockID: auth.NewServer(),
		},
	}
	srv, err := gss.ToServer(context.Background(), host, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	assert.NotNil(t, srv)
}

func TestGrpcClientConfigInvalidBalancer(t *testing.T) {
	tlsClientConfig := configtls.NewDefaultClientConfig()
	settings := ClientConfig{
		Headers: map[string]configopaque.String{
			"test": "test",
		},
		Endpoint:    "localhost:1234",
		Compression: "gzip",
		TLSSetting:  tlsClientConfig,
		Keepalive: &KeepaliveClientConfig{
			Time:                time.Second,
			Timeout:             time.Second,
			PermitWithoutStream: true,
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		WaitForReady:    true,
		BalancerName:    "test",
	}
	assert.ErrorContains(t, settings.Validate(), "invalid balancer_name: test")
}

func TestGRPCClientSettingsError(t *testing.T) {
	tlsClientConfigCADoesntExist := configtls.NewDefaultClientConfig()
	tlsConfig := configtls.NewDefaultConfig()
	tlsConfig.CAFile = "/doesnt/exist"
	tlsClientConfigCADoesntExist.Config = tlsConfig

	tlsClientConfigWithInsecure := configtls.NewDefaultClientConfig()
	tlsClientConfigWithInsecure.Insecure = true
	tests := []struct {
		settings ClientConfig
		err      string
		host     component.Host
	}{
		{
			err: "failed to load TLS config: failed to load CA CertPool File: failed to load cert /doesnt/exist:",
			settings: ClientConfig{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSSetting:  tlsClientConfigCADoesntExist,
				Keepalive:   nil,
			},
		},
		{
			err: "failed to load TLS config: failed to load TLS cert and key: for auth via TLS, provide both certificate and key, or neither",
			settings: ClientConfig{
				Headers:     nil,
				Endpoint:    "",
				Compression: "",
				TLSSetting:  tlsClientConfigCADoesntExist,
				Keepalive:   nil,
			},
		},
		{
			err: "failed to resolve authenticator \"doesntexist\": authenticator not found",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorID: doesntExistID},
			},
			host: &mockHost{ext: map[component.ID]component.Component{}},
		},
		{
			err: "no extensions configuration available",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorID: doesntExistID},
			},
			host: &mockHost{},
		},
		{
			err: "unsupported compression type \"zlib\"",
			settings: ClientConfig{
				Endpoint:    "localhost:1234",
				TLSSetting:  tlsClientConfigWithInsecure,
				Compression: "zlib",
			},
			host: &mockHost{},
		},
		{
			err: "unsupported compression type \"deflate\"",
			settings: ClientConfig{
				Endpoint:    "localhost:1234",
				TLSSetting:  tlsClientConfigWithInsecure,
				Compression: "deflate",
			},
			host: &mockHost{},
		},
		{
			err: "unsupported compression type \"bad\"",
			settings: ClientConfig{
				Endpoint:    "localhost:1234",
				TLSSetting:  tlsClientConfigWithInsecure,
				Compression: "bad",
			},
			host: &mockHost{},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			require.NoError(t, test.settings.Validate())
			_, err := test.settings.ToClientConn(context.Background(), test.host, componenttest.NewNopTelemetrySettings())
			require.Error(t, err)
			assert.ErrorContains(t, err, test.err)
		})
	}
}

func TestUseSecure(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(componentID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	gcs := &ClientConfig{
		Headers:     nil,
		Endpoint:    "",
		Compression: "",
		TLSSetting:  configtls.NewDefaultClientConfig(),
		Keepalive:   nil,
	}
	dialOpts, err := gcs.getGrpcDialOptions(context.Background(), componenttest.NewNopHost(), tt.TelemetrySettings(), []ToClientConnOption{})
	require.NoError(t, err)
	assert.Len(t, dialOpts, 2)
}

func TestGRPCServerWarning(t *testing.T) {
	tests := []struct {
		name     string
		settings ServerConfig
		len      int
	}{
		{
			settings: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "0.0.0.0:1234",
					Transport: confignet.TransportTypeTCP,
				},
			},
			len: 1,
		},
		{
			settings: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "127.0.0.1:1234",
					Transport: confignet.TransportTypeTCP,
				},
			},
			len: 0,
		},
		{
			settings: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "0.0.0.0:1234",
					Transport: confignet.TransportTypeUnix,
				},
			},
			len: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := componenttest.NewNopTelemetrySettings()
			logger, observed := observer.New(zap.DebugLevel)
			set.Logger = zap.New(logger)

			opts, err := test.settings.getGrpcServerOptions(componenttest.NewNopHost(), set, []ToServerOption{})
			require.NoError(t, err)
			require.NotNil(t, opts)
			_ = grpc.NewServer(opts...)

			require.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), test.len)
		})
	}
}

func TestGRPCServerSettingsError(t *testing.T) {
	tlsConfigCADoesntExist := configtls.NewDefaultConfig()
	tlsConfigCADoesntExist.CAFile = "/doesnt/exist"
	tlsServerConfigCADoesntExist := configtls.NewDefaultServerConfig()
	tlsServerConfigCADoesntExist.Config = tlsConfigCADoesntExist

	tlsConfigCertDoesntExist := configtls.NewDefaultConfig()
	tlsConfigCertDoesntExist.CertFile = "/doesnt/exist"
	tlsServerConfigCertDoesntExist := configtls.NewDefaultServerConfig()
	tlsServerConfigCertDoesntExist.Config = tlsConfigCertDoesntExist

	tlsServerConfigClientCADoesntExist := configtls.NewDefaultServerConfig()
	tlsServerConfigClientCADoesntExist.ClientCAFile = "/doesnt/exist"
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
				TLSSetting: &tlsServerConfigCADoesntExist,
			},
		},
		{
			err: "failed to load TLS config: failed to load TLS cert and key: for auth via TLS, provide both certificate and key, or neither",
			settings: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "127.0.0.1:1234",
					Transport: confignet.TransportTypeTCP,
				},
				TLSSetting: &tlsServerConfigCertDoesntExist,
			},
		},
		{
			err: "failed to load client CA CertPool: failed to load CA /doesnt/exist:",
			settings: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "127.0.0.1:1234",
					Transport: confignet.TransportTypeTCP,
				},
				TLSSetting: &tlsServerConfigClientCADoesntExist,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			_, err := test.settings.ToServer(context.Background(), componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
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
	tt, err := componenttest.SetupTelemetry(componentID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	tlsClientConfigWithInsecure := configtls.NewDefaultClientConfig()
	tlsClientConfigWithInsecure.Insecure = true

	tlsClientConfigWithCAFile := configtls.NewDefaultClientConfig()
	tlsConfigWithCAFile := configtls.NewDefaultConfig()
	tlsConfigWithCAFile.CAFile = filepath.Join("testdata", "ca.crt")
	tlsClientConfigWithCAFile.Config = tlsConfigWithCAFile
	tlsClientConfigWithCAFile.ServerName = "localhost"

	tlsClientConfigWithCerts := configtls.NewDefaultClientConfig()
	tlsClientConfigWithCerts.ServerName = "localhost"
	tlsConfig := configtls.NewDefaultConfig()
	tlsConfig.CAFile = filepath.Join("testdata", "ca.crt")
	tlsConfig.CertFile = filepath.Join("testdata", "client.crt")
	tlsConfig.KeyFile = filepath.Join("testdata", "client.key")
	tlsClientConfigWithCerts.Config = tlsConfig

	tlsServerConfigWithCerts := configtls.NewDefaultServerConfig()
	tlsServerConfigWithCerts.Config.CAFile = filepath.Join("testdata", "ca.crt")
	tlsServerConfigWithCerts.Config.CertFile = filepath.Join("testdata", "server.crt")
	tlsServerConfigWithCerts.Config.KeyFile = filepath.Join("testdata", "server.key")

	tlsServerConfigWithCAFile := configtls.NewDefaultServerConfig()
	tlsServerConfigWithCAFile.Config.CAFile = filepath.Join("testdata", "ca.crt")

	tlsServerConfigWithClientCAFile := configtls.NewDefaultServerConfig()
	tlsServerConfigWithClientCAFile.Config.CAFile = filepath.Join("testdata", "ca.crt")
	tlsServerConfigWithClientCAFile.Config.CertFile = filepath.Join("testdata", "server.crt")
	tlsServerConfigWithClientCAFile.Config.KeyFile = filepath.Join("testdata", "server.key")
	tlsServerConfigWithClientCAFile.ClientCAFile = filepath.Join("testdata", "ca.crt")

	tlsServerConfigWithWrongClientCAFile := configtls.NewDefaultServerConfig()
	tlsServerConfigWithWrongClientCAFile.Config.CAFile = filepath.Join("testdata", "ca.crt")
	tlsServerConfigWithWrongClientCAFile.Config.CertFile = filepath.Join("testdata", "server.crt")
	tlsServerConfigWithWrongClientCAFile.Config.KeyFile = filepath.Join("testdata", "server.key")
	tlsServerConfigWithWrongClientCAFile.ClientCAFile = filepath.Join("testdata", "server.crt")
	tests := []struct {
		name           string
		tlsServerCreds *configtls.ServerConfig
		tlsClientCreds *configtls.ClientConfig
		hasError       bool
	}{
		{
			name:           "noTLS",
			tlsServerCreds: nil,
			tlsClientCreds: &tlsClientConfigWithInsecure,
		},
		{
			name:           "TLS",
			tlsServerCreds: &tlsServerConfigWithCerts,
			tlsClientCreds: &tlsClientConfigWithCAFile,
		},
		{
			name:           "NoServerCertificates",
			tlsServerCreds: &tlsServerConfigWithCAFile,
			tlsClientCreds: &tlsClientConfigWithCAFile,
			hasError:       true,
		},
		{
			name:           "mTLS",
			tlsServerCreds: &tlsServerConfigWithClientCAFile,
			tlsClientCreds: &tlsClientConfigWithCerts,
		},
		{
			name:           "NoClientCertificate",
			tlsServerCreds: &tlsServerConfigWithClientCAFile,
			tlsClientCreds: &tlsClientConfigWithCAFile,
			hasError:       true,
		},
		{
			name:           "WrongClientCA",
			tlsServerCreds: &tlsServerConfigWithWrongClientCAFile,
			tlsClientCreds: &tlsClientConfigWithCerts,
			hasError:       true,
		},
	}
	// prepare

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gss := &ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				TLSSetting: test.tlsServerCreds,
			}
			ln, err := gss.NetAddr.Listen(context.Background())
			require.NoError(t, err)
			s, err := gss.ToServer(context.Background(), componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)
			ptraceotlp.RegisterGRPCServer(s, &grpcTraceServer{})

			go func() {
				_ = s.Serve(ln)
			}()

			gcs := &ClientConfig{
				Endpoint:   ln.Addr().String(),
				TLSSetting: *test.tlsClientCreds,
			}
			grpcClientConn, errClient := gcs.ToClientConn(context.Background(), componenttest.NewNopHost(), tt.TelemetrySettings())
			require.NoError(t, errClient)
			defer func() { assert.NoError(t, grpcClientConn.Close()) }()
			c := ptraceotlp.NewGRPCClient(grpcClientConn)
			ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
			resp, errResp := c.Export(ctx, ptraceotlp.NewExportRequest(), grpc.WaitForReady(true))
			if test.hasError {
				require.Error(t, errResp)
			} else {
				require.NoError(t, errResp)
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
	tt, err := componenttest.SetupTelemetry(componentID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	socketName := tempSocketName(t)
	gss := &ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  socketName,
			Transport: confignet.TransportTypeUnix,
		},
	}
	ln, err := gss.NetAddr.Listen(context.Background())
	require.NoError(t, err)
	srv, err := gss.ToServer(context.Background(), componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	ptraceotlp.RegisterGRPCServer(srv, &grpcTraceServer{})

	go func() {
		_ = srv.Serve(ln)
	}()
	tlsClientConfig := configtls.NewDefaultClientConfig()
	tlsClientConfig.Insecure = true
	gcs := &ClientConfig{
		Endpoint:   "unix://" + ln.Addr().String(),
		TLSSetting: tlsClientConfig,
	}
	grpcClientConn, errClient := gcs.ToClientConn(context.Background(), componenttest.NewNopHost(), tt.TelemetrySettings())
	require.NoError(t, errClient)
	defer func() { assert.NoError(t, grpcClientConn.Close()) }()
	c := ptraceotlp.NewGRPCClient(grpcClientConn)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
	resp, errResp := c.Export(ctx, ptraceotlp.NewExportRequest(), grpc.WaitForReady(true))
	require.NoError(t, errResp)
	assert.NotNil(t, resp)
	cancelFunc()
	srv.Stop()
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
	var outContext context.Context

	stream := &mockedStream{
		ctx: inCtx,
	}

	handler := func(_ any, stream grpc.ServerStream) error {
		outContext = stream.Context()
		return nil
	}

	// test
	err := enhanceStreamWithClientInformation(false)(nil, stream, nil, handler)

	// verify
	require.NoError(t, err)

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
		name   string
		tester func(context.Context, ptraceotlp.GRPCClient)
	}{
		{
			// we only have unary services, we don't have any clients we could use
			// to test with streaming services
			name: "unary",
			tester: func(ctx context.Context, cl ptraceotlp.GRPCClient) {
				resp, errResp := cl.Export(ctx, ptraceotlp.NewExportRequest())
				require.NoError(t, errResp)
				require.NotNil(t, resp)
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &grpcTraceServer{}
			var l net.Listener
			tlsClientConfig := configtls.NewDefaultClientConfig()
			tlsClientConfig.Insecure = true
			// prepare the server
			{
				gss := &ServerConfig{
					NetAddr: confignet.AddrConfig{
						Endpoint:  "localhost:0",
						Transport: confignet.TransportTypeTCP,
					},
				}
				srv, err := gss.ToServer(context.Background(), componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
				require.NoError(t, err)
				ptraceotlp.RegisterGRPCServer(srv, mock)

				defer srv.Stop()

				l, err = gss.NetAddr.Listen(context.Background())
				require.NoError(t, err)

				go func() {
					_ = srv.Serve(l)
				}()
			}

			// prepare the client and execute a RPC
			{
				gcs := &ClientConfig{
					Endpoint:   l.Addr().String(),
					TLSSetting: tlsClientConfig,
				}

				tel, err := componenttest.SetupTelemetry(componentID)
				require.NoError(t, err)
				defer func() {
					require.NoError(t, tel.Shutdown(context.Background()))
				}()

				grpcClientConn, errClient := gcs.ToClientConn(context.Background(), componenttest.NewNopHost(), tel.TelemetrySettings())
				require.NoError(t, errClient)
				defer func() { assert.NoError(t, grpcClientConn.Close()) }()

				cl := ptraceotlp.NewGRPCClient(grpcClientConn)
				ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancelFunc()

				// test
				tt.tester(ctx, cl)
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

	// test
	res, err := authUnaryServerInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler, auth.NewServer(auth.WithServerAuthenticate(authFunc)))

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

	// test
	res, err := authUnaryServerInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler, auth.NewServer(auth.WithServerAuthenticate(authFunc)))

	// verify
	assert.Nil(t, res)
	require.ErrorContains(t, err, expectedErr.Error())
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
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

	// test
	res, err := authUnaryServerInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler, auth.NewServer(auth.WithServerAuthenticate(authFunc)))

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
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))
	streamServer := &mockServerStream{
		ctx: ctx,
	}

	// test
	err := authStreamServerInterceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler, auth.NewServer(auth.WithServerAuthenticate(authFunc)))

	// verify
	require.NoError(t, err)
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
	handler := func(any, grpc.ServerStream) error {
		assert.FailNow(t, "the handler should not have been called on auth failure!")
		return nil
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "some-auth-data"))
	streamServer := &mockServerStream{
		ctx: ctx,
	}

	// test
	err := authStreamServerInterceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler, auth.NewServer(auth.WithServerAuthenticate(authFunc)))

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

	// test
	err := authStreamServerInterceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler, auth.NewServer(auth.WithServerAuthenticate(authFunc)))

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
	ext map[component.ID]component.Component
}

func (nh *mockHost) GetExtensions() map[component.ID]component.Component {
	return nh.ext
}
