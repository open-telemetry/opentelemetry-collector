// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, confmap.New().Unmarshal(&cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestInvalidConfig(t *testing.T) {
	assert.Error(t, (&Config{}).Validate())
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))

	expected := factory.CreateDefaultConfig().(*Config)
	expected.ServerConfig = confighttp.NewDefaultServerConfig()
	expected.ServerConfig.NetAddr.Endpoint = "localhost:56888"

	assert.Equal(t, expected, cfg)
}

func TestUnmarshalConfigWithSupportBundle(t *testing.T) {
	defaults := SupportBundleConfig{
		ArchiveName: "otelcol-support-bundle",
		Gatherers:   GatherersConfig{PProf: GathererConfig{Enabled: true}},
	}
	tests := []struct {
		name string
		raw  map[string]any
		// want is nil when the support bundle must be off.
		want *SupportBundleConfig
	}{
		{name: "absent", raw: map[string]any{}},
		{name: "empty section", raw: map[string]any{"support_bundle": nil}, want: &defaults},
		{name: "disabled", raw: map[string]any{"support_bundle": map[string]any{"enabled": false}}},
		{
			name: "settings",
			raw: map[string]any{"support_bundle": map[string]any{
				"archive_name": "my-bundle",
				"gatherers":    map[string]any{"pprof": map[string]any{"enabled": false}},
			}},
			want: &SupportBundleConfig{ArchiveName: "my-bundle"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewFactory().CreateDefaultConfig()
			require.NoError(t, confmap.NewFromStringMap(tt.raw).Unmarshal(&cfg))

			sb := cfg.(*Config).SupportBundle
			if tt.want == nil {
				assert.False(t, sb.HasValue())
				return
			}
			require.True(t, sb.HasValue())
			assert.Equal(t, tt.want, sb.Get())
		})
	}
}

func TestValidateSupportBundleArchiveName(t *testing.T) {
	tests := []struct {
		archiveName string
		valid       bool
	}{
		{archiveName: "otelcol-support-bundle", valid: true},
		{archiveName: "", valid: false},
		{archiveName: ".", valid: false},
		{archiveName: "..", valid: false},
		{archiveName: "a/b", valid: false},
		{archiveName: `a\b`, valid: false},
		{archiveName: "/abs", valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.archiveName, func(t *testing.T) {
			cfg := createDefaultConfig().(*Config)
			cfg.SupportBundle = configoptional.Some(SupportBundleConfig{ArchiveName: tt.archiveName})
			if tt.valid {
				assert.NoError(t, confmap.Validate(cfg))
			} else {
				assert.ErrorContains(t, confmap.Validate(cfg), `"archive_name" must be a valid file name with no path separators`)
			}
		})
	}

	// The extension does not check archive_name when the support bundle is off.
	cfg := createDefaultConfig().(*Config)
	cfg.SupportBundle = configoptional.None[SupportBundleConfig]()
	assert.NoError(t, confmap.Validate(cfg))
}
