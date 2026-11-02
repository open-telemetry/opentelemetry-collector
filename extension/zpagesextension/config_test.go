// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confighttp"
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
	cm := confmap.NewFromStringMap(map[string]any{
		"support_bundle": map[string]any{
			"enabled":      true,
			"archive_name": "my-bundle",
			"gatherers": map[string]any{
				"pprof": map[string]any{"enabled": false},
			},
		},
	})
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))

	zCfg := cfg.(*Config)
	assert.True(t, zCfg.SupportBundle.Enabled)
	assert.Equal(t, "my-bundle", zCfg.SupportBundle.ArchiveName)
	assert.False(t, zCfg.SupportBundle.Gatherers.PProf.Enabled)
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
			cfg := SupportBundleConfig{Enabled: true, ArchiveName: tt.archiveName}
			if tt.valid {
				assert.NoError(t, cfg.Validate())
			} else {
				assert.ErrorContains(t, cfg.Validate(), `"archive_name" must be a valid file name with no path separators`)
			}
		})
	}

	// The extension ignores archive_name when the support bundle is disabled.
	assert.NoError(t, (&SupportBundleConfig{ArchiveName: ""}).Validate())
}
