// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testutil // import "go.opentelemetry.io/collector/internal/testutil"

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
)

// SetFeatureGate sets a global feature gate for the duration of the test.
func SetFeatureGate(tb testing.TB, id string, enabled bool) {
	tb.Helper()
	setFeatureGate(tb, featuregate.GlobalRegistry(), id, enabled)
}

func setFeatureGate(tb testing.TB, registry *featuregate.Registry, id string, enabled bool) {
	var gate *featuregate.Gate
	registry.VisitAll(func(candidate *featuregate.Gate) {
		if candidate.ID() == id {
			gate = candidate
		}
	})
	require.NotNil(tb, gate, "feature gate %q is not registered", id)

	initialValue := gate.IsEnabled()
	require.NoError(tb, registry.Set(id, enabled))
	tb.Cleanup(func() {
		require.NoError(tb, registry.Set(id, initialValue))
	})
}
