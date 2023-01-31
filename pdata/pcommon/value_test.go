// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pcommon

import (
	"encoding/base64"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func TestValue(t *testing.T) {
	v := NewValueStr("abc")
	assert.EqualValues(t, ValueTypeStr, v.Type())
	assert.EqualValues(t, "abc", v.Str())

	v = NewValueInt(123)
	assert.EqualValues(t, ValueTypeInt, v.Type())
	assert.EqualValues(t, 123, v.Int())

	v = NewValueDouble(3.4)
	assert.EqualValues(t, ValueTypeDouble, v.Type())
	assert.EqualValues(t, 3.4, v.Double())

	v = NewValueBool(true)
	assert.EqualValues(t, ValueTypeBool, v.Type())
	assert.True(t, v.Bool())

	v = NewValueBytes()
	assert.EqualValues(t, ValueTypeBytes, v.Type())

	v = NewValueEmpty()
	assert.EqualValues(t, ValueTypeEmpty, v.Type())

	v = NewValueMap()
	assert.EqualValues(t, ValueTypeMap, v.Type())

	v = NewValueSlice()
	assert.EqualValues(t, ValueTypeSlice, v.Type())
}

func TestValueType(t *testing.T) {
	assert.EqualValues(t, "Empty", ValueTypeEmpty.String())
	assert.EqualValues(t, "Str", ValueTypeStr.String())
	assert.EqualValues(t, "Bool", ValueTypeBool.String())
	assert.EqualValues(t, "Int", ValueTypeInt.String())
	assert.EqualValues(t, "Double", ValueTypeDouble.String())
	assert.EqualValues(t, "Map", ValueTypeMap.String())
	assert.EqualValues(t, "Slice", ValueTypeSlice.String())
	assert.EqualValues(t, "Bytes", ValueTypeBytes.String())
	assert.EqualValues(t, "", ValueType(100).String())
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
	assert.EqualValues(t, 2, m1.Map().Len())
	got, exists = m1.Map().Get("double_key")
	assert.True(t, exists)
	assert.Equal(t, NewValueDouble(123), got)
	got, exists = m1.Map().Get("child_map")
	assert.True(t, exists)
	assert.Equal(t, m2, got.Map())

	// Modify the source map m2 that was inserted into m1.
	m2.PutStr("key_in_child", "somestr2")
	assert.EqualValues(t, 1, m2.Len())
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
	assert.EqualValues(t, 1, childMap.Map().Len())
	got, exists = childMap.Map().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewValueStr("somestr3"), got)

	// The source child map should be modified.
	got, exists = m2.Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewValueStr("somestr3"), got)

	removed := m1.Map().Remove("double_key")
	assert.True(t, removed)
	assert.EqualValues(t, 1, m1.Map().Len())
	_, exists = m1.Map().Get("double_key")
	assert.False(t, exists)

	removed = m1.Map().Remove("child_map")
	assert.True(t, removed)
	assert.EqualValues(t, 0, m1.Map().Len())
	_, exists = m1.Map().Get("child_map")
	assert.False(t, exists)

	// Test nil KvlistValue case for Map() func.
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: nil}}
	m1 = newValue(orig)
	assert.EqualValues(t, Map{}, m1.Map())
}

func TestValueSlice(t *testing.T) {
	a1 := NewValueSlice()
	assert.EqualValues(t, ValueTypeSlice, a1.Type())
	assert.EqualValues(t, NewSlice(), a1.Slice())
	assert.EqualValues(t, 0, a1.Slice().Len())

	a1.Slice().AppendEmpty().SetDouble(123)
	assert.EqualValues(t, 1, a1.Slice().Len())
	assert.EqualValues(t, NewValueDouble(123), a1.Slice().At(0))
	// Create a second array.
	a2 := NewValueSlice()
	assert.EqualValues(t, 0, a2.Slice().Len())

	a2.Slice().AppendEmpty().SetStr("somestr")
	assert.EqualValues(t, 1, a2.Slice().Len())
	assert.EqualValues(t, NewValueStr("somestr"), a2.Slice().At(0))

	// Insert the second array as a child.
	a2.CopyTo(a1.Slice().AppendEmpty())
	assert.EqualValues(t, 2, a1.Slice().Len())
	assert.EqualValues(t, NewValueDouble(123), a1.Slice().At(0))
	assert.EqualValues(t, a2, a1.Slice().At(1))

	// Check that the array was correctly inserted.
	childArray := a1.Slice().At(1)
	assert.EqualValues(t, ValueTypeSlice, childArray.Type())
	assert.EqualValues(t, 1, childArray.Slice().Len())

	v := childArray.Slice().At(0)
	assert.EqualValues(t, ValueTypeStr, v.Type())
	assert.EqualValues(t, "somestr", v.Str())

	// Test nil values case for Slice() func.
	a1 = newValue(&otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: nil}})
	assert.EqualValues(t, newSlice(nil), a1.Slice())
}

func TestNilOrigSetValue(t *testing.T) {
	av := NewValueEmpty()
	av.SetStr("abc")
	assert.EqualValues(t, "abc", av.Str())

	av = NewValueEmpty()
	av.SetInt(123)
	assert.EqualValues(t, 123, av.Int())

	av = NewValueEmpty()
	av.SetBool(true)
	assert.True(t, av.Bool())

	av = NewValueEmpty()
	av.SetDouble(1.23)
	assert.EqualValues(t, 1.23, av.Double())

	av = NewValueEmpty()
	av.SetEmptyBytes().FromRaw([]byte{1, 2, 3})
	assert.Equal(t, []byte{1, 2, 3}, av.Bytes().AsRaw())

	av = NewValueEmpty()
	assert.NoError(t, av.SetEmptyMap().FromRaw(map[string]any{"k": "v"}))
	assert.Equal(t, map[string]any{"k": "v"}, av.Map().AsRaw())

	av = NewValueEmpty()
	assert.NoError(t, av.SetEmptySlice().FromRaw([]any{int64(1), "val"}))
	assert.Equal(t, []any{int64(1), "val"}, av.Slice().AsRaw())
}

func TestValue_CopyTo(t *testing.T) {
	// Test nil KvlistValue case for Map() func.
	dest := NewValueEmpty()
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: nil}}
	newValue(orig).CopyTo(dest)
	assert.Nil(t, dest.getOrig().Value.(*otlpcommon.AnyValue_KvlistValue).KvlistValue)

	// Test nil ArrayValue case for Slice() func.
	dest = NewValueEmpty()
	orig = &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: nil}}
	newValue(orig).CopyTo(dest)
	assert.Nil(t, dest.getOrig().Value.(*otlpcommon.AnyValue_ArrayValue).ArrayValue)

	// Test copy empty value.
	orig = &otlpcommon.AnyValue{}
	newValue(orig).CopyTo(dest)
	assert.Nil(t, dest.getOrig().Value)

	av := NewValueEmpty()
	destVal := otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{}}
	av.CopyTo(newValue(&destVal))
	assert.EqualValues(t, nil, destVal.Value)
}

func TestSliceWithNilValues(t *testing.T) {
	origWithNil := []otlpcommon.AnyValue{
		{},
		{Value: &otlpcommon.AnyValue_StringValue{StringValue: "test_value"}},
	}
	sm := newSlice(&origWithNil)

	val := sm.At(0)
	assert.EqualValues(t, ValueTypeEmpty, val.Type())
	assert.EqualValues(t, "", val.Str())

	val = sm.At(1)
	assert.EqualValues(t, ValueTypeStr, val.Type())
	assert.EqualValues(t, "test_value", val.Str())

	sm.AppendEmpty().SetStr("other_value")
	val = sm.At(2)
	assert.EqualValues(t, ValueTypeStr, val.Type())
	assert.EqualValues(t, "other_value", val.Str())
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.input.AsString()
			assert.Equal(t, test.expected, actual)
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.input.AsRaw()
			assert.Equal(t, test.expected, actual)
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
			assert.NoError(t, actual.FromRaw(tt.input))
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestNewValueFromRawInvalid(t *testing.T) {
	actual := NewValueEmpty()
	assert.EqualError(t, actual.FromRaw(ValueTypeDouble), "<Invalid value type pcommon.ValueType>")
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
