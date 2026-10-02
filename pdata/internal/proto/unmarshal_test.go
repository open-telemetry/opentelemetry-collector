// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesToString(t *testing.T) {
	t.Run("Safe", func(t *testing.T) {
		data := []byte("value")
		value := BytesToString(data, false)

		data[0] = 'V'

		assert.Equal(t, "value", value)
	})

	t.Run("Unsafe", func(t *testing.T) {
		data := []byte("value")
		value := BytesToString(data, true)

		data[0] = 'V'

		assert.Equal(t, "Value", value)
	})
}

func TestBytesToBytes(t *testing.T) {
	t.Run("Safe", func(t *testing.T) {
		data := []byte("value")
		value := BytesToBytes(data, false)

		data[0] = 'V'

		assert.Equal(t, []byte("value"), value)
	})

	t.Run("Unsafe", func(t *testing.T) {
		data := []byte("value")
		value := BytesToBytes(data, true)

		data[0] = 'V'

		assert.Equal(t, []byte("Value"), value)
	})
}

func TestGrowRepeated(t *testing.T) {
	buf := []byte{
		1 << 3, 0,
		1 << 3, 0,
		2 << 3, 0,
		1 << 3, 0,
	}

	got := GrowRepeated([]int(nil), buf, 2, 1)
	assert.Equal(t, 3, cap(got))
}

func TestGrowRepeatedLargeRemainingDoesNotScan(t *testing.T) {
	buf := make([]byte, countFieldLimit+8)
	got := GrowRepeated([]int(nil), buf, 0, 1)
	assert.Equal(t, 8, cap(got))
}
