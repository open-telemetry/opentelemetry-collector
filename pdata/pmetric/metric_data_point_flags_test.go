// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogRecordFlags(t *testing.T) {
	flags := DataPointFlags(1)
	assert.True(t, flags.NoRecordedValue())
	assert.EqualValues(t, uint32(1), flags)

	flags = flags.WithNoRecordedValue(false)
	assert.False(t, flags.NoRecordedValue())
	assert.EqualValues(t, uint32(0), flags)

	flags = flags.WithNoRecordedValue(true)
	assert.True(t, flags.NoRecordedValue())
	assert.EqualValues(t, uint32(1), flags)
}

func TestDefaultLogRecordFlags(t *testing.T) {
	flags := DefaultDataPointFlags
	assert.False(t, flags.NoRecordedValue())
	assert.EqualValues(t, uint32(0), flags)
}
