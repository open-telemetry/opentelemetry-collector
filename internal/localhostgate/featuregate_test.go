// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package localhostgate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
)

func setFeatureGateForTest(t testing.TB, gate *featuregate.Gate, enabled bool) func() {
	originalValue := gate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(gate.ID(), enabled))
	return func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(gate.ID(), originalValue))
	}
}

func TestEndpointForPort(t *testing.T) {
	tests := []struct {
		port     int
		enabled  bool
		endpoint string
	}{
		{
			port:     4317,
			enabled:  false,
			endpoint: "0.0.0.0:4317",
		},
		{
			port:     4317,
			enabled:  true,
			endpoint: "localhost:4317",
		},
		{
			port:     0,
			enabled:  false,
			endpoint: "0.0.0.0:0",
		},
		{
			port:     0,
			enabled:  true,
			endpoint: "localhost:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			defer setFeatureGateForTest(t, useLocalHostAsDefaultHostfeatureGate, tt.enabled)()
			assert.Equal(t, EndpointForPort(tt.port), tt.endpoint)
		})
	}
}
