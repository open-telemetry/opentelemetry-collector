// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadInt32(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    int32
		wantErr bool
	}{
		{
			name:    "number",
			jsonStr: `1 `,
			want:    1,
		},
		{
			name:    "string",
			jsonStr: `"1"`,
			want:    1,
		},
		{
			name:    "negative number",
			jsonStr: `-1 `,
			want:    -1,
		},
		{
			name:    "negative string",
			jsonStr: `"-1"`,
			want:    -1,
		},
		{
			name:    "wrong string",
			jsonStr: `"3.f14"`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: `true`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := BorrowIterator([]byte(tt.jsonStr))
			defer ReturnIterator(iter)
			val := iter.ReadInt32()
			if tt.wantErr {
				require.Error(t, iter.Error())
				return
			}
			require.NoError(t, iter.Error())
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestReadUint32(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    uint32
		wantErr bool
	}{
		{
			name:    "number",
			jsonStr: `1 `,
			want:    1,
		},
		{
			name:    "string",
			jsonStr: `"1"`,
			want:    1,
		},
		{
			name:    "negative number",
			jsonStr: `-1 `,
			wantErr: true,
		},
		{
			name:    "negative string",
			jsonStr: `"-1"`,
			wantErr: true,
		},
		{
			name:    "wrong string",
			jsonStr: `"3.f14"`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: `true`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := BorrowIterator([]byte(tt.jsonStr))
			defer ReturnIterator(iter)
			val := iter.ReadUint32()
			if tt.wantErr {
				require.Error(t, iter.Error())
				return
			}
			require.NoError(t, iter.Error())
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestReadInt64(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    int64
		wantErr bool
	}{
		{
			name:    "number",
			jsonStr: `1 `,
			want:    1,
		},
		{
			name:    "string",
			jsonStr: `"1"`,
			want:    1,
		},
		{
			name:    "negative number",
			jsonStr: `-1 `,
			want:    -1,
		},
		{
			name:    "negative string",
			jsonStr: `"-1"`,
			want:    -1,
		},
		{
			name:    "wrong string",
			jsonStr: `"3.f14"`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: `true`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := BorrowIterator([]byte(tt.jsonStr))
			defer ReturnIterator(iter)
			val := iter.ReadInt64()
			if tt.wantErr {
				require.Error(t, iter.Error())
				return
			}
			require.NoError(t, iter.Error())
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestReadUint64(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    uint64
		wantErr bool
	}{
		{
			name:    "number",
			jsonStr: `1 `,
			want:    1,
		},
		{
			name:    "string",
			jsonStr: `"1"`,
			want:    1,
		},
		{
			name:    "negative number",
			jsonStr: `-1 `,
			wantErr: true,
		},
		{
			name:    "negative string",
			jsonStr: `"-1"`,
			wantErr: true,
		},
		{
			name:    "wrong string",
			jsonStr: `"3.f14"`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: `true`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := BorrowIterator([]byte(tt.jsonStr))
			defer ReturnIterator(iter)
			val := iter.ReadUint64()
			if tt.wantErr {
				require.Error(t, iter.Error())
				return
			}
			require.NoError(t, iter.Error())
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestReadFloat32(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    float32
		wantErr bool
	}{
		{
			name:    "number",
			jsonStr: `3.14 `,
			want:    3.14,
		},
		{
			name:    "string",
			jsonStr: `"3.14"`,
			want:    3.14,
		},
		{
			name:    "negative number",
			jsonStr: `-3.14 `,
			want:    -3.14,
		},
		{
			name:    "negative string",
			jsonStr: `"-3.14"`,
			want:    -3.14,
		},
		{
			name:    "wrong string",
			jsonStr: `"3.f14"`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: `true`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := BorrowIterator([]byte(tt.jsonStr))
			defer ReturnIterator(iter)
			val := iter.ReadFloat32()
			if tt.wantErr {
				require.Error(t, iter.Error())
				return
			}
			require.NoError(t, iter.Error())
			assert.InDelta(t, tt.want, val, 0.01)
		})
	}
}

func TestReadFloat32MaxValue(t *testing.T) {
	iter := BorrowIterator([]byte(`{"value": 3.4028234663852886e+38}`))
	defer ReturnIterator(iter)
	for f := iter.ReadObject(); f != ""; f = iter.ReadObject() {
		assert.Equal(t, "value", f)
		assert.InDelta(t, math.MaxFloat32, iter.ReadFloat64(), 0.01)
	}
	require.NoError(t, iter.Error())
}

func TestReadFloat64(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    float64
		wantErr bool
	}{
		{
			name:    "number",
			jsonStr: `3.14 `,
			want:    3.14,
		},
		{
			name:    "string",
			jsonStr: `"3.14"`,
			want:    3.14,
		},
		{
			name:    "negative number",
			jsonStr: `-3.14 `,
			want:    -3.14,
		},
		{
			name:    "negative string",
			jsonStr: `"-3.14"`,
			want:    -3.14,
		},
		{
			name:    "wrong string",
			jsonStr: `"3.f14"`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: `true`,
			wantErr: true,
		},
		{
			name:    "positive infinity",
			jsonStr: `"Infinity"`,
			want:    math.Inf(1),
		},
		{
			name:    "negative infinity",
			jsonStr: `"-Infinity"`,
			want:    math.Inf(-1),
		},
		{
			name:    "not-a-number",
			jsonStr: `"NaN"`,
			want:    math.NaN(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := BorrowIterator([]byte(tt.jsonStr))
			defer ReturnIterator(iter)
			val := iter.ReadFloat64()
			if tt.wantErr {
				require.Error(t, iter.Error())
				return
			}
			require.NoError(t, iter.Error())
			assert.InDelta(t, tt.want, val, 0.01)
		})
	}
}

func TestReadFloat64MaxValue(t *testing.T) {
	iter := BorrowIterator([]byte(`{"value": 1.7976931348623157e+308}`))
	defer ReturnIterator(iter)
	for f := iter.ReadObject(); f != ""; f = iter.ReadObject() {
		assert.Equal(t, "value", f)
		assert.InDelta(t, math.MaxFloat64, iter.ReadFloat64(), 0.01)
	}
	require.NoError(t, iter.Error())
}

func TestReadEnumValue(t *testing.T) {
	valueMap := map[string]int32{
		"undefined": 0,
		"foo":       1,
		"bar":       2,
	}
	tests := []struct {
		name    string
		jsonStr string
		want    int32
		wantErr bool
	}{
		{
			name:    "foo string",
			jsonStr: "\"foo\"\n",
			want:    1,
		},
		{
			name:    "foo number",
			jsonStr: "1\n",
			want:    1,
		},
		{
			name:    "unknown number",
			jsonStr: "5\n",
			want:    5,
		},
		{
			name:    "bar string",
			jsonStr: "\"bar\"\n",
			want:    2,
		},
		{
			name:    "bar number",
			jsonStr: "2\n",
			want:    2,
		},
		{
			name:    "unknown string",
			jsonStr: "\"baz\"\n",
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: "true",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := BorrowIterator([]byte(tt.jsonStr))
			defer ReturnIterator(iter)
			val := iter.ReadEnumValue(valueMap)
			if tt.wantErr {
				assert.Error(t, iter.Error())
				return
			}
			require.NoError(t, iter.Error())
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestReadBytes(t *testing.T) {
	iter := BorrowIterator([]byte(`{"value": "dGVzdA=="}`))
	defer ReturnIterator(iter)
	for f := iter.ReadObject(); f != ""; f = iter.ReadObject() {
		assert.Equal(t, "value", f)
		assert.Equal(t, "test", string(iter.ReadBytes()))
	}
	require.NoError(t, iter.Error())
}
