// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfileID(t *testing.T) {
	tid := ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	assert.Equal(t, [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}, [16]byte(tid))
	assert.False(t, tid.IsEmpty())
}

func TestNewProfileIDEmpty(t *testing.T) {
	tid := NewProfileIDEmpty()
	assert.Equal(t, [16]byte{}, [16]byte(tid))
	assert.True(t, tid.IsEmpty())
}

func TestProfileIDString(t *testing.T) {
	tid := ProfileID([16]byte{})
	assert.Equal(t, "", tid.String())

	tid = [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	assert.Equal(t, "12345678123456781234567812345678", tid.String())
}

func TestProfileIDImmutable(t *testing.T) {
	initialBytes := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	tid := ProfileID(initialBytes)
	assert.Equal(t, ProfileID(initialBytes), tid)

	// Get the bytes and try to mutate.
	tid[4] = 0x23

	// Does not change the already created ProfileID.
	assert.NotEqual(t, ProfileID(initialBytes), tid)
}
