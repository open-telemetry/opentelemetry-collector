// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_ValidateItemBasedBatching(t *testing.T) {
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

func TestConfig_ValidateBytesBasedBatching(t *testing.T) {
	cfg := Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		MinSizeConfig: MinSizeConfig{
			MinSizeBytes: 20000,
		},
		MaxSizeConfig: MaxSizeConfig{
			MaxSizeBytes: 20001,
		},
	}
	require.NoError(t, cfg.Validate())

	cfg = Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		MinSizeConfig: MinSizeConfig{
			MinSizeBytes: -1,
		},
		MaxSizeConfig: MaxSizeConfig{
			MaxSizeBytes: 20001,
		},
	}
	require.EqualError(t, cfg.Validate(), "min_size_bytes must be greater than or equal to zero")

	cfg = Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		MinSizeConfig: MinSizeConfig{
			MinSizeBytes: 20000,
		},
		MaxSizeConfig: MaxSizeConfig{
			MaxSizeBytes: -1,
		},
	}
	require.EqualError(t, cfg.Validate(), "max_size_bytes must be greater than or equal to zero")

	cfg = Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		MinSizeConfig: MinSizeConfig{
			MinSizeBytes: 20001,
		},
		MaxSizeConfig: MaxSizeConfig{
			MaxSizeBytes: 20000,
		},
	}
	assert.EqualError(t, cfg.Validate(), "max_size_bytes must be greater than or equal to min_size_bytes")
}

func TestConfig_ItemSizeLimitAndByteSizeLimitShouldNotBeBothSet(t *testing.T) {
	cfg := Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		MinSizeConfig: MinSizeConfig{
			MinSizeItems: 20000,
		},
		MaxSizeConfig: MaxSizeConfig{
			MaxSizeBytes: 20001,
		},
	}
	require.EqualError(t, cfg.Validate(), "byte size limit and item limit cannot be specified at the same time")

	cfg = Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		MinSizeConfig: MinSizeConfig{
			MinSizeBytes: 20000,
		},
		MaxSizeConfig: MaxSizeConfig{
			MaxSizeItems: 20001,
		},
	}
	require.EqualError(t, cfg.Validate(), "byte size limit and item limit cannot be specified at the same time")

	cfg = Config{
		Enabled:       true,
		FlushTimeout:  200 * time.Millisecond,
		MinSizeConfig: MinSizeConfig{},
		MaxSizeConfig: MaxSizeConfig{
			MaxSizeItems: 20001,
			MaxSizeBytes: 20001,
		},
	}
	require.EqualError(t, cfg.Validate(), "byte size limit and item limit cannot be specified at the same time")

	cfg = Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		MinSizeConfig: MinSizeConfig{
			MinSizeItems: 20001,
			MinSizeBytes: 20001,
		},
		MaxSizeConfig: MaxSizeConfig{},
	}
	require.EqualError(t, cfg.Validate(), "byte size limit and item limit cannot be specified at the same time")
}
