// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
)

func TestClientConfig_DSCP_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dscp    int
		wantErr bool
	}{
		{name: "zero_default", dscp: 0, wantErr: false},
		{name: "valid_ef", dscp: 46, wantErr: false},
		{name: "valid_af41", dscp: 34, wantErr: false},
		{name: "valid_max", dscp: 63, wantErr: false},
		{name: "invalid_negative", dscp: -1, wantErr: true},
		{name: "invalid_too_large", dscp: 64, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultClientConfig()
			cfg.Endpoint = "localhost:1234"
			cfg.DSCP = tt.dscp
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid DSCP value")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClientConfig_DSCP_ToClientConn(t *testing.T) {
	// Verify that a gRPC client connection can be created with DSCP set
	cfg := ClientConfig{
		Endpoint: "localhost:1234",
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
		DSCP: 46, // EF
	}

	conn, err := cfg.ToClientConn(context.Background(), nil, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestClientConfig_DSCP_GetGrpcDialOptions(t *testing.T) {
	tests := []struct {
		name      string
		dscp      int
		expectLen int // minimum number of options expected
	}{
		{
			name: "no_dscp",
			dscp: 0,
		},
		{
			name: "with_dscp",
			dscp: 46,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: true,
				},
				DSCP: tt.dscp,
			}

			opts, err := cfg.getGrpcDialOptions(context.Background(), nil, componenttest.NewNopTelemetrySettings(), nil)
			require.NoError(t, err)
			require.NotEmpty(t, opts)

			// When DSCP is set, we expect one more option (WithContextDialer)
			if tt.dscp > 0 {
				cfgNoDSCP := &ClientConfig{
					Endpoint: "localhost:1234",
					TLS: configtls.ClientConfig{
						Insecure: true,
					},
					DSCP: 0,
				}
				optsNoDSCP, err := cfgNoDSCP.getGrpcDialOptions(context.Background(), nil, componenttest.NewNopTelemetrySettings(), nil)
				require.NoError(t, err)
				assert.Greater(t, len(opts), len(optsNoDSCP), "DSCP should add an extra dial option")
			}
		})
	}
}

func TestClientConfig_DSCP_EndToEnd(t *testing.T) {
	// Create a real gRPC server and connect with DSCP marking
	srv, addr := (&grpcTraceServer{}).startTestServer(t, configoptional.Some(ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "localhost:0",
			Transport: confignet.TransportTypeTCP,
		},
	}))
	defer srv.Stop()

	cfg := ClientConfig{
		Endpoint: addr,
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
		DSCP: 46, // EF
	}

	conn, err := cfg.ToClientConn(context.Background(), nil, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}
