// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
)

func setFeatureGateForTest(tb testing.TB, gate *featuregate.Gate, enabled bool) func() {
	originalValue := gate.IsEnabled()
	require.NoError(tb, featuregate.GlobalRegistry().Set(gate.ID(), enabled))
	return func() {
		require.NoError(tb, featuregate.GlobalRegistry().Set(gate.ID(), originalValue))
	}
}
