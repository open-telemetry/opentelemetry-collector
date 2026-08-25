// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/service/internal/metadata"
)

func setGateForTest(t *testing.T, enabled bool) {
	initial := metadata.TelemetryNewPipelineTelemetryFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.TelemetryNewPipelineTelemetryFeatureGate.ID(), enabled))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.TelemetryNewPipelineTelemetryFeatureGate.ID(), initial))
	})
}
