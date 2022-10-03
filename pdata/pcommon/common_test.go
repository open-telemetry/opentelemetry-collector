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

	"go.opentelemetry.io/collector/pdata/internal"
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
	assert.EqualValues(t, "EMPTY", ValueTypeEmpty.String())
	assert.EqualValues(t, "STRING", ValueTypeStr.String())
	assert.EqualValues(t, "BOOL", ValueTypeBool.String())
	assert.EqualValues(t, "INT", ValueTypeInt.String())
	assert.EqualValues(t, "DOUBLE", ValueTypeDouble.String())
	assert.EqualValues(t, "MAP", ValueTypeMap.String())
	assert.EqualValues(t, "SLICE", ValueTypeSlice.String())
	assert.EqualValues(t, "BYTES", ValueTypeBytes.String())
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
	av.SetEmptyMap().FromRaw(map[string]interface{}{"k": "v"})
	assert.Equal(t, map[string]interface{}{"k": "v"}, av.Map().AsRaw())

	av = NewValueEmpty()
	av.SetEmptySlice().FromRaw([]interface{}{int64(1), "val"})
	assert.Equal(t, []interface{}{int64(1), "val"}, av.Slice().AsRaw())
}

func TestValueEqual(t *testing.T) {
	av1 := NewValueEmpty()
	assert.True(t, av1.Equal(av1)) // nolint:gocritic
	av2 := NewValueEmpty()
	assert.True(t, av1.Equal(av2))

	av2 = NewValueStr("abc")
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueStr("abc")
	assert.True(t, av1.Equal(av2))

	av2 = NewValueStr("edf")
	assert.False(t, av1.Equal(av2))

	av2 = NewValueInt(123)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueInt(234)
	assert.False(t, av1.Equal(av2))

	av1 = NewValueInt(123)
	assert.True(t, av1.Equal(av2))

	av2 = NewValueDouble(123)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueDouble(234)
	assert.False(t, av1.Equal(av2))

	av1 = NewValueDouble(123)
	assert.True(t, av1.Equal(av2))

	av2 = NewValueBool(false)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueBool(true)
	assert.False(t, av1.Equal(av2))

	av1 = NewValueBool(false)
	assert.True(t, av1.Equal(av2))

	av2 = NewValueBytes()
	av2.Bytes().FromRaw([]byte{1, 2, 3})
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueBytes()
	av1.Bytes().FromRaw([]byte{1, 2, 4})
	assert.False(t, av1.Equal(av2))

	av1.Bytes().SetAt(2, 3)
	assert.True(t, av1.Equal(av2))

	av1 = NewValueSlice()
	av1.Slice().AppendEmpty().SetInt(123)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av2 = NewValueSlice()
	av2.Slice().AppendEmpty().SetDouble(123)
	assert.False(t, av1.Equal(av2))

	NewValueInt(123).CopyTo(av2.Slice().At(0))
	assert.True(t, av1.Equal(av2))

	av1.CopyTo(av2.Slice().AppendEmpty())
	assert.False(t, av1.Equal(av2))

	av1 = NewValueMap()
	av1.Map().PutStr("foo", "bar")
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av2 = NewValueMap()
	av2.Map().PutStr("foo", "bar")
	assert.True(t, av1.Equal(av2))

	fooVal, ok := av2.Map().Get("foo")
	assert.True(t, ok)
	fooVal.SetStr("not-bar")
	assert.False(t, av1.Equal(av2))
}

func TestMap(t *testing.T) {
	assert.EqualValues(t, 0, NewMap().Len())

	val, exist := NewMap().Get("test_key")
	assert.False(t, exist)
	assert.EqualValues(t, newValue(nil), val)

	putString := NewMap()
	putString.PutStr("k", "v")
	assert.EqualValues(t, Map(internal.GenerateTestMap()), putString)

	putInt := NewMap()
	putInt.PutInt("k", 123)
	assert.EqualValues(t, generateTestIntMap(), putInt)

	putDouble := NewMap()
	putDouble.PutDouble("k", 12.3)
	assert.EqualValues(t, generateTestDoubleMap(), putDouble)

	putBool := NewMap()
	putBool.PutBool("k", true)
	assert.EqualValues(t, generateTestBoolMap(), putBool)

	putBytes := NewMap()
	putBytes.PutEmptyBytes("k").FromRaw([]byte{1, 2, 3, 4, 5})
	assert.EqualValues(t, generateTestBytesMap(), putBytes)

	putMap := NewMap()
	putMap.PutEmptyMap("k")
	assert.EqualValues(t, generateTestEmptyMap(), putMap)

	putSlice := NewMap()
	putSlice.PutEmptySlice("k")
	assert.EqualValues(t, generateTestEmptySlice(), putSlice)

	removeMap := NewMap()
	assert.False(t, removeMap.Remove("k"))
	assert.EqualValues(t, NewMap(), removeMap)

	// Test Sort
	assert.EqualValues(t, NewMap(), NewMap().Sort())
}

func TestMapPutEmpty(t *testing.T) {
	m := NewMap()
	v := m.PutEmpty("k1")
	assert.EqualValues(t, map[string]interface{}{
		"k1": nil,
	}, m.AsRaw())

	v.SetBool(true)
	assert.EqualValues(t, map[string]interface{}{
		"k1": true,
	}, m.AsRaw())

	v = m.PutEmpty("k1")
	v.SetInt(1)
	v2, ok := m.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, int64(1), v2.Int())
}

func TestMapPutEmptyMap(t *testing.T) {
	m := NewMap()
	childMap := m.PutEmptyMap("k1")
	assert.EqualValues(t, map[string]interface{}{
		"k1": map[string]interface{}{},
	}, m.AsRaw())
	childMap.PutEmptySlice("k2").AppendEmpty().SetStr("val")
	assert.EqualValues(t, map[string]interface{}{
		"k1": map[string]interface{}{
			"k2": []interface{}{"val"},
		},
	}, m.AsRaw())

	childMap.PutEmptyMap("k2").PutInt("k3", 1)
	assert.EqualValues(t, map[string]interface{}{
		"k1": map[string]interface{}{
			"k2": map[string]interface{}{"k3": int64(1)},
		},
	}, m.AsRaw())
}

func TestMapPutEmptySlice(t *testing.T) {
	m := NewMap()
	childSlice := m.PutEmptySlice("k")
	assert.EqualValues(t, map[string]interface{}{
		"k": []interface{}{},
	}, m.AsRaw())
	childSlice.AppendEmpty().SetDouble(1.1)
	assert.EqualValues(t, map[string]interface{}{
		"k": []interface{}{1.1},
	}, m.AsRaw())

	m.PutEmptySlice("k")
	assert.EqualValues(t, map[string]interface{}{
		"k": []interface{}{},
	}, m.AsRaw())
	childSliceVal, ok := m.Get("k")
	assert.True(t, ok)
	childSliceVal.Slice().AppendEmpty().SetEmptySlice().AppendEmpty().SetStr("val")
	assert.EqualValues(t, map[string]interface{}{
		"k": []interface{}{[]interface{}{"val"}},
	}, m.AsRaw())
}

func TestMapPutEmptyBytes(t *testing.T) {
	m := NewMap()
	b := m.PutEmptyBytes("k")
	bv, ok := m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte(nil), bv.Bytes().AsRaw())
	b.FromRaw([]byte{1, 2, 3})
	bv, ok = m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte{1, 2, 3}, bv.Bytes().AsRaw())

	m.PutEmptyBytes("k")
	bv, ok = m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte(nil), bv.Bytes().AsRaw())
	bv.Bytes().FromRaw([]byte{3, 2, 1})
	bv, ok = m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte{3, 2, 1}, bv.Bytes().AsRaw())
}

func TestMapWithEmpty(t *testing.T) {
	origWithNil := []otlpcommon.KeyValue{
		{},
		{
			Key:   "test_key",
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "test_value"}},
		},
		{
			Key:   "test_key2",
			Value: otlpcommon.AnyValue{Value: nil},
		},
	}
	sm := newMap(&origWithNil)
	val, exist := sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeStr, val.Type())
	assert.EqualValues(t, "test_value", val.Str())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeEmpty, val.Type())
	assert.EqualValues(t, "", val.Str())

	sm.PutStr("other_key_string", "other_value")
	val, exist = sm.Get("other_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeStr, val.Type())
	assert.EqualValues(t, "other_value", val.Str())

	sm.PutInt("other_key_int", 123)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 123, val.Int())

	sm.PutDouble("other_key_double", 1.23)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 1.23, val.Double())

	sm.PutBool("other_key_bool", true)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.True(t, val.Bool())

	sm.PutEmptyBytes("other_key_bytes").FromRaw([]byte{7, 8, 9})
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{7, 8, 9}, val.Bytes().AsRaw())

	sm.PutStr("another_key_string", "another_value")
	val, exist = sm.Get("another_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeStr, val.Type())
	assert.EqualValues(t, "another_value", val.Str())

	sm.PutInt("another_key_int", 456)
	val, exist = sm.Get("another_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 456, val.Int())

	sm.PutDouble("another_key_double", 4.56)
	val, exist = sm.Get("another_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 4.56, val.Double())

	sm.PutBool("another_key_bool", false)
	val, exist = sm.Get("another_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.False(t, val.Bool())

	sm.PutEmptyBytes("another_key_bytes").FromRaw([]byte{1})
	val, exist = sm.Get("another_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{1}, val.Bytes().AsRaw())

	assert.True(t, sm.Remove("other_key_string"))
	assert.True(t, sm.Remove("other_key_int"))
	assert.True(t, sm.Remove("other_key_double"))
	assert.True(t, sm.Remove("other_key_bool"))
	assert.True(t, sm.Remove("other_key_bytes"))
	assert.True(t, sm.Remove("another_key_string"))
	assert.True(t, sm.Remove("another_key_int"))
	assert.True(t, sm.Remove("another_key_double"))
	assert.True(t, sm.Remove("another_key_bool"))
	assert.True(t, sm.Remove("another_key_bytes"))

	assert.False(t, sm.Remove("other_key_string"))
	assert.False(t, sm.Remove("another_key_string"))

	// Test that the initial key is still there.
	val, exist = sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeStr, val.Type())
	assert.EqualValues(t, "test_value", val.Str())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeEmpty, val.Type())
	assert.EqualValues(t, "", val.Str())

	_, exist = sm.Get("test_key3")
	assert.False(t, exist)

	// Test Sort
	assert.EqualValues(t, newMap(&origWithNil), sm.Sort())
}

func TestMapIterationNil(t *testing.T) {
	NewMap().Range(func(k string, v Value) bool {
		// Fail if any element is returned
		t.Fail()
		return true
	})
}

func TestMap_Range(t *testing.T) {
	rawMap := map[string]interface{}{
		"k_string": "123",
		"k_int":    int64(123),
		"k_double": float64(1.23),
		"k_bool":   true,
		"k_empty":  nil,
	}
	am := NewMap()
	am.FromRaw(rawMap)
	assert.Equal(t, 5, am.Len())

	calls := 0
	am.Range(func(k string, v Value) bool {
		calls++
		return false
	})
	assert.Equal(t, 1, calls)

	am.Range(func(k string, v Value) bool {
		assert.Equal(t, rawMap[k], v.AsRaw())
		delete(rawMap, k)
		return true
	})
	assert.EqualValues(t, 0, len(rawMap))
}

func TestMap_FromRaw(t *testing.T) {
	am := NewMap()
	am.FromRaw(map[string]interface{}{})
	assert.Equal(t, 0, am.Len())
	am.PutEmpty("k")
	assert.Equal(t, 1, am.Len())

	am.FromRaw(nil)
	assert.Equal(t, 0, am.Len())
	am.PutEmpty("k")
	assert.Equal(t, 1, am.Len())

	am.FromRaw(map[string]interface{}{
		"k_string": "123",
		"k_int":    123,
		"k_double": 1.23,
		"k_bool":   true,
		"k_null":   nil,
		"k_bytes":  []byte{1, 2, 3},
		"k_slice":  []interface{}{1, 2.1, "val"},
		"k_map": map[string]interface{}{
			"k_int":    1,
			"k_string": "val",
		},
	})
	assert.Equal(t, 8, am.Len())
	v, ok := am.Get("k_string")
	assert.True(t, ok)
	assert.Equal(t, "123", v.Str())
	v, ok = am.Get("k_int")
	assert.True(t, ok)
	assert.Equal(t, int64(123), v.Int())
	v, ok = am.Get("k_double")
	assert.True(t, ok)
	assert.Equal(t, 1.23, v.Double())
	v, ok = am.Get("k_null")
	assert.True(t, ok)
	assert.Equal(t, ValueTypeEmpty, v.Type())
	v, ok = am.Get("k_bytes")
	assert.True(t, ok)
	assert.Equal(t, []byte{1, 2, 3}, v.Bytes().AsRaw())
	v, ok = am.Get("k_slice")
	assert.True(t, ok)
	assert.Equal(t, []interface{}{int64(1), 2.1, "val"}, v.Slice().AsRaw())
	v, ok = am.Get("k_map")
	assert.True(t, ok)
	assert.Equal(t, map[string]interface{}{
		"k_int":    int64(1),
		"k_string": "val",
	}, v.Map().AsRaw())
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

func TestMap_CopyTo(t *testing.T) {
	dest := NewMap()
	// Test CopyTo to empty
	NewMap().CopyTo(dest)
	assert.EqualValues(t, 0, dest.Len())

	// Test CopyTo larger slice
	Map(internal.GenerateTestMap()).CopyTo(dest)
	assert.EqualValues(t, Map(internal.GenerateTestMap()), dest)

	// Test CopyTo same size slice
	Map(internal.GenerateTestMap()).CopyTo(dest)
	assert.EqualValues(t, Map(internal.GenerateTestMap()), dest)

	// Test CopyTo with an empty Value in the destination
	(*dest.getOrig())[0].Value = otlpcommon.AnyValue{}
	Map(internal.GenerateTestMap()).CopyTo(dest)
	assert.EqualValues(t, Map(internal.GenerateTestMap()), dest)
}

func TestMap_EnsureCapacity_Zero(t *testing.T) {
	am := NewMap()
	am.EnsureCapacity(0)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 0, cap(*am.getOrig()))
}

func TestMap_EnsureCapacity(t *testing.T) {
	am := NewMap()
	am.EnsureCapacity(5)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 5, cap(*am.getOrig()))
	am.EnsureCapacity(3)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 5, cap(*am.getOrig()))
	am.EnsureCapacity(8)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 8, cap(*am.getOrig()))
}

func TestMap_Clear(t *testing.T) {
	am := NewMap()
	assert.Nil(t, *am.getOrig())
	am.Clear()
	assert.Nil(t, *am.getOrig())
	am.EnsureCapacity(5)
	assert.NotNil(t, *am.getOrig())
	am.Clear()
	assert.Nil(t, *am.getOrig())
}

func TestMap_RemoveIf(t *testing.T) {
	am := NewMap()
	am.PutStr("k_string", "123")
	am.PutInt("k_int", int64(123))
	am.PutDouble("k_double", float64(1.23))
	am.PutBool("k_bool", true)
	am.PutEmpty("k_empty")

	assert.Equal(t, 5, am.Len())

	am.RemoveIf(func(key string, val Value) bool {
		return key == "k_int" || val.Type() == ValueTypeBool
	})
	assert.Equal(t, 3, am.Len())
	_, exists := am.Get("k_string")
	assert.True(t, exists)
	_, exists = am.Get("k_int")
	assert.False(t, exists)
	_, exists = am.Get("k_double")
	assert.True(t, exists)
	_, exists = am.Get("k_bool")
	assert.False(t, exists)
	_, exists = am.Get("k_empty")
	assert.True(t, exists)
}

func generateTestEmptyMap() Map {
	m := NewMap()
	m.FromRaw(map[string]interface{}{
		"k": map[string]interface{}(nil),
	})
	return m
}

func generateTestEmptySlice() Map {
	m := NewMap()
	m.FromRaw(map[string]interface{}{
		"k": []interface{}(nil),
	})
	return m
}

func generateTestIntMap() Map {
	m := NewMap()
	m.FromRaw(map[string]interface{}{
		"k": 123,
	})
	return m
}

func generateTestDoubleMap() Map {
	m := NewMap()
	m.FromRaw(map[string]interface{}{
		"k": 12.3,
	})
	return m
}

func generateTestBoolMap() Map {
	m := NewMap()
	m.FromRaw(map[string]interface{}{
		"k": true,
	})
	return m
}

func generateTestBytesMap() Map {
	m := NewMap()
	m.FromRaw(map[string]interface{}{
		"k": []byte{1, 2, 3, 4, 5},
	})
	return m
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

func TestAsString(t *testing.T) {
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
		expected interface{}
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
			expected: []interface{}{"strVal", int64(7), 18.6, false, nil},
		},
		{
			name:  "map",
			input: generateTestValueMap(),
			expected: map[string]interface{}{
				"mapKey":   map[string]interface{}{"keyOne": "valOne", "keyTwo": "valTwo"},
				"nullKey":  nil,
				"strKey":   "strVal",
				"arrKey":   []interface{}{"strOne", "strTwo"},
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

func TestMapAsRaw(t *testing.T) {
	raw := map[string]interface{}{
		"array":  []interface{}{false, []byte("test"), 12.9, int64(91), "another string"},
		"bool":   true,
		"bytes":  []byte("bytes value"),
		"double": 1.2,
		"empty":  nil,
		"int":    int64(900),
		"map":    map[string]interface{}{"str": "val"},
		"string": "string value",
	}
	m := NewMap()
	m.FromRaw(raw)
	assert.Equal(t, raw, m.AsRaw())
}

func TestNewValueFromRaw(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
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
			input: map[string]interface{}{
				"k": "v",
			},
			expected: func() Value {
				m := NewValueMap()
				m.Map().FromRaw(map[string]interface{}{"k": "v"})
				return m
			}(),
		},
		{
			name:  "empty map",
			input: map[string]interface{}{},
			expected: func() Value {
				m := NewValueMap()
				m.Map().FromRaw(map[string]interface{}{})
				return m
			}(),
		},
		{
			name:  "slice",
			input: []interface{}{"v1", "v2"},
			expected: (func() Value {
				s := NewValueSlice()
				s.Slice().FromRaw([]interface{}{"v1", "v2"})
				return s
			})(),
		},
		{
			name:  "empty slice",
			input: []interface{}{},
			expected: (func() Value {
				s := NewValueSlice()
				s.Slice().FromRaw([]interface{}{})
				return s
			})(),
		},
		{
			name:  "invalid value",
			input: ValueTypeDouble,
			expected: (func() Value {
				return NewValueStr("<Invalid value type pcommon.ValueType>")
			})(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := NewValueEmpty()
			actual.FromRaw(tt.input)
			assert.Equal(t, tt.expected, actual)
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
