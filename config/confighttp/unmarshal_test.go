// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestClientConfigUnmarshal(t *testing.T) {
	tests := []struct {
		name           string
		configFile     string
		expectError    bool
		expectWarnings bool
		want           configoptional.Optional[KeepaliveClientConfig]
	}{
		{
			name:           "legacy_all_fields",
			configFile:     "client/legacy_all_fields.yaml",
			expectWarnings: true,
			want: configoptional.Some(KeepaliveClientConfig{
				IdleConnTimeout:     90 * time.Second,
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
			}),
		},
		{
			name:           "legacy_partial_fields",
			configFile:     "client/legacy_partial_fields.yaml",
			expectWarnings: true,
			want: configoptional.Some(KeepaliveClientConfig{
				IdleConnTimeout: 60 * time.Second,
				MaxIdleConns:    100,
			}),
		},
		{
			name:           "legacy_with_disable",
			configFile:     "client/legacy_with_disable.yaml",
			expectWarnings: true,
		},
		{
			name:       "new_keepalive",
			configFile: "client/new_keepalive.yaml",
			want: configoptional.Some(KeepaliveClientConfig{
				IdleConnTimeout:     60 * time.Second,
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 5,
			}),
		},
		{
			name:        "mixed_fields_error",
			configFile:  "client/mixed_fields_error.yaml",
			expectError: true,
		},
		{
			name:       "no_keepalive_fields",
			configFile: "client/no_keepalive_fields.yaml",
			want: configoptional.Some(KeepaliveClientConfig{
				IdleConnTimeout: 90 * time.Second,
				MaxIdleConns:    100,
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", tt.configFile))
			require.NoError(t, err)

			var cfg ClientConfig = NewDefaultClientConfig()
			err = cm.Unmarshal(&cfg)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.expectWarnings {
				require.NotEmpty(t, cfg.warnings)
			}
			assert.Equal(t, tt.want, cfg.Keepalive)
		})
	}
}

func TestServerConfigUnmarshal(t *testing.T) {
	tests := []struct {
		name           string
		configFile     string
		expectError    bool
		expectWarnings bool
		want           configoptional.Optional[KeepaliveServerConfig]
	}{
		{
			name:           "legacy_with_idle_timeout",
			configFile:     "server/legacy_with_idle_timeout.yaml",
			expectWarnings: true,
			want: configoptional.Some(KeepaliveServerConfig{
				IdleTimeout: 120 * time.Second,
			}),
		},
		{
			name:           "legacy_no_idle_timeout",
			configFile:     "server/legacy_no_idle_timeout.yaml",
			expectWarnings: true,
			want: configoptional.Some(KeepaliveServerConfig{
				IdleTimeout: 60 * time.Second,
			}),
		},
		{
			name:           "legacy_disabled",
			configFile:     "server/legacy_disabled.yaml",
			expectWarnings: true,
		},
		{
			name:       "new_keepalive",
			configFile: "server/new_keepalive.yaml",
			want: configoptional.Some(KeepaliveServerConfig{
				IdleTimeout: 90 * time.Second,
			}),
		},
		{
			name:        "mixed_fields_error",
			configFile:  "server/mixed_fields_error.yaml",
			expectError: true,
		},
		{
			name:       "no_keepalive_fields",
			configFile: "server/no_keepalive_fields.yaml",
			want: configoptional.Some(KeepaliveServerConfig{
				IdleTimeout: 60 * time.Second,
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config", tt.configFile))
			require.NoError(t, err)

			var cfg ServerConfig = NewDefaultServerConfig()
			err = cm.Unmarshal(&cfg)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.expectWarnings {
				require.NotEmpty(t, cfg.warnings)
			}

			assert.Equal(t, tt.want, cfg.Keepalive)
		})
	}
}
