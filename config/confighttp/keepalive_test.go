// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
)

// ---- ClientConfig ----

// Unmarshal folds the 'keepalive' section into the deprecated fields and always
// resets the Optional to None; the section's effect shows in the flat fields.
func TestClientConfigUnmarshalKeepalive(t *testing.T) {
	tests := []struct {
		name         string
		prepare      func(*ClientConfig)
		conf         map[string]any
		expectError  bool
		verifyConfig func(*testing.T, *ClientConfig)
	}{
		{
			name: "no keepalive config — defaults",
			conf: map[string]any{},
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, 90*time.Second, cfg.IdleConnTimeout)
				assert.Equal(t, 100, cfg.MaxIdleConns)
				assert.False(t, cfg.DisableKeepAlives)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "new keepalive only — unset fields keep defaults",
			conf: map[string]any{"keepalive": map[string]any{
				"idle_conn_timeout":       "60s",
				"max_idle_conns_per_host": 5,
			}},
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, 60*time.Second, cfg.IdleConnTimeout)
				assert.Equal(t, 100, cfg.MaxIdleConns)
				assert.Equal(t, 5, cfg.MaxIdleConnsPerHost)
				assert.False(t, cfg.DisableKeepAlives)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "deprecated fields only",
			conf: map[string]any{"idle_conn_timeout": "60s", "max_idle_conns": 50},
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, 60*time.Second, cfg.IdleConnTimeout)
				assert.Equal(t, 50, cfg.MaxIdleConns)
				assert.Len(t, cfg.deprecationWarnings, 2)
			},
		},
		{
			name: "null keepalive treated as unset",
			conf: map[string]any{"keepalive": nil},
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, 90*time.Second, cfg.IdleConnTimeout)
				assert.False(t, cfg.DisableKeepAlives)
			},
		},
		{
			name: "keepalive disabled via enabled false",
			conf: map[string]any{"keepalive": map[string]any{"enabled": false}},
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.True(t, cfg.DisableKeepAlives)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "keepalive section re-enables keep-alives",
			prepare: func(cfg *ClientConfig) {
				cfg.DisableKeepAlives = true
			},
			conf: map[string]any{"keepalive": map[string]any{"idle_conn_timeout": "60s"}},
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.False(t, cfg.DisableKeepAlives)
			},
		},
		{
			name: "disable_keep_alives only",
			conf: map[string]any{"disable_keep_alives": true},
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.True(t, cfg.DisableKeepAlives)
				assert.Len(t, cfg.deprecationWarnings, 1)
			},
		},
		{
			name: "null keepalive + deprecated field — no conflict",
			conf: map[string]any{"keepalive": nil, "idle_conn_timeout": "60s"},
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, 60*time.Second, cfg.IdleConnTimeout)
			},
		},
		{
			name: "keepalive section + no-op deprecated value — no conflict",
			conf: map[string]any{"keepalive": map[string]any{}, "idle_conn_timeout": "0s"},
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				// The explicit legacy zero is honored, as it would be on its own.
				assert.Equal(t, time.Duration(0), cfg.IdleConnTimeout)
				assert.Equal(t, 100, cfg.MaxIdleConns)
			},
		},
		{
			name: "programmatic Keepalive folded into deprecated fields",
			prepare: func(cfg *ClientConfig) {
				cfg.Keepalive = configoptional.Some(KeepaliveClientConfig{
					IdleConnTimeout: 5 * time.Minute,
					MaxIdleConns:    10,
				})
			},
			conf: map[string]any{},
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, 5*time.Minute, cfg.IdleConnTimeout)
				assert.Equal(t, 10, cfg.MaxIdleConns)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "programmatic Keepalive overridden per-field by config",
			prepare: func(cfg *ClientConfig) {
				cfg.Keepalive = configoptional.Some(KeepaliveClientConfig{
					IdleConnTimeout: 5 * time.Minute,
					MaxIdleConns:    10,
				})
			},
			conf: map[string]any{"keepalive": map[string]any{"idle_conn_timeout": "1m"}},
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, 1*time.Minute, cfg.IdleConnTimeout)
				assert.Equal(t, 10, cfg.MaxIdleConns)
			},
		},
		{
			name:        "conflict: keepalive section + idle_conn_timeout",
			conf:        map[string]any{"keepalive": map[string]any{}, "idle_conn_timeout": "60s"},
			expectError: true,
		},
		{
			name:        "conflict: keepalive section + disable_keep_alives",
			conf:        map[string]any{"keepalive": map[string]any{}, "disable_keep_alives": true},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultClientConfig()
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
			assert.Equal(t, configoptional.None[KeepaliveClientConfig](), cfg.Keepalive)
			if tt.verifyConfig != nil {
				tt.verifyConfig(t, &cfg)
			}
		})
	}
}

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

func TestClientConfigDeprecatedWarningsLogged(t *testing.T) {
	cfg := NewDefaultClientConfig()
	conf := confmap.NewFromStringMap(map[string]any{
		"endpoint":                "http://localhost:4318",
		"idle_conn_timeout":       "90s",
		"max_idle_conns":          100,
		"max_idle_conns_per_host": 10,
	})
	require.NoError(t, conf.Unmarshal(&cfg))

	core, observed := observer.New(zapcore.WarnLevel)
	settings := component.TelemetrySettings{Logger: zap.New(core)}
	_, err := cfg.ToClient(t.Context(), nil, settings)
	require.NoError(t, err)

	entries := observed.All()
	require.Len(t, entries, 3)
	for _, entry := range entries {
		assert.Equal(t, zapcore.WarnLevel, entry.Level)
		assert.Contains(t, entry.Message, "deprecated")
	}
}

// Both spellings of disabling keep-alives must reach the transport; only the
// deprecated one warns.
func TestClientConfigDisableKeepAlives(t *testing.T) {
	tests := []struct {
		name           string
		conf           map[string]any
		expectWarnings int
	}{
		{
			name:           "deprecated disable_keep_alives",
			conf:           map[string]any{"disable_keep_alives": true},
			expectWarnings: 1,
		},
		{
			name:           "keepalive enabled false",
			conf:           map[string]any{"keepalive": map[string]any{"enabled": false}},
			expectWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultClientConfig()
			require.NoError(t, confmap.NewFromStringMap(tt.conf).Unmarshal(&cfg))

			core, observed := observer.New(zapcore.WarnLevel)
			settings := componenttest.NewNopTelemetrySettings()
			settings.MeterProvider = nil
			settings.TracerProvider = nil
			settings.Logger = zap.New(core)

			client, err := cfg.ToClient(t.Context(), nil, settings)
			require.NoError(t, err)
			assert.True(t, client.Transport.(*http.Transport).DisableKeepAlives)
			assert.Len(t, observed.All(), tt.expectWarnings)
		})
	}
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
				assert.Equal(t, 1*time.Minute, cfg.IdleTimeout)
				assert.True(t, cfg.KeepAlivesEnabled)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "new keepalive only",
			conf: map[string]any{"keepalive": map[string]any{"idle_timeout": "2m"}},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Equal(t, 2*time.Minute, cfg.IdleTimeout)
				assert.True(t, cfg.KeepAlivesEnabled)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "deprecated idle_timeout only",
			conf: map[string]any{"idle_timeout": "2m"},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Equal(t, 2*time.Minute, cfg.IdleTimeout)
				assert.Len(t, cfg.deprecationWarnings, 1)
			},
		},
		{
			name: "null keepalive treated as unset",
			conf: map[string]any{"keepalive": nil},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Equal(t, 1*time.Minute, cfg.IdleTimeout)
				assert.True(t, cfg.KeepAlivesEnabled)
			},
		},
		{
			name: "keepalive disabled via enabled false",
			conf: map[string]any{"keepalive": map[string]any{"enabled": false}},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.False(t, cfg.KeepAlivesEnabled)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "keepalive section re-enables keep-alives",
			prepare: func(cfg *ServerConfig) {
				cfg.KeepAlivesEnabled = false
			},
			conf: map[string]any{"keepalive": map[string]any{"idle_timeout": "2m"}},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.True(t, cfg.KeepAlivesEnabled)
			},
		},
		{
			name: "keep_alives_enabled false only",
			conf: map[string]any{"keep_alives_enabled": false},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.False(t, cfg.KeepAlivesEnabled)
				assert.Len(t, cfg.deprecationWarnings, 1)
			},
		},
		{
			name: "keep_alives_enabled true only — no-op",
			conf: map[string]any{"keep_alives_enabled": true},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.True(t, cfg.KeepAlivesEnabled)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "keep_alives_enabled true + keepalive section — no conflict",
			conf: map[string]any{
				"keep_alives_enabled": true,
				"keepalive":           map[string]any{"idle_timeout": "2m"},
			},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Equal(t, 2*time.Minute, cfg.IdleTimeout)
				assert.True(t, cfg.KeepAlivesEnabled)
			},
		},
		{
			name: "null keepalive + deprecated field — no conflict",
			conf: map[string]any{"keepalive": nil, "idle_timeout": "2m"},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Equal(t, 2*time.Minute, cfg.IdleTimeout)
			},
		},
		{
			name: "programmatic Keepalive folded into deprecated fields",
			prepare: func(cfg *ServerConfig) {
				cfg.Keepalive = configoptional.Some(KeepaliveServerConfig{IdleTimeout: 5 * time.Minute})
			},
			conf: map[string]any{},
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Equal(t, 5*time.Minute, cfg.IdleTimeout)
				assert.True(t, cfg.KeepAlivesEnabled)
			},
		},
		{
			name:        "conflict: keepalive section + idle_timeout",
			conf:        map[string]any{"keepalive": map[string]any{}, "idle_timeout": "2m"},
			expectError: true,
		},
		{
			name:        "conflict: keepalive section + keep_alives_enabled false",
			conf:        map[string]any{"keepalive": map[string]any{}, "keep_alives_enabled": false},
			expectError: true,
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
			assert.Equal(t, configoptional.None[KeepaliveServerConfig](), cfg.Keepalive)
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
	require.NoError(t, confmap.NewFromStringMap(map[string]any{"keep_alives_enabled": false}).Unmarshal(&cfg))
	cfg.Keepalive = configoptional.Some(KeepaliveServerConfig{IdleTimeout: 5 * time.Minute})

	srv, err := cfg.ToServer(t.Context(), nil, componenttest.NewNopTelemetrySettings(), http.NewServeMux())
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, srv.IdleTimeout)
}

func TestServerConfigDeprecatedWarningsLogged(t *testing.T) {
	cfg := NewDefaultServerConfig()
	conf := confmap.NewFromStringMap(map[string]any{
		"endpoint":            "0.0.0.0:4318",
		"idle_timeout":        "120s",
		"keep_alives_enabled": true,
	})
	require.NoError(t, conf.Unmarshal(&cfg))

	core, observed := observer.New(zapcore.WarnLevel)
	settings := component.TelemetrySettings{Logger: zap.New(core)}
	srv, err := cfg.ToServer(t.Context(), nil, settings, http.NewServeMux())
	require.NoError(t, err)
	require.NotNil(t, srv)

	// keep_alives_enabled: true is a no-op and produces no warning.
	entries := observed.All()
	require.Len(t, entries, 1)
	assert.Equal(t, zapcore.WarnLevel, entries[0].Level)
	assert.Contains(t, entries[0].Message, "'idle_timeout' is deprecated")
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
		"endpoint":          "http://localhost:4318",
		"idle_conn_timeout": "60s",
		"extra":             "sibling",
	})
	require.NoError(t, conf.Unmarshal(&cfg))

	assert.Equal(t, "http://localhost:4318", cfg.ClientConfig.Endpoint)
	assert.Equal(t, "sibling", cfg.Extra)
	assert.Equal(t, configoptional.None[KeepaliveClientConfig](), cfg.ClientConfig.Keepalive)
	assert.Equal(t, 60*time.Second, cfg.ClientConfig.IdleConnTimeout)
	assert.Equal(t, 100, cfg.ClientConfig.MaxIdleConns)
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
	assert.Equal(t, configoptional.None[KeepaliveServerConfig](), cfg.ServerConfig.Keepalive)
	assert.False(t, cfg.ServerConfig.KeepAlivesEnabled)
}

func TestClientConfigSquashMixedFieldsError(t *testing.T) {
	cfg := namedSquashClientConfig{ClientConfig: NewDefaultClientConfig()}
	conf := confmap.NewFromStringMap(map[string]any{
		"keepalive":         map[string]any{"idle_conn_timeout": "30s"},
		"idle_conn_timeout": "60s",
		"extra":             "sibling",
	})
	err := conf.Unmarshal(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "keepalive")
}
