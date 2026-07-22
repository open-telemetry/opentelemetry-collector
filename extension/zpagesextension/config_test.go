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

	expectedServerConfig := confighttp.NewDefaultServerConfig()
	expectedServerConfig.NetAddr.Endpoint = "localhost:56888"

	assert.Equal(t, &Config{ServerConfig: expectedServerConfig}, cfg)
}

// Regression test: fields declared next to the squash-embedded ServerConfig must
// still be decoded now that ServerConfig implements confmap.Unmarshaler.
func TestUnmarshalConfigWithExpvar(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"endpoint": "localhost:56888",
		"expvar":   map[string]any{"enabled": true},
	})
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))

	zCfg := cfg.(*Config)
	assert.Equal(t, "localhost:56888", zCfg.ServerConfig.NetAddr.Endpoint)
	assert.True(t, zCfg.Expvar.Enabled)
}
