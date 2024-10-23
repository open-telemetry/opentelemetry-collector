// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKindString(t *testing.T) {
	assert.EqualValues(t, "", Kind(0).String())
	assert.EqualValues(t, "Receiver", KindReceiver.String())
	assert.EqualValues(t, "Processor", KindProcessor.String())
	assert.EqualValues(t, "Exporter", KindExporter.String())
	assert.EqualValues(t, "Extension", KindExtension.String())
	assert.EqualValues(t, "Connector", KindConnector.String())
	assert.EqualValues(t, "", Kind(100).String())
}

func TestStabilityLevelString(t *testing.T) {
	assert.EqualValues(t, "Undefined", StabilityLevelUndefined.String())
	assert.EqualValues(t, "Unmaintained", StabilityLevelUnmaintained.String())
	assert.EqualValues(t, "Deprecated", StabilityLevelDeprecated.String())
	assert.EqualValues(t, "Development", StabilityLevelDevelopment.String())
	assert.EqualValues(t, "Alpha", StabilityLevelAlpha.String())
	assert.EqualValues(t, "Beta", StabilityLevelBeta.String())
	assert.EqualValues(t, "Stable", StabilityLevelStable.String())
	assert.EqualValues(t, "", StabilityLevel(100).String())
}

func TestStabilityLevelLogMessage(t *testing.T) {
	assert.Equal(t, "Stability level of component is undefined", StabilityLevelUndefined.LogMessage())
	assert.Equal(t, "Unmaintained component. Actively looking for contributors. Component will become deprecated after 6 months of remaining unmaintained.", StabilityLevelUnmaintained.LogMessage())
	assert.Equal(t, "Deprecated component. Will be removed in future releases.", StabilityLevelDeprecated.LogMessage())
	assert.Equal(t, "Development component. May change in the future.", StabilityLevelDevelopment.LogMessage())
	assert.Equal(t, "Alpha component. May change in the future.", StabilityLevelAlpha.LogMessage())
	assert.Equal(t, "Beta component. May change in the future.", StabilityLevelBeta.LogMessage())
	assert.Equal(t, "Stable component.", StabilityLevelStable.LogMessage())
	assert.Equal(t, "Stability level of component is undefined", StabilityLevel(100).LogMessage())
}
