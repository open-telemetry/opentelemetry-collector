// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap/confmaptest"
)

// TestUnmarshalConfig loads testdata/config.yaml, which documents every default
// value, and asserts it unmarshals to the factory's default config. This guards
// against drift between the documented defaults and createDefaultConfig.
func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	cfg := NewFactory().CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	assert.Equal(t, createDefaultConfig(), cfg)
}
