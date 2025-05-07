// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogRecordFlags(t *testing.T) {
	flags := LogRecordFlags(1)
	assert.True(t, flags.IsSampled())
	assert.EqualValues(t, uint32(1), flags)

	flags = flags.WithIsSampled(false)
	assert.False(t, flags.IsSampled())
	assert.EqualValues(t, uint32(0), flags)

	flags = flags.WithIsSampled(true)
	assert.True(t, flags.IsSampled())
	assert.EqualValues(t, uint32(1), flags)
}

func TestDefaultLogRecordFlags(t *testing.T) {
	flags := DefaultLogRecordFlags
	assert.False(t, flags.IsSampled())
	assert.EqualValues(t, uint32(0), flags)
}
