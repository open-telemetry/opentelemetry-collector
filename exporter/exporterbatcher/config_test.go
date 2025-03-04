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

func TestUnmarshalDeprecatedFields(t *testing.T) {
	p111 := 111
	p222 := 222
	tests := []struct {
		name     string
		input    map[string]any
		expected Config
	}{
		{
			name: "only_deprecated_fields_used",
			input: map[string]any{
				"enabled":        true,
				"flush_timeout":  200,
				"min_size_items": 111,
				"max_size_items": 222,
			},
			expected: Config{
				Enabled:      true,
				FlushTimeout: 200,
				SizeConfig: SizeConfig{
					Sizer:   SizerTypeItems,
					MinSize: 111,
					MaxSize: 222,
				},
				MinSizeConfig: MinSizeConfig{
					MinSizeItems: &p111,
				},
				MaxSizeConfig: MaxSizeConfig{
					MaxSizeItems: &p222,
				},
			},
		},
		{
			name: "both_new_and_deprecated_fields_used",
			input: map[string]any{
				"enabled":        true,
				"flush_timeout":  200,
				"min_size_items": 111,
				"max_size_items": 222,
				"min_size":       11,
				"max_size":       22,
			},
			expected: Config{
				Enabled:      true,
				FlushTimeout: 200,
				SizeConfig: SizeConfig{
					Sizer:   SizerTypeItems,
					MinSize: 11,
					MaxSize: 22,
				},
				MinSizeConfig: MinSizeConfig{
					MinSizeItems: &p111,
				},
				MaxSizeConfig: MaxSizeConfig{
					MaxSizeItems: &p222,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := NewDefaultConfig()
			require.NoError(t, cfg.Unmarshal(confmap.NewFromStringMap(test.input)))
			require.Equal(t, test.expected, cfg)
			require.NoError(t, cfg.Validate())
		})
	}
}
