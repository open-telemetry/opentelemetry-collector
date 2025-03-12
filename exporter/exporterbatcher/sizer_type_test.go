// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/confmap"
)

func TestSizeTypeUnmarshaler(t *testing.T) {
	var rawConf map[string]any
	cfg := NewDefaultConfig()

	require.NoError(t, yaml.Unmarshal([]byte(`sizer: items`), &rawConf))
	require.NoError(t, confmap.NewFromStringMap(rawConf).Unmarshal(&cfg))
	require.NoError(t, cfg.Validate())

	require.NoError(t, yaml.Unmarshal([]byte(`sizer: 'items'`), &rawConf))
	require.NoError(t, confmap.NewFromStringMap(rawConf).Unmarshal(&cfg))
	require.NoError(t, cfg.Validate())

	require.NoError(t, yaml.Unmarshal([]byte(`sizer: invalid`), &rawConf))
	require.ErrorContains(t,
		confmap.NewFromStringMap(rawConf).Unmarshal(&cfg),
		"decoding failed due to the following error(s):\n\nerror decoding 'sizer': invalid sizer: \"invalid\"")
}
