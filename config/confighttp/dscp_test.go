// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
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
			cfg := &ClientConfig{DSCP: tt.dscp}
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

func TestClientConfig_DSCP_ToClient_NoDSCP(t *testing.T) {
	// When DSCP is 0 (default), the transport should use the default DialContext
	cfg := NewDefaultClientConfig()
	cfg.Endpoint = "http://localhost:1234"
	cfg.DSCP = 0

	tel := componenttest.NewNopTelemetrySettings()
	tel.TracerProvider = nil
	client, err := cfg.ToClient(context.Background(), nil, tel)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify the transport's DialContext is the same as the default transport's
	transport := client.Transport.(*http.Transport)
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	// When DSCP is 0, we should not have overridden the DialContext from Clone()
	// Both should be equivalent (either both nil or both set by Clone)
	assert.Equal(t,
		defaultTransport.DialContext == nil,
		transport.DialContext == nil,
		"DialContext nil-ness should match default transport when DSCP is 0")
}

func TestClientConfig_DSCP_ToClient_WithDSCP(t *testing.T) {
	// When DSCP is set, a custom DialContext should be configured
	cfg := NewDefaultClientConfig()
	cfg.Endpoint = "http://localhost:1234"
	cfg.DSCP = 46 // EF

	tel := componenttest.NewNopTelemetrySettings()
	tel.TracerProvider = nil
	client, err := cfg.ToClient(context.Background(), nil, tel)
	require.NoError(t, err)
	require.NotNil(t, client)

	transport := client.Transport.(*http.Transport)
	assert.NotNil(t, transport.DialContext, "DialContext should be set when DSCP > 0")
}

func TestClientConfig_DSCP_Connection(t *testing.T) {
	// Verify DSCP-configured HTTP client can actually connect
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	// Start a simple HTTP server
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := &http.Server{Handler: handler}
	go func() {
		_ = server.Serve(ln)
	}()
	defer server.Close()

	cfg := NewDefaultClientConfig()
	cfg.Endpoint = "http://" + ln.Addr().String()
	cfg.DSCP = 46 // EF

	tel := componenttest.NewNopTelemetrySettings()
	tel.TracerProvider = nil
	client, err := cfg.ToClient(context.Background(), nil, tel)
	require.NoError(t, err)

	resp, err := client.Get("http://" + ln.Addr().String() + "/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}
