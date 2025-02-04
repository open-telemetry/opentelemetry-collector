// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/confmap"
)

func TestValidateConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	require.NoError(t, cfg.Validate())

	cfg.MinSizeItems = -1
	require.EqualError(t, cfg.Validate(), "min_size_items must be greater than or equal to zero")

	cfg = NewDefaultConfig()
	cfg.FlushTimeout = 0
	require.EqualError(t, cfg.Validate(), "timeout must be greater than zero")

	cfg.MaxSizeItems = -1
	require.EqualError(t, cfg.Validate(), "max_size_items must be greater than or equal to zero")

	cfg = NewDefaultConfig()
	cfg.MaxSizeItems = 20000
	cfg.MinSizeItems = 20001
	assert.EqualError(t, cfg.Validate(), "max_size_items must be greater than or equal to min_size_items")
}

func TestValidateSizeConfig(t *testing.T) {
	cfg := SizeConfig{
		Sizer:   MustNewSizer("bytes"),
		MaxSize: -100,
		MinSize: 100,
	}
	require.EqualError(t, cfg.Validate(), "max_size must be greater than or equal to zero")

	cfg = SizeConfig{
		Sizer:   MustNewSizer("bytes"),
		MaxSize: 100,
		MinSize: -100,
	}
	require.EqualError(t, cfg.Validate(), "min_size must be greater than or equal to zero")

	cfg = SizeConfig{
		Sizer:   MustNewSizer("bytes"),
		MaxSize: 100,
		MinSize: 200,
	}
	require.EqualError(t, cfg.Validate(), "max_size must be greater than or equal to mix_size")
}

func TestSizeUnmarshaler(t *testing.T) {
	var rawConf map[string]any
	cfg := NewDefaultConfig()

	require.NoError(t, yaml.Unmarshal([]byte(`sizer: bytes`), &rawConf))
	require.NoError(t, confmap.NewFromStringMap(rawConf).Unmarshal(&cfg))
	require.NoError(t, cfg.Validate())

	require.NoError(t, yaml.Unmarshal([]byte(`sizer: "bytes"`), &rawConf))
	require.NoError(t, confmap.NewFromStringMap(rawConf).Unmarshal(&cfg))
	require.NoError(t, cfg.Validate())

	require.NoError(t, yaml.Unmarshal([]byte(`sizer: items`), &rawConf))
	require.NoError(t, confmap.NewFromStringMap(rawConf).Unmarshal(&cfg))
	require.NoError(t, cfg.Validate())

	require.NoError(t, yaml.Unmarshal([]byte(`sizer: 'items'`), &rawConf))
	require.NoError(t, confmap.NewFromStringMap(rawConf).Unmarshal(&cfg))
	require.NoError(t, cfg.Validate())

	require.NoError(t, yaml.Unmarshal([]byte(`sizer: invalid`), &rawConf))
	require.EqualError(t,
		confmap.NewFromStringMap(rawConf).Unmarshal(&cfg),
		"decoding failed due to the following error(s):\n\nerror decoding 'sizer': invalid sizer: \"invalid\"")
}
