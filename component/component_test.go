// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKindString(t *testing.T) {
	assert.Empty(t, Kind{}.String())
	assert.Equal(t, "Receiver", KindReceiver.String())
	assert.Equal(t, "Processor", KindProcessor.String())
	assert.Equal(t, "Exporter", KindExporter.String())
	assert.Equal(t, "Extension", KindExtension.String())
	assert.Equal(t, "Connector", KindConnector.String())
}

func TestStabilityLevelUnmarshal(t *testing.T) {
	tests := []struct {
		input       string
		output      StabilityLevel
		expectedErr string
	}{
		{
			input:  "Undefined",
			output: StabilityLevelUndefined,
		},
		{
			input:  "UnmaintaineD",
			output: StabilityLevelUnmaintained,
		},
		{
			input:  "DepreCated",
			output: StabilityLevelDeprecated,
		},
		{
			input:  "Development",
			output: StabilityLevelDevelopment,
		},
		{
			input:  "alpha",
			output: StabilityLevelAlpha,
		},
		{
			input:  "BETA",
			output: StabilityLevelBeta,
		},
		{
			input:  "sTABLe",
			output: StabilityLevelStable,
		},
		{
			input:       "notfound",
			expectedErr: "unsupported stability level: \"notfound\"",
		},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			var sl StabilityLevel
			err := sl.UnmarshalText([]byte(test.input))
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				assert.Equal(t, test.output, sl)
			}
		})
	}
}

func TestStabilityLevelString(t *testing.T) {
	assert.Equal(t, "Undefined", StabilityLevelUndefined.String())
	assert.Equal(t, "Unmaintained", StabilityLevelUnmaintained.String())
	assert.Equal(t, "Deprecated", StabilityLevelDeprecated.String())
	assert.Equal(t, "Development", StabilityLevelDevelopment.String())
	assert.Equal(t, "Alpha", StabilityLevelAlpha.String())
	assert.Equal(t, "Beta", StabilityLevelBeta.String())
	assert.Equal(t, "Stable", StabilityLevelStable.String())
	assert.Empty(t, StabilityLevel(100).String())
}

func TestStabilityLevelLogMessage(t *testing.T) {
	assert.Equal(t, "Stability level of component is undefined", StabilityLevelUndefined.LogMessage())
	assert.Equal(t, "Unmaintained component. Actively looking for contributors. Component will become deprecated after 3 months of remaining unmaintained.", StabilityLevelUnmaintained.LogMessage())
	assert.Equal(t, "Deprecated component. Will be removed in future releases.", StabilityLevelDeprecated.LogMessage())
	assert.Equal(t, "Development component. May change in the future.", StabilityLevelDevelopment.LogMessage())
	assert.Equal(t, "Alpha component. May change in the future.", StabilityLevelAlpha.LogMessage())
	assert.Equal(t, "Beta component. May change in the future.", StabilityLevelBeta.LogMessage())
	assert.Equal(t, "Stable component.", StabilityLevelStable.LogMessage())
	assert.Equal(t, "Stability level of component is undefined", StabilityLevel(100).LogMessage())
}
