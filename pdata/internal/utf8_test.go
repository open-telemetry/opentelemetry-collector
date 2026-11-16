// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUTF8(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.True(t, ValidateUTF8(nil))
	})

	t.Run("nil pointer and nil interface field", func(t *testing.T) {
		type sample struct {
			P *string
			V any
		}
		assert.True(t, ValidateUTF8((*sample)(nil)))
		assert.True(t, ValidateUTF8(sample{}))
	})

	t.Run("valid composite", func(t *testing.T) {
		type sample struct {
			S string
			A []string
		}
		assert.True(t, ValidateUTF8(&sample{S: "ok", A: []string{"a", "b"}}))
	})

	t.Run("invalid string", func(t *testing.T) {
		assert.False(t, ValidateUTF8(string([]byte{0xff})))
	})

	t.Run("invalid nested composite", func(t *testing.T) {
		type sample struct {
			A [1]string
		}
		assert.False(t, ValidateUTF8(sample{A: [1]string{string([]byte{0xff})}}))
	})

	t.Run("invalid behind interface", func(t *testing.T) {
		type sample struct {
			V any
		}
		assert.False(t, ValidateUTF8(sample{V: &AnyValue{Value: &AnyValue_StringValue{StringValue: string([]byte{0xff})}}}))
	})

	t.Run("scalar collections are ignored", func(t *testing.T) {
		type sample struct {
			ID    [16]byte
			Bytes []byte
			Ints  []int64
		}
		assert.True(t, ValidateUTF8(sample{ID: [16]byte{0xff}, Bytes: []byte{0xff}, Ints: []int64{1}}))
	})

	t.Run("pdata requests", func(t *testing.T) {
		assert.True(t, ValidateUTF8(GenTestTracesWrapper().orig))
		assert.True(t, ValidateUTF8(GenTestLogsWrapper().orig))
		assert.True(t, ValidateUTF8(GenTestMetricsWrapper().orig))
		assert.True(t, ValidateUTF8(GenTestProfilesWrapper().orig))
	})
}
