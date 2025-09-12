// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
	servicetelemetry "go.opentelemetry.io/collector/service/internal/telemetry"
)

func setGateForTest(t *testing.T, enabled bool) {
	initial := servicetelemetry.NewPipelineTelemetryGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(servicetelemetry.NewPipelineTelemetryGate.ID(), enabled))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(servicetelemetry.NewPipelineTelemetryGate.ID(), initial))
	})
}
