// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"testing"

	"github.com/stretchr/testify/require"
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
