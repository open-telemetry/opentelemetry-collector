// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
)

// ---- ClientConfig ----

// A Keepalive set programmatically after unmarshaling takes precedence over the
// deprecated fields in ToClient.
func TestClientConfigProgrammaticKeepaliveAfterUnmarshal(t *testing.T) {
	cfg := NewDefaultClientConfig()
	require.NoError(t, confmap.NewFromStringMap(map[string]any{}).Unmarshal(&cfg))
	cfg.Keepalive = configoptional.Some(KeepaliveClientConfig{
		IdleConnTimeout: 5 * time.Minute,
		MaxIdleConns:    7,
	})

	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = nil
	settings.TracerProvider = nil
	client, err := cfg.ToClient(t.Context(), nil, settings)
	require.NoError(t, err)
	transport := client.Transport.(*http.Transport)
	assert.Equal(t, 5*time.Minute, transport.IdleConnTimeout)
	assert.Equal(t, 7, transport.MaxIdleConns)
	assert.False(t, transport.DisableKeepAlives)
}

// The keepalive defaults must stay in sync with the defaults that
// NewDefaultClientConfig sets on the corresponding deprecated fields.
func TestNewDefaultKeepaliveClientConfig(t *testing.T) {
	defaultCfg := NewDefaultClientConfig()
	keepalive := NewDefaultKeepaliveClientConfig()
	assert.Equal(t, defaultCfg.Keepalive.Get().IdleConnTimeout, keepalive.IdleConnTimeout)
	assert.Equal(t, defaultCfg.Keepalive.Get().MaxIdleConns, keepalive.MaxIdleConns)
	assert.Equal(t, defaultCfg.Keepalive.Get().MaxIdleConnsPerHost, keepalive.MaxIdleConnsPerHost)
}

// ---- ServerConfig ----

func TestServerConfigUnmarshalKeepalive(t *testing.T) {
	tests := []struct {
		name         string
		prepare      func(*ServerConfig)
		conf         map[string]any
		expectError  bool
		verifyConfig func(*testing.T, *ServerConfig)
	}{
		{
			name: "no keepalive config — defaults",
			conf: map[string]any{},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Equal(t, 60*time.Second, cfg.Keepalive.Get().IdleTimeout)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "new keepalive only",
			conf: map[string]any{"keepalive": map[string]any{"idle_timeout": "2m"}},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Equal(t, 2*time.Minute, cfg.Keepalive.Get().IdleTimeout)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "programmatic keepalive",
			prepare: func(cfg *ServerConfig) {
				cfg.Keepalive = configoptional.Some(KeepaliveServerConfig{IdleTimeout: 5 * time.Minute})
			},
			conf: map[string]any{},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Equal(t, 5*time.Minute, cfg.Keepalive.Get().IdleTimeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultServerConfig()
			if tt.prepare != nil {
				tt.prepare(&cfg)
			}
			err := confmap.NewFromStringMap(tt.conf).Unmarshal(&cfg)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "keepalive")
				return
			}
			require.NoError(t, err)
			if tt.verifyConfig != nil {
				tt.verifyConfig(t, &cfg)
			}
		})
	}
}

// A Keepalive set programmatically after unmarshaling takes precedence over the
// deprecated fields in ToServer.
func TestServerConfigProgrammaticKeepaliveAfterUnmarshal(t *testing.T) {
	cfg := NewDefaultServerConfig()
	cfg.Keepalive = configoptional.Some(KeepaliveServerConfig{IdleTimeout: 5 * time.Minute})

	srv, err := cfg.ToServer(t.Context(), nil, componenttest.NewNopTelemetrySettings(), http.NewServeMux())
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, srv.IdleTimeout)
}

// The keepalive defaults must stay in sync with the defaults that
// NewDefaultServerConfig sets on the corresponding deprecated fields.
func TestNewDefaultKeepaliveServerConfig(t *testing.T) {
	defaultCfg := NewDefaultServerConfig()
	keepalive := NewDefaultKeepaliveServerConfig()
	assert.Equal(t, defaultCfg.Keepalive.Get().IdleTimeout, keepalive.IdleTimeout)
}

// ---- squash embedding ----

// namedSquashClientConfig mirrors how components like otlphttpexporter embed
// ClientConfig as a named field with a squash tag alongside sibling fields.
type namedSquashClientConfig struct {
	ClientConfig ClientConfig `mapstructure:",squash"`
	Extra        string       `mapstructure:"extra"`
}

func TestClientConfigSquashNamedField(t *testing.T) {
	cfg := namedSquashClientConfig{ClientConfig: NewDefaultClientConfig()}
	conf := confmap.NewFromStringMap(map[string]any{
		"endpoint": "http://localhost:4318",
	})
	require.NoError(t, conf.Unmarshal(&cfg))

	assert.Equal(t, "http://localhost:4318", cfg.ClientConfig.Endpoint)
	assert.Equal(t, configoptional.Some(NewDefaultKeepaliveClientConfig()), cfg.ClientConfig.Keepalive)
}

// namedSquashServerConfig mirrors how components like zpagesextension embed
// ServerConfig as a named field with a squash tag alongside sibling fields.
type namedSquashServerConfig struct {
	ServerConfig ServerConfig `mapstructure:",squash"`
	Extra        string       `mapstructure:"extra"`
}

func TestServerConfigSquashNamedField(t *testing.T) {
	cfg := namedSquashServerConfig{ServerConfig: NewDefaultServerConfig()}
	conf := confmap.NewFromStringMap(map[string]any{
		"endpoint":            "localhost:0",
		"keep_alives_enabled": false,
		"extra":               "sibling",
	})
	require.NoError(t, conf.Unmarshal(&cfg))

	assert.Equal(t, "localhost:0", cfg.ServerConfig.NetAddr.Endpoint)
	assert.Equal(t, "sibling", cfg.Extra)
	assert.Equal(t, configoptional.Some(NewDefaultKeepaliveServerConfig()), cfg.ServerConfig.Keepalive)
}
