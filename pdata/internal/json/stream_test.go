// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNestedObject(t *testing.T) {
	s := BorrowStream(nil)
	defer ReturnStream(s)

	s.WriteObjectStart()
	s.WriteObjectField("field1")
	s.WriteString("val1")
	s.WriteObjectField("field2")
	s.WriteObjectStart()
	s.WriteObjectField("field3")
	s.WriteObjectStart()
	s.WriteObjectField("field4")
	s.WriteString("val4")
	s.WriteObjectField("field5")
	s.WriteString("val5")
	s.WriteObjectEnd()
	s.WriteObjectField("field6")
	s.WriteString("val6")
	s.WriteObjectField("field7")
	s.WriteObjectStart()
	s.WriteObjectEnd()
	s.WriteObjectEnd()
	s.WriteObjectEnd()

	expected := `{
		"field1": "val1",
		"field2": {
			"field3": {
				"field4": "val4",
				"field5": "val5"
			},
			"field6": "val6",
			"field7": {}
		}
	}`
	assert.JSONEq(t, expected, string(s.Buffer()))
}

func TestMarshalFloat(t *testing.T) {
	tests := []struct {
		name       string
		inputFloat float64
		expected   string
	}{
		{
			name:       "positive infinity",
			inputFloat: math.Inf(1),
			expected:   `"Infinity"`,
		},
		{
			name:       "negative infinity",
			inputFloat: math.Inf(-1),
			expected:   `"-Infinity"`,
		},
		{
			name:       "not-a-number",
			inputFloat: math.NaN(),
			expected:   `"NaN"`,
		},
		{
			name:       "regular float",
			inputFloat: math.MaxFloat64,
			expected:   "1.7976931348623157e+308",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := BorrowStream(nil)
			defer ReturnStream(s)
			s.WriteFloat64(tt.inputFloat)
			require.Equal(t, tt.expected, string(s.Buffer()))
		})
	}
}

func TestWriteBytes(t *testing.T) {
	s := BorrowStream(nil)
	defer ReturnStream(s)
	s.WriteBytes([]byte("test"))
	require.Equal(t, `"dGVzdA=="`, string(s.Buffer()))
}

func BenchmarkWriteInt64(b *testing.B) {
	s := BorrowStream(nil)
	defer ReturnStream(s)
	for b.Loop() {
		s.Reset(nil)
		s.WriteInt64(1_000_000_000_000_000_000)
	}
}

func BenchmarkWriteUint64(b *testing.B) {
	s := BorrowStream(nil)
	defer ReturnStream(s)
	for b.Loop() {
		s.Reset(nil)
		s.WriteUint64(1_000_000_000_000_000_000)
	}
}

func TestWriteInt64(t *testing.T) {
	tests := []struct {
		val      int64
		expected string
	}{
		{0, `"0"`},
		{1, `"1"`},
		{-1, `"-1"`},
		{math.MaxInt64, `"9223372036854775807"`},
		{math.MinInt64, `"-9223372036854775808"`},
		// typical nanosecond timestamp
		{1_000_000_000_000_000_000, `"1000000000000000000"`},
	}
	for _, tt := range tests {
		s := BorrowStream(nil)
		s.WriteInt64(tt.val)
		require.Equal(t, tt.expected, string(s.Buffer()))
		ReturnStream(s)
	}
}

func TestWriteUint64(t *testing.T) {
	tests := []struct {
		val      uint64
		expected string
	}{
		{0, `"0"`},
		{1, `"1"`},
		{math.MaxUint64, `"18446744073709551615"`},
		// typical nanosecond timestamp
		{1_000_000_000_000_000_000, `"1000000000000000000"`},
	}
	for _, tt := range tests {
		s := BorrowStream(nil)
		s.WriteUint64(tt.val)
		require.Equal(t, tt.expected, string(s.Buffer()))
		ReturnStream(s)
	}
}
