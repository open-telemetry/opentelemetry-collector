// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStageString(t *testing.T) {
	assert.Equal(t, "Alpha", StageAlpha.String())
	assert.Equal(t, "Beta", StageBeta.String())
	assert.Equal(t, "Stable", StageStable.String())
	assert.Equal(t, "Deprecated", StageDeprecated.String())
	assert.Equal(t, "Unknown", Stage(-1).String())
}
