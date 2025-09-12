// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/telemetryimpl"
)

func setGateForTest(t *testing.T, enabled bool) {
	initial := telemetryimpl.NewPipelineTelemetryGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(telemetryimpl.NewPipelineTelemetryGate.ID(), enabled))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(telemetryimpl.NewPipelineTelemetryGate.ID(), initial))
	})
}
