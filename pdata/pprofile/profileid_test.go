// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfileID(t *testing.T) {
	pid := ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	assert.Equal(t, [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}, [16]byte(pid))
	assert.False(t, pid.IsEmpty())
}

func TestNewProfileIDEmpty(t *testing.T) {
	pid := NewProfileIDEmpty()
	assert.Equal(t, [16]byte{}, [16]byte(pid))
	assert.True(t, pid.IsEmpty())
}

func TestProfileIDString(t *testing.T) {
	pid := ProfileID([16]byte{})
	assert.Empty(t, pid.String())

	pid = [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	assert.Equal(t, "12345678123456781234567812345678", pid.String())
}

func TestProfileIDImmutable(t *testing.T) {
	initialBytes := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	pid := ProfileID(initialBytes)
	assert.Equal(t, ProfileID(initialBytes), pid)

	// Get the bytes and try to mutate.
	pid[4] = 0x23

	// Does not change the already created ProfileID.
	assert.NotEqual(t, ProfileID(initialBytes), pid)
}
