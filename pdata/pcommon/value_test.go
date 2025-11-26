// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"encoding/base64"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/internal"
)

func TestValue(t *testing.T) {
	v := NewValueStr("abc")
	assert.Equal(t, ValueTypeStr, v.Type())
	assert.Equal(t, "abc", v.Str())

	v = NewValueInt(123)
	assert.Equal(t, ValueTypeInt, v.Type())
	assert.EqualValues(t, 123, v.Int())

	v = NewValueDouble(3.4)
	assert.Equal(t, ValueTypeDouble, v.Type())
	assert.InDelta(t, 3.4, v.Double(), 0.01)

	v = NewValueBool(true)
	assert.Equal(t, ValueTypeBool, v.Type())
	assert.True(t, v.Bool())

	v = NewValueBytes()
	assert.Equal(t, ValueTypeBytes, v.Type())

	v = NewValueEmpty()
	assert.Equal(t, ValueTypeEmpty, v.Type())

	v = NewValueMap()
	assert.Equal(t, ValueTypeMap, v.Type())

	v = NewValueSlice()
	assert.Equal(t, ValueTypeSlice, v.Type())
}

func TestValueReadOnly(t *testing.T) {
	state := internal.NewState()
	state.MarkReadOnly()
	v := newValue(&internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "v"}}, state)

	assert.Equal(t, ValueTypeStr, v.Type())
	assert.Equal(t, "v", v.Str())
	assert.EqualValues(t, 0, v.Int())
	assert.InDelta(t, 0, v.Double(), 0.01)
	assert.False(t, v.Bool())
	assert.Equal(t, ByteSlice{}, v.Bytes())
	assert.Equal(t, Map{}, v.Map())
	assert.Equal(t, Slice{}, v.Slice())

	assert.Equal(t, "v", v.AsString())

	assert.Panics(t, func() { v.SetStr("abc") })
	assert.Panics(t, func() { v.SetInt(123) })
	assert.Panics(t, func() { v.SetDouble(3.4) })
	assert.Panics(t, func() { v.SetBool(true) })
	assert.Panics(t, func() { v.SetEmptyBytes() })
	assert.Panics(t, func() { v.SetEmptyMap() })
	assert.Panics(t, func() { v.SetEmptySlice() })

	v2 := NewValueEmpty()
	v.CopyTo(v2)
	assert.Equal(t, v.AsRaw(), v2.AsRaw())
	assert.Panics(t, func() { v2.CopyTo(v) })
}

func TestValueType(t *testing.T) {
	assert.Equal(t, "Empty", ValueTypeEmpty.String())
	assert.Equal(t, "Str", ValueTypeStr.String())
	assert.Equal(t, "Bool", ValueTypeBool.String())
	assert.Equal(t, "Int", ValueTypeInt.String())
	assert.Equal(t, "Double", ValueTypeDouble.String())
	assert.Equal(t, "Map", ValueTypeMap.String())
	assert.Equal(t, "Slice", ValueTypeSlice.String())
	assert.Equal(t, "Bytes", ValueTypeBytes.String())
	assert.Empty(t, ValueType(100).String())
}

func TestValueMap(t *testing.T) {
	m1 := NewValueMap()
	assert.Equal(t, ValueTypeMap, m1.Type())
	assert.Equal(t, NewMap(), m1.Map())
	assert.Equal(t, 0, m1.Map().Len())

	m1.Map().PutDouble("double_key", 123)
	assert.Equal(t, 1, m1.Map().Len())
	got, exists := m1.Map().Get("double_key")
	assert.True(t, exists)
	assert.Equal(t, NewValueDouble(123), got)

	// Create a second map.
	m2 := m1.Map().PutEmptyMap("child_map")
	assert.Equal(t, 0, m2.Len())

	// Modify the source map that was inserted.
	m2.PutStr("key_in_child", "somestr")
	assert.Equal(t, 1, m2.Len())
	got, exists = m2.Get("key_in_child")
	assert.True(t, exists)
	assert.Equal(t, NewValueStr("somestr"), got)

	// Insert the second map as a child. This should perform a deep copy.
	assert.Equal(t, 2, m1.Map().Len())
	got, exists = m1.Map().Get("double_key")
	assert.True(t, exists)
	assert.Equal(t, NewValueDouble(123), got)
	got, exists = m1.Map().Get("child_map")
	assert.True(t, exists)
	assert.Equal(t, m2, got.Map())

	// Modify the source map m2 that was inserted into m1.
	m2.PutStr("key_in_child", "somestr2")
	assert.Equal(t, 1, m2.Len())
	got, exists = m2.Get("key_in_child")
	assert.True(t, exists)
	assert.Equal(t, NewValueStr("somestr2"), got)

	// The child map inside m1 should be modified.
	childMap, childMapExists := m1.Map().Get("child_map")
	require.True(t, childMapExists)
	got, exists = childMap.Map().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewValueStr("somestr2"), got)

	// Now modify the inserted map (not the source)
	childMap.Map().PutStr("key_in_child", "somestr3")
	assert.Equal(t, 1, childMap.Map().Len())
	got, exists = childMap.Map().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewValueStr("somestr3"), got)

	// The source child map should be modified.
	got, exists = m2.Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewValueStr("somestr3"), got)

	removed := m1.Map().Remove("double_key")
	assert.True(t, removed)
	assert.Equal(t, 1, m1.Map().Len())
	_, exists = m1.Map().Get("double_key")
	assert.False(t, exists)

	removed = m1.Map().Remove("child_map")
	assert.True(t, removed)
	assert.Equal(t, 0, m1.Map().Len())
	_, exists = m1.Map().Get("child_map")
	assert.False(t, exists)

	// Test nil KvlistValue case for MapWrapper() func.
	orig := &internal.AnyValue{Value: &internal.AnyValue_KvlistValue{KvlistValue: nil}}
	m1 = newValue(orig, internal.NewState())
	assert.Equal(t, Map{}, m1.Map())
}

func TestValueSlice(t *testing.T) {
	a1 := NewValueSlice()
	assert.Equal(t, ValueTypeSlice, a1.Type())
	assert.Equal(t, NewSlice(), a1.Slice())
	assert.Equal(t, 0, a1.Slice().Len())

	a1.Slice().AppendEmpty().SetDouble(123)
	assert.Equal(t, 1, a1.Slice().Len())
	assert.Equal(t, NewValueDouble(123), a1.Slice().At(0))
	// Create a second array.
	a2 := NewValueSlice()
	assert.Equal(t, 0, a2.Slice().Len())

	a2.Slice().AppendEmpty().SetStr("somestr")
	assert.Equal(t, 1, a2.Slice().Len())
	assert.Equal(t, NewValueStr("somestr"), a2.Slice().At(0))

	// Insert the second array as a child.
	a2.CopyTo(a1.Slice().AppendEmpty())
	assert.Equal(t, 2, a1.Slice().Len())
	assert.Equal(t, NewValueDouble(123), a1.Slice().At(0))
	assert.Equal(t, a2, a1.Slice().At(1))

	// Check that the array was correctly inserted.
	childArray := a1.Slice().At(1)
	assert.Equal(t, ValueTypeSlice, childArray.Type())
	assert.Equal(t, 1, childArray.Slice().Len())

	v := childArray.Slice().At(0)
	assert.Equal(t, ValueTypeStr, v.Type())
	assert.Equal(t, "somestr", v.Str())

	// Test nil values case for Slice() func.
	a1 = newValue(&internal.AnyValue{Value: &internal.AnyValue_ArrayValue{ArrayValue: nil}}, internal.NewState())
	assert.Equal(t, newSlice(nil, nil), a1.Slice())
}

func TestNilOrigSetValue(t *testing.T) {
	av := NewValueEmpty()
	av.SetStr("abc")
	assert.Equal(t, "abc", av.Str())

	av = NewValueEmpty()
	av.SetInt(123)
	assert.EqualValues(t, 123, av.Int())

	av = NewValueEmpty()
	av.SetBool(true)
	assert.True(t, av.Bool())

	av = NewValueEmpty()
	av.SetDouble(1.23)
	assert.InDelta(t, 1.23, av.Double(), 0.01)

	av = NewValueEmpty()
	av.SetEmptyBytes().FromRaw([]byte{1, 2, 3})
	assert.Equal(t, []byte{1, 2, 3}, av.Bytes().AsRaw())

	av = NewValueEmpty()
	require.NoError(t, av.SetEmptyMap().FromRaw(map[string]any{"k": "v"}))
	assert.Equal(t, map[string]any{"k": "v"}, av.Map().AsRaw())

	av = NewValueEmpty()
	require.NoError(t, av.SetEmptySlice().FromRaw([]any{int64(1), "val"}))
	assert.Equal(t, []any{int64(1), "val"}, av.Slice().AsRaw())
}

func TestValue_MoveTo(t *testing.T) {
	src := NewValueMap()
	src.Map().PutStr("key", "value")

	dest := NewValueEmpty()
	assert.True(t, dest.Equal(NewValueEmpty()))

	src.MoveTo(dest)
	assert.True(t, src.Equal(NewValueEmpty()))

	expected := NewValueMap()
	expected.Map().PutStr("key", "value")
	assert.True(t, dest.Equal(expected))

	dest.MoveTo(dest)
	assert.True(t, dest.Equal(expected))
}

func TestValue_CopyTo(t *testing.T) {
	dest := NewValueEmpty()
	orig := internal.GenTestAnyValue()
	newValue(orig, internal.NewState()).CopyTo(dest)
	assert.Equal(t, internal.GenTestAnyValue(), dest.getOrig())
}

func TestSliceWithNilValues(t *testing.T) {
	origWithNil := []internal.AnyValue{
		{},
		{Value: &internal.AnyValue_StringValue{StringValue: "test_value"}},
	}
	sm := newSlice(&origWithNil, internal.NewState())

	val := sm.At(0)
	assert.Equal(t, ValueTypeEmpty, val.Type())
	assert.Empty(t, val.Str())

	val = sm.At(1)
	assert.Equal(t, ValueTypeStr, val.Type())
	assert.Equal(t, "test_value", val.Str())

	sm.AppendEmpty().SetStr("other_value")
	val = sm.At(2)
	assert.Equal(t, ValueTypeStr, val.Type())
	assert.Equal(t, "other_value", val.Str())
}

func TestValueAsString(t *testing.T) {
	tests := []struct {
		name     string
		input    Value
		expected string
	}{
		{
			name:     "string",
			input:    NewValueStr("string value"),
			expected: "string value",
		},
		{
			name:     "int64",
			input:    NewValueInt(42),
			expected: "42",
		},
		{
			name:     "float64",
			input:    NewValueDouble(1.61803399),
			expected: "1.61803399",
		},
		{
			name:     "small float64",
			input:    NewValueDouble(.000000009),
			expected: "9e-9",
		},
		{
			name:     "bad float64",
			input:    NewValueDouble(math.Inf(1)),
			expected: "json: unsupported value: +Inf",
		},
		{
			name:     "boolean",
			input:    NewValueBool(true),
			expected: "true",
		},
		{
			name:     "empty_map",
			input:    NewValueMap(),
			expected: "{}",
		},
		{
			name:     "simple_map",
			input:    generateTestValueMap(),
			expected: "{\"arrKey\":[\"strOne\",\"strTwo\"],\"boolKey\":false,\"floatKey\":18.6,\"intKey\":7,\"mapKey\":{\"keyOne\":\"valOne\",\"keyTwo\":\"valTwo\"},\"nullKey\":null,\"strKey\":\"strVal\"}",
		},
		{
			name:     "empty_array",
			input:    NewValueSlice(),
			expected: "[]",
		},
		{
			name:     "simple_array",
			input:    generateTestValueSlice(),
			expected: "[\"strVal\",7,18.6,false,null]",
		},
		{
			name:     "empty",
			input:    NewValueEmpty(),
			expected: "",
		},
		{
			name:     "bytes",
			input:    generateTestValueBytes(),
			expected: base64.StdEncoding.EncodeToString([]byte("String bytes")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.input.AsString()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestValueAsRaw(t *testing.T) {
	tests := []struct {
		name     string
		input    Value
		expected any
	}{
		{
			name:     "string",
			input:    NewValueStr("value"),
			expected: "value",
		},
		{
			name:     "int",
			input:    NewValueInt(11),
			expected: int64(11),
		},
		{
			name:     "double",
			input:    NewValueDouble(1.2),
			expected: 1.2,
		},
		{
			name:     "bytes",
			input:    generateTestValueBytes(),
			expected: []byte("String bytes"),
		},
		{
			name:     "empty",
			input:    NewValueEmpty(),
			expected: nil,
		},
		{
			name:     "slice",
			input:    generateTestValueSlice(),
			expected: []any{"strVal", int64(7), 18.6, false, nil},
		},
		{
			name:  "map",
			input: generateTestValueMap(),
			expected: map[string]any{
				"mapKey":   map[string]any{"keyOne": "valOne", "keyTwo": "valTwo"},
				"nullKey":  nil,
				"strKey":   "strVal",
				"arrKey":   []any{"strOne", "strTwo"},
				"boolKey":  false,
				"floatKey": 18.6,
				"intKey":   int64(7),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.input.AsRaw()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestNewValueFromRaw(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected Value
	}{
		{
			name:     "nil",
			input:    nil,
			expected: NewValueEmpty(),
		},
		{
			name:     "string",
			input:    "text",
			expected: NewValueStr("text"),
		},
		{
			name:     "int",
			input:    123,
			expected: NewValueInt(int64(123)),
		},
		{
			name:     "int8",
			input:    int8(12),
			expected: NewValueInt(int64(12)),
		},
		{
			name:     "int16",
			input:    int16(23),
			expected: NewValueInt(int64(23)),
		},
		{
			name:     "int32",
			input:    int32(34),
			expected: NewValueInt(int64(34)),
		},
		{
			name:     "int64",
			input:    int64(45),
			expected: NewValueInt(45),
		},
		{
			name:     "uint",
			input:    uint(56),
			expected: NewValueInt(int64(56)),
		},
		{
			name:     "uint8",
			input:    uint8(67),
			expected: NewValueInt(int64(67)),
		},
		{
			name:     "uint16",
			input:    uint16(78),
			expected: NewValueInt(int64(78)),
		},
		{
			name:     "uint32",
			input:    uint32(89),
			expected: NewValueInt(int64(89)),
		},
		{
			name:     "uint64",
			input:    uint64(90),
			expected: NewValueInt(int64(90)),
		},
		{
			name:     "float32",
			input:    float32(1.234),
			expected: NewValueDouble(float64(float32(1.234))),
		},
		{
			name:     "float64",
			input:    float64(2.345),
			expected: NewValueDouble(float64(2.345)),
		},
		{
			name:     "bool",
			input:    true,
			expected: NewValueBool(true),
		},
		{
			name:  "bytes",
			input: []byte{1, 2, 3},
			expected: func() Value {
				m := NewValueBytes()
				m.Bytes().FromRaw([]byte{1, 2, 3})
				return m
			}(),
		},
		{
			name: "map",
			input: map[string]any{
				"k": "v",
			},
			expected: func() Value {
				m := NewValueMap()
				assert.NoError(t, m.Map().FromRaw(map[string]any{"k": "v"}))
				return m
			}(),
		},
		{
			name:  "empty map",
			input: map[string]any{},
			expected: func() Value {
				m := NewValueMap()
				assert.NoError(t, m.Map().FromRaw(map[string]any{}))
				return m
			}(),
		},
		{
			name:  "slice",
			input: []any{"v1", "v2"},
			expected: (func() Value {
				s := NewValueSlice()
				assert.NoError(t, s.Slice().FromRaw([]any{"v1", "v2"}))
				return s
			})(),
		},
		{
			name:  "empty slice",
			input: []any{},
			expected: (func() Value {
				s := NewValueSlice()
				assert.NoError(t, s.Slice().FromRaw([]any{}))
				return s
			})(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := NewValueEmpty()
			require.NoError(t, actual.FromRaw(tt.input))
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestNewValueFromRawInvalid(t *testing.T) {
	actual := NewValueEmpty()
	assert.EqualError(t, actual.FromRaw(ValueTypeDouble), "<Invalid value type pcommon.ValueType>")
}

func TestInvalidValue(t *testing.T) {
	v := Value{}
	assert.False(t, v.Bool())
	assert.Equal(t, int64(0), v.Int())
	assert.InDelta(t, float64(0), v.Double(), 0.01)
	assert.Empty(t, v.Str())
	assert.Equal(t, ByteSlice{}, v.Bytes())
	assert.Equal(t, Map{}, v.Map())
	assert.Equal(t, Slice{}, v.Slice())
	assert.Panics(t, func() { v.AsString() })
	assert.Panics(t, func() { v.AsRaw() })
	assert.Panics(t, func() { _ = v.FromRaw(1) })
	assert.Panics(t, func() { v.Type() })
	assert.Panics(t, func() { v.SetStr("") })
	assert.Panics(t, func() { v.SetInt(0) })
	assert.Panics(t, func() { v.SetDouble(0) })
	assert.Panics(t, func() { v.SetBool(false) })
	assert.Panics(t, func() { v.SetEmptyBytes() })
	assert.Panics(t, func() { v.SetEmptyMap() })
	assert.Panics(t, func() { v.SetEmptySlice() })
	nv := NewValueEmpty()
	v.CopyTo(nv)
	assert.Nil(t, nv.getOrig().Value)
}

func TestValueEqual(t *testing.T) {
	for _, tt := range []struct {
		name       string
		value      Value
		comparison Value
		expected   bool
	}{
		{
			name:       "different types",
			value:      NewValueEmpty(),
			comparison: NewValueStr("test"),
			expected:   false,
		},
		{
			name:       "same empty",
			value:      NewValueEmpty(),
			comparison: NewValueEmpty(),
			expected:   true,
		},
		{
			name:       "same strings",
			value:      NewValueStr("test"),
			comparison: NewValueStr("test"),
			expected:   true,
		},
		{
			name:       "different strings",
			value:      NewValueStr("test"),
			comparison: NewValueStr("non-test"),
			expected:   false,
		},
		{
			name:       "same booleans",
			value:      NewValueBool(true),
			comparison: NewValueBool(true),
			expected:   true,
		},
		{
			name:       "different booleans",
			value:      NewValueBool(true),
			comparison: NewValueBool(false),
			expected:   false,
		},
		{
			name:       "same int",
			value:      NewValueInt(42),
			comparison: NewValueInt(42),
			expected:   true,
		},
		{
			name:       "different ints",
			value:      NewValueInt(42),
			comparison: NewValueInt(1701),
			expected:   false,
		},
		{
			name:       "same double",
			value:      NewValueDouble(13.37),
			comparison: NewValueDouble(13.37),
			expected:   true,
		},
		{
			name:       "different doubles",
			value:      NewValueDouble(13.37),
			comparison: NewValueDouble(17.01),
			expected:   false,
		},
		{
			name: "same byte slice",
			value: func() Value {
				m := NewValueBytes()
				m.Bytes().FromRaw([]byte{1, 3, 3, 7})
				return m
			}(),
			comparison: func() Value {
				m := NewValueBytes()
				m.Bytes().FromRaw([]byte{1, 3, 3, 7})
				return m
			}(),
			expected: true,
		},
		{
			name: "different byte slice",
			value: func() Value {
				m := NewValueBytes()
				m.Bytes().FromRaw([]byte{1, 3, 3, 7})
				return m
			}(),
			comparison: func() Value {
				m := NewValueBytes()
				m.Bytes().FromRaw([]byte{1, 7, 0, 1})
				return m
			}(),
			expected: false,
		},
		{
			name: "same slice",
			value: func() Value {
				m := NewValueSlice()
				require.NoError(t, m.Slice().FromRaw([]any{1337}))
				return m
			}(),
			comparison: func() Value {
				m := NewValueSlice()
				require.NoError(t, m.Slice().FromRaw([]any{1337}))
				return m
			}(),
			expected: true,
		},
		{
			name: "different slice",
			value: func() Value {
				m := NewValueSlice()
				require.NoError(t, m.Slice().FromRaw([]any{1337}))
				return m
			}(),
			comparison: func() Value {
				m := NewValueSlice()
				require.NoError(t, m.Slice().FromRaw([]any{1701}))
				return m
			}(),
			expected: false,
		},
		{
			name: "same map",
			value: func() Value {
				m := NewValueMap()
				m.Map().PutStr("hello", "world")
				return m
			}(),
			comparison: func() Value {
				m := NewValueMap()
				m.Map().PutStr("hello", "world")
				return m
			}(),
			expected: true,
		},
		{
			name: "different maps",
			value: func() Value {
				m := NewValueMap()
				m.Map().PutStr("hello", "world")
				return m
			}(),
			comparison: func() Value {
				m := NewValueMap()
				m.Map().PutStr("bonjour", "monde")
				return m
			}(),
			expected: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.value.Equal(tt.comparison))
		})
	}
}

func BenchmarkValueEqual(b *testing.B) {
	testutil.SkipMemoryBench(b)

	for _, bb := range []struct {
		name       string
		value      Value
		comparison Value
	}{
		{
			name:       "nil",
			value:      NewValueEmpty(),
			comparison: NewValueEmpty(),
		},
		{
			name:       "strings",
			value:      NewValueStr("test"),
			comparison: NewValueStr("test"),
		},
		{
			name:       "booleans",
			value:      NewValueBool(true),
			comparison: NewValueBool(true),
		},
		{
			name:       "ints",
			value:      NewValueInt(42),
			comparison: NewValueInt(42),
		},
		{
			name:       "doubles",
			value:      NewValueDouble(13.37),
			comparison: NewValueDouble(13.37),
		},
		{
			name: "byte slices",
			value: func() Value {
				m := NewValueBytes()
				m.Bytes().FromRaw([]byte{1, 3, 3, 7})
				return m
			}(),
			comparison: func() Value {
				m := NewValueBytes()
				m.Bytes().FromRaw([]byte{1, 3, 3, 7})
				return m
			}(),
		},
		{
			name: "slices",
			value: func() Value {
				m := NewValueSlice()
				require.NoError(b, m.Slice().FromRaw([]any{1337}))
				return m
			}(),
			comparison: func() Value {
				m := NewValueSlice()
				require.NoError(b, m.Slice().FromRaw([]any{1337}))
				return m
			}(),
		},
		{
			name: "maps",
			value: func() Value {
				m := NewValueMap()
				m.Map().PutStr("hello", "world")
				return m
			}(),
			comparison: func() Value {
				m := NewValueMap()
				m.Map().PutStr("hello", "world")
				return m
			}(),
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = bb.value.Equal(bb.comparison)
			}
		})
	}
}

func generateTestValueMap() Value {
	ret := NewValueMap()
	attrMap := ret.Map()
	attrMap.PutStr("strKey", "strVal")
	attrMap.PutInt("intKey", 7)
	attrMap.PutDouble("floatKey", 18.6)
	attrMap.PutBool("boolKey", false)
	attrMap.PutEmpty("nullKey")

	m := attrMap.PutEmptyMap("mapKey")
	m.PutStr("keyOne", "valOne")
	m.PutStr("keyTwo", "valTwo")

	s := attrMap.PutEmptySlice("arrKey")
	s.AppendEmpty().SetStr("strOne")
	s.AppendEmpty().SetStr("strTwo")

	return ret
}

func generateTestValueSlice() Value {
	ret := NewValueSlice()
	attrArr := ret.Slice()
	attrArr.AppendEmpty().SetStr("strVal")
	attrArr.AppendEmpty().SetInt(7)
	attrArr.AppendEmpty().SetDouble(18.6)
	attrArr.AppendEmpty().SetBool(false)
	attrArr.AppendEmpty()
	return ret
}

func generateTestValueBytes() Value {
	v := NewValueBytes()
	v.Bytes().FromRaw([]byte("String bytes"))
	return v
}
