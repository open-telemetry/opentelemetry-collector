// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_ValidateOldConfiguration(t *testing.T) {
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

func TestConfig(t *testing.T) {
	cfg := Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		SizeConfig: SizeConfig{
			Sizer:   "invalidsizer",
			MaxSize: 100,
			MinSize: 100,
		},
	}
	require.EqualError(t, cfg.Validate(), "sizer should either be bytes or items")

	cfg = Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		SizeConfig: SizeConfig{
			Sizer:   "bytes",
			MaxSize: -100,
			MinSize: 100,
		},
	}
	require.EqualError(t, cfg.Validate(), "max_size must be greater than or equal to zero")

	cfg = Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		SizeConfig: SizeConfig{
			Sizer:   "bytes",
			MaxSize: 100,
			MinSize: -100,
		},
	}
	require.EqualError(t, cfg.Validate(), "min_size must be greater than or equal to zero")

	cfg = Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		SizeConfig: SizeConfig{
			Sizer:   "bytes",
			MaxSize: 100,
			MinSize: 200,
		},
	}
	require.EqualError(t, cfg.Validate(), "max_size must be greater than or equal to mix_size")
}
