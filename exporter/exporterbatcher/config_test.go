// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
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
		SizerType: SizerType{
			Sizer: "bytes",
		},
		MaxSize: -100,
		MinSize: 100,
	}
	require.EqualError(t, cfg.Validate(), "max_size must be greater than or equal to zero")

	cfg = SizeConfig{
		SizerType: SizerType{
			Sizer: "bytes",
		},
		MaxSize: 100,
		MinSize: -100,
	}
	require.EqualError(t, cfg.Validate(), "min_size must be greater than or equal to zero")

	cfg = SizeConfig{
		SizerType: SizerType{
			Sizer: "bytes",
		},
		MaxSize: 100,
		MinSize: 200,
	}
	require.EqualError(t, cfg.Validate(), "max_size must be greater than or equal to mix_size")
}

func TestValidateSizer(t *testing.T) {
	cfg := SizerType{
		Sizer: "invalid",
	}
	require.EqualError(t, cfg.Validate(), "sizer should either be bytes or items")

	cfg.Sizer = "bytes"
	require.NoError(t, cfg.Validate())

	cfg.Sizer = "items"
	require.NoError(t, cfg.Validate())
}

func TestValidateConfigInvokedOnNestedFieldsAutomatically(t *testing.T) {
	cfg := SizeConfig{
		SizerType: SizerType{
			Sizer: "invalid",
		},
		MaxSize: 100,
		MinSize: 0,
	}
	require.EqualError(t, component.ValidateConfig(cfg), "sizer should either be bytes or items")
}
