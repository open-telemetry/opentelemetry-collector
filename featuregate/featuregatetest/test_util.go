// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregatetest // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
)

// SetGate sets the value to the given gate. Also, it installs a cleanup function to restore
// the gate to the initial value when the test is done.
func SetGate(tb testing.TB, gate *featuregate.Gate, enabled bool) {
	originalValue := gate.IsEnabled()
	require.NoError(tb, featuregate.GlobalRegistry().Set(gate.ID(), enabled))
	tb.Cleanup(func() {
		require.NoError(tb, featuregate.GlobalRegistry().Set(gate.ID(), originalValue))
	})
}
