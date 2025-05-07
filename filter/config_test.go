// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func readTestdataConfigYamls(t *testing.T, filename string) map[string][]Config {
	testFile := filepath.Join("testdata", filename)
	v, err := confmaptest.LoadConf(testFile)
	require.NoError(t, err)

	cfgs := map[string][]Config{}
	require.NoErrorf(t, v.Unmarshal(&cfgs, confmap.WithIgnoreUnused()), "unable to unmarshal yaml from file %v", testFile)
	return cfgs
}

func TestConfig(t *testing.T) {
	actualConfigs := readTestdataConfigYamls(t, "config.yaml")
	expectedConfigs := map[string][]Config{
		"regexp/default": {
			{
				Regex: "one|two",
			},
		},
		"strict/default": {
			{
				Strict: "strict",
			},
		},
	}

	for testName, actualCfg := range actualConfigs {
		t.Run(testName, func(t *testing.T) {
			expCfg, ok := expectedConfigs[testName]
			assert.True(t, ok)
			assert.Equal(t, expCfg, actualCfg)

			for _, cfg := range actualCfg {
				require.NoError(t, cfg.Validate())
			}
			fs := CreateFilter(actualCfg)
			assert.NotNil(t, fs)
		})
	}
}

func TestMatches(t *testing.T) {
	cfg := []Config{
		{
			Strict: "a",
		},
		{
			Strict: "b",
		},
		{
			Regex: "a|b|c",
		},
	}

	for _, c := range cfg {
		require.NoError(t, c.Validate())
	}
	fs := CreateFilter(cfg)

	assert.True(t, fs.Matches("a"))
	assert.True(t, fs.Matches("b"))
	assert.True(t, fs.Matches("c"))
}

func TestConfigInvalid(t *testing.T) {
	actualConfigs := readTestdataConfigYamls(t, "config_invalid.yaml")
	expectedConfigs := map[string][]Config{
		"invalid/regexp": {
			{
				Regex: "(.*[",
			},
		},
		"invalid/config_empty": {
			{
				Regex:  "",
				Strict: "",
			},
		},
		"invalid/config_both_set": {
			{
				Regex:  "1",
				Strict: "1",
			},
		},
	}

	for testName, actualCfg := range actualConfigs {
		t.Run(testName, func(t *testing.T) {
			expCfg, ok := expectedConfigs[testName]
			assert.True(t, ok)
			assert.Equal(t, expCfg, actualCfg)

			for _, cfg := range actualCfg {
				assert.Error(t, cfg.Validate())
			}
		})
	}
}
