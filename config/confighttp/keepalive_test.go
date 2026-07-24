// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"net/http"
	"path/filepath"
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
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func defaultClientKeepalive() configoptional.Optional[KeepaliveClientConfig] {
	return configoptional.Default(KeepaliveClientConfig{
		IdleConnTimeout: 90 * time.Second,
		MaxIdleConns:    100,
	})
}

func defaultServerKeepalive() configoptional.Optional[KeepaliveServerConfig] {
	return configoptional.Default(KeepaliveServerConfig{
		IdleTimeout: 1 * time.Minute,
	})
}

// ---- ClientConfig ----

func TestClientConfigUnmarshalKeepalive(t *testing.T) {
	tests := []struct {
		name         string
		conf         map[string]any
		expectError  bool
		expected     configoptional.Optional[KeepaliveClientConfig]
		verifyConfig func(*testing.T, *ClientConfig)
	}{
		{
			name:     "no keepalive config — section stays unset",
			conf:     map[string]any{},
			expected: defaultClientKeepalive(),
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, 90*time.Second, cfg.IdleConnTimeout)
				assert.Equal(t, 100, cfg.MaxIdleConns)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "new keepalive only — unset fields keep defaults",
			conf: map[string]any{"keepalive": map[string]any{"idle_conn_timeout": "60s"}},
			expected: configoptional.Some(KeepaliveClientConfig{
				IdleConnTimeout: 60 * time.Second,
				MaxIdleConns:    100,
			}),
		},
		{
			name:     "deprecated fields only — section stays unset",
			conf:     map[string]any{"idle_conn_timeout": "60s", "max_idle_conns": 50},
			expected: defaultClientKeepalive(),
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, 60*time.Second, cfg.IdleConnTimeout)
				assert.Equal(t, 50, cfg.MaxIdleConns)
				assert.Len(t, cfg.deprecationWarnings, 2)
			},
		},
		{
			name:     "null keepalive treated as unset",
			conf:     map[string]any{"keepalive": nil},
			expected: defaultClientKeepalive(),
		},
		{
			name:     "keepalive disabled via enabled false",
			conf:     map[string]any{"keepalive": map[string]any{"enabled": false}},
			expected: configoptional.None[KeepaliveClientConfig](),
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.True(t, cfg.DisableKeepAlives)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name:     "disable_keep_alives only — section stays unset",
			conf:     map[string]any{"disable_keep_alives": true},
			expected: defaultClientKeepalive(),
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.True(t, cfg.DisableKeepAlives)
				assert.Len(t, cfg.deprecationWarnings, 1)
			},
		},
		{
			name:     "null keepalive + deprecated field — no conflict",
			conf:     map[string]any{"keepalive": nil, "idle_conn_timeout": "60s"},
			expected: defaultClientKeepalive(),
			verifyConfig: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, 60*time.Second, cfg.IdleConnTimeout)
			},
		},
		{
			name: "keepalive section + no-op deprecated value — no conflict",
			conf: map[string]any{"keepalive": map[string]any{}, "idle_conn_timeout": "0s"},
			expected: configoptional.Some(KeepaliveClientConfig{
				IdleConnTimeout: 90 * time.Second,
				MaxIdleConns:    100,
			}),
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
			err := confmap.NewFromStringMap(tt.conf).Unmarshal(&cfg)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "keepalive")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Keepalive)
			if tt.verifyConfig != nil {
				tt.verifyConfig(t, &cfg)
			}
		})
	}
}

func TestClientConfigDeprecatedWarningsLogged(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "client/legacy_all_fields.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultClientConfig()
	require.NoError(t, cm.Unmarshal(&cfg))

	core, observed := observer.New(zapcore.WarnLevel)
	settings := component.TelemetrySettings{Logger: zap.New(core)}
	_, err = cfg.ToClient(t.Context(), nil, settings)
	require.NoError(t, err)

	entries := observed.All()
	require.Len(t, entries, 3)
	for _, entry := range entries {
		assert.Equal(t, zapcore.WarnLevel, entry.Level)
		assert.Contains(t, entry.Message, "deprecated")
	}
}

func TestClientConfigMixedFieldsError(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "client/mixed_fields_error.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultClientConfig()
	err = cm.Unmarshal(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "keepalive")
}

func TestClientConfigNewKeepalive(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "client/new_keepalive.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultClientConfig()
	require.NoError(t, cm.Unmarshal(&cfg))

	assert.Equal(t, configoptional.Some(KeepaliveClientConfig{
		IdleConnTimeout:     60 * time.Second,
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
	}), cfg.Keepalive)
	assert.Empty(t, cfg.deprecationWarnings)
}

func TestClientConfigKeepaliveDisabled(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "client/keepalive_disabled.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultClientConfig()
	require.NoError(t, cm.Unmarshal(&cfg))

	assert.Equal(t, configoptional.None[KeepaliveClientConfig](), cfg.Keepalive)

	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = nil
	settings.TracerProvider = nil
	client, err := cfg.ToClient(t.Context(), nil, settings)
	require.NoError(t, err)
	assert.True(t, client.Transport.(*http.Transport).DisableKeepAlives)
}

func TestClientConfigDeprecatedDisableKeepAlives(t *testing.T) {
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = nil
	settings.TracerProvider = nil

	core, observed := observer.New(zapcore.WarnLevel)
	settings.Logger = zap.New(core)

	cfg := NewDefaultClientConfig()
	conf := confmap.NewFromStringMap(map[string]any{"disable_keep_alives": true})
	require.NoError(t, conf.Unmarshal(&cfg))

	client, err := cfg.ToClient(t.Context(), nil, settings)
	require.NoError(t, err)
	assert.True(t, client.Transport.(*http.Transport).DisableKeepAlives)

	entries := observed.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "deprecated")
}

// ---- ServerConfig ----

func TestServerConfigUnmarshalKeepalive(t *testing.T) {
	tests := []struct {
		name         string
		conf         map[string]any
		expectError  bool
		expected     configoptional.Optional[KeepaliveServerConfig]
		verifyConfig func(*testing.T, *ServerConfig)
	}{
		{
			name:     "no keepalive config — section stays unset",
			conf:     map[string]any{},
			expected: defaultServerKeepalive(),
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Equal(t, 1*time.Minute, cfg.IdleTimeout)
				assert.True(t, cfg.KeepAlivesEnabled)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name:     "new keepalive only",
			conf:     map[string]any{"keepalive": map[string]any{"idle_timeout": "2m"}},
			expected: configoptional.Some(KeepaliveServerConfig{IdleTimeout: 2 * time.Minute}),
		},
		{
			name:     "deprecated idle_timeout only — section stays unset",
			conf:     map[string]any{"idle_timeout": "2m"},
			expected: defaultServerKeepalive(),
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Equal(t, 2*time.Minute, cfg.IdleTimeout)
				assert.Len(t, cfg.deprecationWarnings, 1)
			},
		},
		{
			name:     "null keepalive treated as unset",
			conf:     map[string]any{"keepalive": nil},
			expected: defaultServerKeepalive(),
		},
		{
			name:     "keepalive disabled via enabled false",
			conf:     map[string]any{"keepalive": map[string]any{"enabled": false}},
			expected: configoptional.None[KeepaliveServerConfig](),
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.False(t, cfg.KeepAlivesEnabled)
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name:     "keep_alives_enabled false only — section stays unset",
			conf:     map[string]any{"keep_alives_enabled": false},
			expected: defaultServerKeepalive(),
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.False(t, cfg.KeepAlivesEnabled)
				assert.Len(t, cfg.deprecationWarnings, 1)
			},
		},
		{
			name:     "keep_alives_enabled true only — no-op",
			conf:     map[string]any{"keep_alives_enabled": true},
			expected: defaultServerKeepalive(),
			verifyConfig: func(t *testing.T, cfg *ServerConfig) {
				assert.Empty(t, cfg.deprecationWarnings)
			},
		},
		{
			name: "keep_alives_enabled true + keepalive section — no conflict",
			conf: map[string]any{
				"keep_alives_enabled": true,
				"keepalive":           map[string]any{"idle_timeout": "2m"},
			},
			expected: configoptional.Some(KeepaliveServerConfig{IdleTimeout: 2 * time.Minute}),
		},
		{
			name:     "null keepalive + deprecated field — no conflict",
			conf:     map[string]any{"keepalive": nil, "idle_timeout": "2m"},
			expected: defaultServerKeepalive(),
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
			err := confmap.NewFromStringMap(tt.conf).Unmarshal(&cfg)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "keepalive")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Keepalive)
			if tt.verifyConfig != nil {
				tt.verifyConfig(t, &cfg)
			}
		})
	}
}

func TestServerConfigDeprecatedWarningsLogged(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "server/legacy_with_idle_timeout.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultServerConfig()
	require.NoError(t, cm.Unmarshal(&cfg))

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

func TestServerConfigMixedFieldsError(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "server/mixed_fields_error.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultServerConfig()
	err = cm.Unmarshal(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "keepalive")
}

func TestServerConfigNewKeepalive(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "server/new_keepalive.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultServerConfig()
	require.NoError(t, cm.Unmarshal(&cfg))

	assert.Equal(t, configoptional.Some(KeepaliveServerConfig{IdleTimeout: 90 * time.Second}), cfg.Keepalive)
	assert.Empty(t, cfg.deprecationWarnings)
}

func TestServerConfigKeepaliveDisabled(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "server/keepalive_disabled.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultServerConfig()
	require.NoError(t, cm.Unmarshal(&cfg))

	assert.Equal(t, configoptional.None[KeepaliveServerConfig](), cfg.Keepalive)
	assert.False(t, cfg.KeepAlivesEnabled)
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
	assert.Equal(t, defaultClientKeepalive(), cfg.ClientConfig.Keepalive)
	assert.Equal(t, 60*time.Second, cfg.ClientConfig.IdleConnTimeout)
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
	assert.Equal(t, defaultServerKeepalive(), cfg.ServerConfig.Keepalive)
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
