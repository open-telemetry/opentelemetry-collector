// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreportconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
)

func TestConfigure(t *testing.T) {
	originalValue := UseOtelForInternalMetricsfeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(UseOtelForInternalMetricsfeatureGate.ID(), false))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(UseOtelForInternalMetricsfeatureGate.ID(), originalValue))
	}()
	tests := []struct {
		name         string
		level        configtelemetry.Level
		wantViewsLen int
	}{
		{
			name:  "none",
			level: configtelemetry.LevelNone,
		},
		{
			name:         "basic",
			level:        configtelemetry.LevelBasic,
			wantViewsLen: 27,
		},
		{
			name:         "normal",
			level:        configtelemetry.LevelNormal,
			wantViewsLen: 27,
		},
		{
			name:         "detailed",
			level:        configtelemetry.LevelDetailed,
			wantViewsLen: 27,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Len(t, AllViews(tt.level), tt.wantViewsLen)
		})
	}
}
