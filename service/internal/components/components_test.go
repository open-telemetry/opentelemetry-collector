// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components // import "go.opentelemetry.io/collector/service/internal/components"

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
)

func TestLogStabilityLevel(t *testing.T) {
	tests := []struct {
		level        zapcore.Level
		expectedLogs int
	}{
		{
			level:        zapcore.DebugLevel,
			expectedLogs: 7,
		},
		{
			level:        zapcore.InfoLevel,
			expectedLogs: 4,
		},
	}

	for _, tt := range tests {
		observed, logs := observer.New(tt.level)
		logger := zap.New(observed)
		// ensure log levels are set correctly for each stability level
		LogStabilityLevel(logger, component.StabilityLevelUndefined)
		LogStabilityLevel(logger, component.StabilityLevelUnmaintained)
		LogStabilityLevel(logger, component.StabilityLevelDeprecated)
		LogStabilityLevel(logger, component.StabilityLevelDevelopment)
		LogStabilityLevel(logger, component.StabilityLevelAlpha)
		LogStabilityLevel(logger, component.StabilityLevelBeta)
		LogStabilityLevel(logger, component.StabilityLevelStable)
		require.Equal(t, tt.expectedLogs, logs.Len())
	}
}
