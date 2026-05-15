// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
)

func TestUnmarshalClientDeprecatedError(t *testing.T) {
	conf := confmap.NewFromStringMap(map[string]any{"timeout": "not-a-duration"})
	cfg := NewDefaultClientConfig()
	require.Error(t, unmarshalClientDeprecated(&cfg, conf))
}

func TestUnmarshalServerDeprecatedError(t *testing.T) {
	conf := confmap.NewFromStringMap(map[string]any{"read_timeout": "not-a-duration"})
	cfg := NewDefaultServerConfig()
	require.Error(t, unmarshalServerDeprecated(&cfg, conf))
}

func TestRenamedFieldLog(t *testing.T) {
	core, observed := observer.New(zapcore.WarnLevel)
	f := renamedField{old: "old_name", new: "new::name"}
	f.Log(zap.New(core))

	entries := observed.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "Use of deprecated configuration option `old_name`, use `new::name` instead.", entries[0].Message)
	assert.Equal(t, zapcore.WarnLevel, entries[0].Level)
}

func TestCollectLegacyKeepaliveWarnings(t *testing.T) {
	fields := []renamedField{
		{"idle_conn_timeout", "keepalive::idle_conn_timeout"},
		{"max_idle_conns", "keepalive::max_idle_conns"},
	}

	tests := []struct {
		name        string
		input       map[string]any
		want        []renamedField
		expectError bool
	}{
		{
			name:  "empty",
			input: map[string]any{},
			want:  nil,
		},
		{
			name:  "unrelated field",
			input: map[string]any{"endpoint": "http://localhost"},
			want:  nil,
		},
		{
			name:  "single legacy field",
			input: map[string]any{"idle_conn_timeout": "60s"},
			want:  []renamedField{{"idle_conn_timeout", "keepalive::idle_conn_timeout"}},
		},
		{
			name: "multiple legacy fields",
			input: map[string]any{
				"idle_conn_timeout": "60s",
				"max_idle_conns":    50,
			},
			want: []renamedField{
				{"idle_conn_timeout", "keepalive::idle_conn_timeout"},
				{"max_idle_conns", "keepalive::max_idle_conns"},
			},
		},
		{
			name:  "new section only",
			input: map[string]any{"keepalive": map[string]any{"idle_conn_timeout": "60s"}},
			want:  nil,
		},
		{
			name: "legacy and new section together",
			input: map[string]any{
				"idle_conn_timeout": "60s",
				"keepalive":         map[string]any{"idle_conn_timeout": "30s"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := confmap.NewFromStringMap(tt.input)
			got, err := collectLegacyKeepaliveWarnings(conf, fields, "testConfig")
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "testConfig")
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplyLegacyClientKeepalive(t *testing.T) {
	defaultKA := func() configoptional.Optional[KeepaliveClientConfig] {
		return configoptional.Some(KeepaliveClientConfig{
			IdleConnTimeout: 90 * time.Second,
			MaxIdleConns:    100,
		})
	}

	tests := []struct {
		name    string
		input   map[string]any
		legacy  legacyClientKeepaliveFields
		initial configoptional.Optional[KeepaliveClientConfig]
		want    configoptional.Optional[KeepaliveClientConfig]
	}{
		{
			name:    "no legacy fields keeps existing",
			input:   map[string]any{},
			initial: defaultKA(),
			want:    defaultKA(),
		},
		{
			name:    "disable_keep_alives clears keepalive",
			input:   map[string]any{"disable_keep_alives": true},
			legacy:  legacyClientKeepaliveFields{DisableKeepAlives: true},
			initial: defaultKA(),
			want:    configoptional.None[KeepaliveClientConfig](),
		},
		{
			name:    "idle_conn_timeout overrides only that field",
			input:   map[string]any{"idle_conn_timeout": "30s"},
			legacy:  legacyClientKeepaliveFields{IdleConnTimeout: 30 * time.Second, MaxIdleConns: 100},
			initial: defaultKA(),
			want: configoptional.Some(KeepaliveClientConfig{
				IdleConnTimeout: 30 * time.Second,
				MaxIdleConns:    100,
			}),
		},
		{
			name:  "all legacy fields override",
			input: map[string]any{"idle_conn_timeout": "30s", "max_idle_conns": 5, "max_idle_conns_per_host": 2},
			legacy: legacyClientKeepaliveFields{
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConns:        5,
				MaxIdleConnsPerHost: 2,
			},
			initial: defaultKA(),
			want: configoptional.Some(KeepaliveClientConfig{
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConns:        5,
				MaxIdleConnsPerHost: 2,
			}),
		},
		{
			name:    "initializes empty keepalive when none and not disabled",
			input:   map[string]any{},
			initial: configoptional.None[KeepaliveClientConfig](),
			want:    configoptional.Some(KeepaliveClientConfig{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := confmap.NewFromStringMap(tt.input)
			ka := tt.initial
			applyLegacyClientKeepalive(conf, &ka, tt.legacy)
			assert.Equal(t, tt.want, ka)
		})
	}
}

func TestApplyLegacyServerKeepalive(t *testing.T) {
	defaultKA := func() configoptional.Optional[KeepaliveServerConfig] {
		return configoptional.Some(KeepaliveServerConfig{IdleTimeout: 60 * time.Second})
	}

	tests := []struct {
		name    string
		input   map[string]any
		legacy  legacyServerKeepaliveFields
		initial configoptional.Optional[KeepaliveServerConfig]
		want    configoptional.Optional[KeepaliveServerConfig]
	}{
		{
			name:    "keep_alives_enabled=true keeps existing",
			input:   map[string]any{},
			legacy:  legacyServerKeepaliveFields{KeepAlivesEnabled: true},
			initial: defaultKA(),
			want:    defaultKA(),
		},
		{
			name:    "keep_alives_enabled=false clears keepalive",
			input:   map[string]any{"keep_alives_enabled": false},
			legacy:  legacyServerKeepaliveFields{KeepAlivesEnabled: false},
			initial: defaultKA(),
			want:    configoptional.None[KeepaliveServerConfig](),
		},
		{
			name:    "idle_timeout overrides",
			input:   map[string]any{"idle_timeout": "120s"},
			legacy:  legacyServerKeepaliveFields{IdleTimeout: 120 * time.Second, KeepAlivesEnabled: true},
			initial: defaultKA(),
			want:    configoptional.Some(KeepaliveServerConfig{IdleTimeout: 120 * time.Second}),
		},
		{
			name:    "initializes empty keepalive when none and enabled",
			input:   map[string]any{},
			legacy:  legacyServerKeepaliveFields{KeepAlivesEnabled: true},
			initial: configoptional.None[KeepaliveServerConfig](),
			want:    configoptional.Some(KeepaliveServerConfig{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := confmap.NewFromStringMap(tt.input)
			ka := tt.initial
			applyLegacyServerKeepalive(conf, &ka, tt.legacy)
			assert.Equal(t, tt.want, ka)
		})
	}
}
