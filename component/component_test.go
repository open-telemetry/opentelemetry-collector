// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
