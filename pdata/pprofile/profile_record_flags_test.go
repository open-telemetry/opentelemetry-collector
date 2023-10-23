// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfileRecordFlags(t *testing.T) {
	flags := ProfileRecordFlags(1)
	assert.True(t, flags.IsSampled())
	assert.EqualValues(t, uint32(1), flags)

	flags = flags.WithIsSampled(false)
	assert.False(t, flags.IsSampled())
	assert.EqualValues(t, uint32(0), flags)

	flags = flags.WithIsSampled(true)
	assert.True(t, flags.IsSampled())
	assert.EqualValues(t, uint32(1), flags)
}

func TestDefaultProfileRecordFlags(t *testing.T) {
	flags := DefaultProfileRecordFlags
	assert.False(t, flags.IsSampled())
	assert.EqualValues(t, uint32(0), flags)
}
