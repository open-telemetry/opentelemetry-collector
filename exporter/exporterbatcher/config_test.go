// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

func TestValidateConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	require.NoError(t, cfg.Validate())

	cfg = NewDefaultConfig()
	cfg.FlushTimeout = 0
	require.EqualError(t, cfg.Validate(), "`flush_timeout` must be greater than zero")
}

func TestValidateSizeConfig(t *testing.T) {
	cfg := SizeConfig{
		Sizer:   SizerTypeItems,
		MaxSize: -100,
		MinSize: 100,
	}
	require.EqualError(t, cfg.Validate(), "`max_size` must be greater than or equal to zero")

	cfg = SizeConfig{
		Sizer:   SizerTypeItems,
		MaxSize: 100,
		MinSize: -100,
	}
	require.EqualError(t, cfg.Validate(), "`min_size` must be greater than or equal to zero")

	cfg = SizeConfig{
		Sizer:   SizerTypeItems,
		MaxSize: 100,
		MinSize: 200,
	}
	require.EqualError(t, cfg.Validate(), "`max_size` must be greater than or equal to mix_size")
}

func TestUnmarshalDeprecatedConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	require.NoError(t, cfg.Unmarshal(confmap.NewFromStringMap(map[string]any{
		"enabled":        true,
		"flush_timeout":  200,
		"min_size_items": 111,
		"max_size_items": 222,
	})))
	require.NoError(t, cfg.Validate())
	require.Equal(t, Config{
		Enabled:      true,
		FlushTimeout: 200,
		SizeConfig: SizeConfig{
			Sizer:   SizerTypeItems,
			MinSize: 111,
			MaxSize: 222,
		},
		MinSizeConfig: MinSizeConfig{
			MinSizeItems: 111,
		},
		MaxSizeConfig: MaxSizeConfig{
			MaxSizeItems: 222,
		},
	}, cfg)
}
