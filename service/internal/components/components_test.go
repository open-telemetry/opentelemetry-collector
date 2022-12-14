// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
