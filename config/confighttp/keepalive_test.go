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
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

// ---- ClientConfig ----

func TestClientConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ClientConfig
		expectError bool
	}{
		{
			name: "new keepalive only",
			cfg: func() ClientConfig {
				c := NewDefaultClientConfig()
				c.Keepalive = configoptional.Some(KeepaliveClientConfig{IdleConnTimeout: 60 * time.Second})
				return c
			}(),
		},
		{
			name: "deprecated fields only — no conflict",
			cfg: func() ClientConfig {
				c := NewDefaultClientConfig()
				c.IdleConnTimeout = 60 * time.Second
				return c
			}(),
		},
		{
			name: "keepalive disabled via None",
			cfg: func() ClientConfig {
				c := NewDefaultClientConfig()
				c.Keepalive = configoptional.None[KeepaliveClientConfig]()
				return c
			}(),
		},
		{
			name: "disable_keep_alives without explicit keepalive section — no conflict",
			cfg: func() ClientConfig {
				c := NewDefaultClientConfig()
				c.DisableKeepAlives = true
				return c
			}(),
		},
		{
			name: "conflict: keepalive section + idle_conn_timeout",
			cfg: func() ClientConfig {
				c := NewDefaultClientConfig()
				c.Keepalive = configoptional.Some(KeepaliveClientConfig{})
				c.IdleConnTimeout = 60 * time.Second
				return c
			}(),
			expectError: true,
		},
		{
			name: "conflict: keepalive section + disable_keep_alives",
			cfg: func() ClientConfig {
				c := NewDefaultClientConfig()
				c.Keepalive = configoptional.Some(KeepaliveClientConfig{})
				c.DisableKeepAlives = true
				return c
			}(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "keepalive")
			} else {
				require.NoError(t, err)
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
	require.NotEmpty(t, entries)
	for _, entry := range entries {
		assert.Equal(t, zapcore.WarnLevel, entry.Level)
		assert.Contains(t, entry.Message, "deprecated")
	}
}

func TestClientConfigMixedFieldsError(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "client/mixed_fields_error.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultClientConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Error(t, cfg.Validate())
}

func TestClientConfigNewKeepalive(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "client/new_keepalive.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultClientConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	require.NoError(t, cfg.Validate())

	assert.Equal(t, configoptional.Some(KeepaliveClientConfig{
		IdleConnTimeout:     60 * time.Second,
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
	}), cfg.Keepalive)
}

func TestClientConfigKeepaliveDisabled(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "client/keepalive_disabled.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultClientConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	require.NoError(t, cfg.Validate())

	assert.True(t, cfg.Keepalive.IsNone())
}

// ---- ServerConfig ----

func TestServerConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ServerConfig
		expectError bool
	}{
		{
			name: "new keepalive only",
			cfg: func() ServerConfig {
				c := NewDefaultServerConfig()
				c.Keepalive = configoptional.Some(KeepaliveServerConfig{IdleTimeout: 2 * time.Minute})
				return c
			}(),
		},
		{
			name: "deprecated idle_timeout only — no conflict",
			cfg: func() ServerConfig {
				c := NewDefaultServerConfig()
				c.IdleTimeout = 2 * time.Minute
				return c
			}(),
		},
		{
			name: "keepalive disabled via None",
			cfg: func() ServerConfig {
				c := NewDefaultServerConfig()
				c.Keepalive = configoptional.None[KeepaliveServerConfig]()
				return c
			}(),
		},
		{
			name: "deprecated keep_alives_enabled false only — no conflict",
			cfg: func() ServerConfig {
				c := NewDefaultServerConfig()
				disabled := false
				c.KeepAlivesEnabled = &disabled
				return c
			}(),
		},
		{
			name: "conflict: keepalive section + idle_timeout",
			cfg: func() ServerConfig {
				c := NewDefaultServerConfig()
				c.Keepalive = configoptional.Some(KeepaliveServerConfig{})
				c.IdleTimeout = 2 * time.Minute
				return c
			}(),
			expectError: true,
		},
		{
			name: "conflict: keepalive section + keep_alives_enabled",
			cfg: func() ServerConfig {
				c := NewDefaultServerConfig()
				c.Keepalive = configoptional.Some(KeepaliveServerConfig{})
				disabled := false
				c.KeepAlivesEnabled = &disabled
				return c
			}(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "keepalive")
			} else {
				require.NoError(t, err)
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

	entries := observed.All()
	require.NotEmpty(t, entries)
	for _, entry := range entries {
		assert.Equal(t, zapcore.WarnLevel, entry.Level)
		assert.Contains(t, entry.Message, "deprecated")
	}
}

func TestServerConfigMixedFieldsError(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "server/mixed_fields_error.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultServerConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Error(t, cfg.Validate())
}

func TestServerConfigNewKeepalive(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "server/new_keepalive.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultServerConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	require.NoError(t, cfg.Validate())

	assert.Equal(t, configoptional.Some(KeepaliveServerConfig{IdleTimeout: 90 * time.Second}), cfg.Keepalive)
}

func TestServerConfigKeepaliveDisabled(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", "server/keepalive_disabled.yaml"))
	require.NoError(t, err)

	cfg := NewDefaultServerConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	require.NoError(t, cfg.Validate())

	assert.True(t, cfg.Keepalive.IsNone())
}

func TestClientConfigDeprecatedDisableKeepAlives(t *testing.T) {
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = nil
	settings.TracerProvider = nil

	core, observed := observer.New(zapcore.WarnLevel)
	settings.Logger = zap.New(core)

	cfg := NewDefaultClientConfig()
	cfg.DisableKeepAlives = true
	client, err := cfg.ToClient(t.Context(), nil, settings)
	require.NoError(t, err)
	transport := client.Transport.(*http.Transport)
	assert.True(t, transport.DisableKeepAlives)

	entries := observed.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "deprecated")
}

func TestServerConfigDeprecatedKeepAlivesDisabled(t *testing.T) {
	core, observed := observer.New(zapcore.WarnLevel)
	settings := component.TelemetrySettings{Logger: zap.New(core)}

	cfg := NewDefaultServerConfig()
	disabled := false
	cfg.KeepAlivesEnabled = &disabled

	srv, err := cfg.ToServer(t.Context(), nil, settings, http.NewServeMux())
	require.NoError(t, err)
	require.NotNil(t, srv)

	entries := observed.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "deprecated")
}
