// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
)

func TestSetFeatureGate(t *testing.T) {
	registry := featuregate.NewRegistry()
	gate := registry.MustRegister("test.featureGate", featuregate.StageAlpha)

	t.Run("enabled", func(t *testing.T) {
		setFeatureGate(t, registry, gate.ID(), true)
		require.True(t, gate.IsEnabled())
	})

	require.False(t, gate.IsEnabled())
}
