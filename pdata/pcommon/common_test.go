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
	v := NewValueString("abc")
	assert.EqualValues(t, ValueTypeString, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v = NewValueInt(123)
	assert.EqualValues(t, ValueTypeInt, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v = NewValueDouble(3.4)
	assert.EqualValues(t, ValueTypeDouble, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v = NewValueBool(true)
	assert.EqualValues(t, ValueTypeBool, v.Type())
	assert.True(t, v.BoolVal())

	v = NewValueBytesEmpty()
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
	assert.EqualValues(t, "STRING", ValueTypeString.String())
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
	assert.Equal(t, NewMap(), m1.MapVal())
	assert.Equal(t, 0, m1.MapVal().Len())

	m1.MapVal().InsertDouble("double_key", 123)
	assert.Equal(t, 1, m1.MapVal().Len())
	got, exists := m1.MapVal().Get("double_key")
	assert.True(t, exists)
	assert.Equal(t, NewValueDouble(123), got)

	// Create a second map.
	m2 := NewValueMap()
	assert.Equal(t, 0, m2.MapVal().Len())

	// Modify the source map that was inserted.
	m2.MapVal().PutString("key_in_child", "somestr")
	assert.Equal(t, 1, m2.MapVal().Len())
	got, exists = m2.MapVal().Get("key_in_child")
	assert.True(t, exists)
	assert.Equal(t, NewValueString("somestr"), got)

	// Insert the second map as a child. This should perform a deep copy.
	m1.MapVal().Insert("child_map", m2)
	assert.EqualValues(t, 2, m1.MapVal().Len())
	got, exists = m1.MapVal().Get("double_key")
	assert.True(t, exists)
	assert.Equal(t, NewValueDouble(123), got)
	got, exists = m1.MapVal().Get("child_map")
	assert.True(t, exists)
	assert.Equal(t, m2, got)

	// Modify the source map m2 that was inserted into m1.
	m2.MapVal().UpdateString("key_in_child", "somestr2")
	assert.EqualValues(t, 1, m2.MapVal().Len())
	got, exists = m2.MapVal().Get("key_in_child")
	assert.True(t, exists)
	assert.Equal(t, NewValueString("somestr2"), got)

	// The child map inside m1 should not be modified.
	childMap, childMapExists := m1.MapVal().Get("child_map")
	require.True(t, childMapExists)
	got, exists = childMap.MapVal().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewValueString("somestr"), got)

	// Now modify the inserted map (not the source)
	childMap.MapVal().UpdateString("key_in_child", "somestr3")
	assert.EqualValues(t, 1, childMap.MapVal().Len())
	got, exists = childMap.MapVal().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewValueString("somestr3"), got)

	// The source child map should not be modified.
	got, exists = m2.MapVal().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewValueString("somestr2"), got)

	removed := m1.MapVal().Remove("double_key")
	assert.True(t, removed)
	assert.EqualValues(t, 1, m1.MapVal().Len())
	_, exists = m1.MapVal().Get("double_key")
	assert.False(t, exists)

	removed = m1.MapVal().Remove("child_map")
	assert.True(t, removed)
	assert.EqualValues(t, 0, m1.MapVal().Len())
	_, exists = m1.MapVal().Get("child_map")
	assert.False(t, exists)

	// Test nil KvlistValue case for MapVal() func.
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: nil}}
	m1 = newValue(orig)
	assert.EqualValues(t, Map{}, m1.MapVal())
}

func TestNilOrigSetValue(t *testing.T) {
	av := NewValueEmpty()
	av.SetStringVal("abc")
	assert.EqualValues(t, "abc", av.StringVal())

	av = NewValueEmpty()
	av.SetIntVal(123)
	assert.EqualValues(t, 123, av.IntVal())

	av = NewValueEmpty()
	av.SetBoolVal(true)
	assert.True(t, av.BoolVal())

	av = NewValueEmpty()
	av.SetDoubleVal(1.23)
	assert.EqualValues(t, 1.23, av.DoubleVal())

	av = NewValueEmpty()
	av.SetEmptyBytesVal().FromRaw([]byte{1, 2, 3})
	assert.Equal(t, []byte{1, 2, 3}, av.BytesVal().AsRaw())

	av = NewValueEmpty()
	av.SetEmptyMapVal().FromRaw(map[string]interface{}{"k": "v"})
	assert.Equal(t, map[string]interface{}{"k": "v"}, av.MapVal().AsRaw())

	av = NewValueEmpty()
	av.SetEmptySliceVal().FromRaw([]interface{}{int64(1), "val"})
	assert.Equal(t, []interface{}{int64(1), "val"}, av.SliceVal().AsRaw())
}

func TestValueEqual(t *testing.T) {
	av1 := NewValueEmpty()
	assert.True(t, av1.Equal(av1)) // nolint:gocritic
	av2 := NewValueEmpty()
	assert.True(t, av1.Equal(av2))

	av2 = NewValueString("abc")
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueString("abc")
	assert.True(t, av1.Equal(av2))

	av2 = NewValueString("edf")
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

	av2 = NewValueBytesEmpty()
	av2.BytesVal().FromRaw([]byte{1, 2, 3})
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueBytesEmpty()
	av1.BytesVal().FromRaw([]byte{1, 2, 4})
	assert.False(t, av1.Equal(av2))

	av1.BytesVal().SetAt(2, 3)
	assert.True(t, av1.Equal(av2))

	av1 = NewValueSlice()
	av1.SliceVal().AppendEmpty().SetIntVal(123)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av2 = NewValueSlice()
	av2.SliceVal().AppendEmpty().SetDoubleVal(123)
	assert.False(t, av1.Equal(av2))

	NewValueInt(123).CopyTo(av2.SliceVal().At(0))
	assert.True(t, av1.Equal(av2))

	av1.CopyTo(av2.SliceVal().AppendEmpty())
	assert.False(t, av1.Equal(av2))

	av1 = NewValueMap()
	av1.MapVal().PutString("foo", "bar")
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av2 = NewValueMap()
	av2.MapVal().PutString("foo", "bar")
	assert.True(t, av1.Equal(av2))

	fooVal, ok := av2.MapVal().Get("foo")
	assert.True(t, ok)
	fooVal.SetStringVal("not-bar")
	assert.False(t, av1.Equal(av2))
}

func TestMap(t *testing.T) {
	assert.EqualValues(t, 0, NewMap().Len())

	val, exist := NewMap().Get("test_key")
	assert.False(t, exist)
	assert.EqualValues(t, newValue(nil), val)

	insertMap := NewMap()
	insertMap.Insert("k", NewValueString("v"))
	assert.EqualValues(t, Map(internal.GenerateTestMap()), insertMap)

	insertMapString := NewMap()
	insertMapString.InsertString("k", "v")
	assert.EqualValues(t, Map(internal.GenerateTestMap()), insertMapString)

	insertMapNull := NewMap()
	insertMapNull.Insert("k", NewValueEmpty())
	assert.EqualValues(t, generateTestEmptyMap(), insertMapNull)

	insertMapInt := NewMap()
	insertMapInt.InsertInt("k", 123)
	assert.EqualValues(t, generateTestIntMap(), insertMapInt)

	insertMapDouble := NewMap()
	insertMapDouble.InsertDouble("k", 12.3)
	assert.EqualValues(t, generateTestDoubleMap(), insertMapDouble)

	insertMapBool := NewMap()
	insertMapBool.InsertBool("k", true)
	assert.EqualValues(t, generateTestBoolMap(), insertMapBool)

	insertMapBytes := NewMap()
	insertMapBytes.InsertBytes("k", NewImmutableByteSlice([]byte{1, 2, 3, 4, 5}))
	assert.EqualValues(t, generateTestBytesMap(), insertMapBytes)

	updateMapString := NewMap()
	updateMapString.UpdateString("k", "v")
	assert.EqualValues(t, NewMap(), updateMapString)

	updateMapInt := NewMap()
	updateMapInt.UpdateInt("k", 123)
	assert.EqualValues(t, NewMap(), updateMapInt)

	updateMapDouble := NewMap()
	updateMapDouble.UpdateDouble("k", 12.3)
	assert.EqualValues(t, NewMap(), updateMapDouble)

	updateMapBool := NewMap()
	updateMapBool.UpdateBool("k", true)
	assert.EqualValues(t, NewMap(), updateMapBool)

	updateMapBytes := NewMap()
	updateMapBytes.UpdateBytes("k", NewImmutableByteSlice([]byte{1, 2, 3}))
	assert.EqualValues(t, NewMap(), updateMapBytes)

	putMapString := NewMap()
	putMapString.PutString("k", "v")
	assert.EqualValues(t, Map(internal.GenerateTestMap()), putMapString)

	putMapInt := NewMap()
	putMapInt.PutInt("k", 123)
	assert.EqualValues(t, generateTestIntMap(), putMapInt)

	putMapDouble := NewMap()
	putMapDouble.PutDouble("k", 12.3)
	assert.EqualValues(t, generateTestDoubleMap(), putMapDouble)

	putMapBool := NewMap()
	putMapBool.PutBool("k", true)
	assert.EqualValues(t, generateTestBoolMap(), putMapBool)

	putMapBytes := NewMap()
	putMapBytes.PutEmptyBytes("k").FromRaw([]byte{1, 2, 3, 4, 5})
	assert.EqualValues(t, generateTestBytesMap(), putMapBytes)

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

	v.SetBoolVal(true)
	assert.EqualValues(t, map[string]interface{}{
		"k1": true,
	}, m.AsRaw())

	v = m.PutEmpty("k1")
	v.SetIntVal(1)
	v2, ok := m.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, int64(1), v2.IntVal())
}

func TestMapPutEmptyMap(t *testing.T) {
	m := NewMap()
	childMap := m.PutEmptyMap("k1")
	assert.EqualValues(t, map[string]interface{}{
		"k1": map[string]interface{}{},
	}, m.AsRaw())
	childMap.PutEmptySlice("k2").AppendEmpty().SetStringVal("val")
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
	childSlice.AppendEmpty().SetDoubleVal(1.1)
	assert.EqualValues(t, map[string]interface{}{
		"k": []interface{}{1.1},
	}, m.AsRaw())

	m.PutEmptySlice("k")
	assert.EqualValues(t, map[string]interface{}{
		"k": []interface{}{},
	}, m.AsRaw())
	childSliceVal, ok := m.Get("k")
	assert.True(t, ok)
	childSliceVal.SliceVal().AppendEmpty().SetEmptySliceVal().AppendEmpty().SetStringVal("val")
	assert.EqualValues(t, map[string]interface{}{
		"k": []interface{}{[]interface{}{"val"}},
	}, m.AsRaw())
}

func TestMapPutEmptyBytes(t *testing.T) {
	m := NewMap()
	b := m.PutEmptyBytes("k")
	bv, ok := m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte(nil), bv.BytesVal().AsRaw())
	b.FromRaw([]byte{1, 2, 3})
	bv, ok = m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte{1, 2, 3}, bv.BytesVal().AsRaw())

	m.PutEmptyBytes("k")
	bv, ok = m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte(nil), bv.BytesVal().AsRaw())
	bv.BytesVal().FromRaw([]byte{3, 2, 1})
	bv, ok = m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte{3, 2, 1}, bv.BytesVal().AsRaw())
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
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "test_value", val.StringVal())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeEmpty, val.Type())
	assert.EqualValues(t, "", val.StringVal())

	sm.Insert("other_key", NewValueString("other_value"))
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.InsertString("other_key_string", "other_value")
	val, exist = sm.Get("other_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.InsertInt("other_key_int", 123)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 123, val.IntVal())

	sm.InsertDouble("other_key_double", 1.23)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 1.23, val.DoubleVal())

	sm.InsertBool("other_key_bool", true)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.True(t, val.BoolVal())

	sm.InsertBytes("other_key_bytes", NewImmutableByteSlice([]byte{1, 2, 3}))
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{1, 2, 3}, val.BytesVal().AsRaw())

	sm.UpdateString("other_key_string", "yet_another_value")
	val, exist = sm.Get("other_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpdateInt("other_key_int", 456)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 456, val.IntVal())

	sm.UpdateDouble("other_key_double", 4.56)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 4.56, val.DoubleVal())

	sm.UpdateBool("other_key_bool", false)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.False(t, val.BoolVal())

	sm.UpdateBytes("other_key_bytes", NewImmutableByteSlice([]byte{4, 5, 6}))
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{4, 5, 6}, val.BytesVal().AsRaw())

	sm.PutString("other_key_string", "other_value")
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.PutInt("other_key_int", 123)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 123, val.IntVal())

	sm.PutDouble("other_key_double", 1.23)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 1.23, val.DoubleVal())

	sm.PutBool("other_key_bool", true)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.True(t, val.BoolVal())

	sm.PutEmptyBytes("other_key_bytes").FromRaw([]byte{7, 8, 9})
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{7, 8, 9}, val.BytesVal().AsRaw())

	sm.PutString("yet_another_key_string", "yet_another_value")
	val, exist = sm.Get("yet_another_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.PutInt("yet_another_key_int", 456)
	val, exist = sm.Get("yet_another_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 456, val.IntVal())

	sm.PutDouble("yet_another_key_double", 4.56)
	val, exist = sm.Get("yet_another_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 4.56, val.DoubleVal())

	sm.PutBool("yet_another_key_bool", false)
	val, exist = sm.Get("yet_another_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.False(t, val.BoolVal())

	sm.PutEmptyBytes("yet_another_key_bytes").FromRaw([]byte{1})
	val, exist = sm.Get("yet_another_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{1}, val.BytesVal().AsRaw())

	assert.True(t, sm.Remove("other_key"))
	assert.True(t, sm.Remove("other_key_string"))
	assert.True(t, sm.Remove("other_key_int"))
	assert.True(t, sm.Remove("other_key_double"))
	assert.True(t, sm.Remove("other_key_bool"))
	assert.True(t, sm.Remove("other_key_bytes"))
	assert.True(t, sm.Remove("yet_another_key_string"))
	assert.True(t, sm.Remove("yet_another_key_int"))
	assert.True(t, sm.Remove("yet_another_key_double"))
	assert.True(t, sm.Remove("yet_another_key_bool"))
	assert.True(t, sm.Remove("yet_another_key_bytes"))
	assert.False(t, sm.Remove("other_key"))
	assert.False(t, sm.Remove("yet_another_key"))

	// Test that the initial key is still there.
	val, exist = sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "test_value", val.StringVal())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeEmpty, val.Type())
	assert.EqualValues(t, "", val.StringVal())

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
		assert.Equal(t, rawMap[k], v.asRaw())
		delete(rawMap, k)
		return true
	})
	assert.EqualValues(t, 0, len(rawMap))
}

func TestMap_InitFromRawDeprecated(t *testing.T) {
	am := NewMapFromRaw(map[string]interface{}(nil))
	assert.EqualValues(t, NewMap(), am)

	rawMap := map[string]interface{}{
		"k_string": "123",
		"k_int":    123,
		"k_double": 1.23,
		"k_bool":   true,
		"k_null":   nil,
		"k_bytes":  []byte{1, 2, 3},
	}
	rawOrig := []otlpcommon.KeyValue{
		newAttributeKeyValueString("k_string", "123"),
		newAttributeKeyValueInt("k_int", 123),
		newAttributeKeyValueDouble("k_double", 1.23),
		newAttributeKeyValueBool("k_bool", true),
		newAttributeKeyValueNull("k_null"),
		newAttributeKeyValueBytes("k_bytes", NewImmutableByteSlice([]byte{1, 2, 3})),
	}
	am = NewMapFromRaw(rawMap)
	assert.EqualValues(t, newMap(&rawOrig).Sort(), am.Sort())
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
	assert.Equal(t, "123", v.StringVal())
	v, ok = am.Get("k_int")
	assert.True(t, ok)
	assert.Equal(t, int64(123), v.IntVal())
	v, ok = am.Get("k_double")
	assert.True(t, ok)
	assert.Equal(t, 1.23, v.DoubleVal())
	v, ok = am.Get("k_null")
	assert.True(t, ok)
	assert.Equal(t, ValueTypeEmpty, v.Type())
	v, ok = am.Get("k_bytes")
	assert.True(t, ok)
	assert.Equal(t, []byte{1, 2, 3}, v.BytesVal().AsRaw())
	v, ok = am.Get("k_slice")
	assert.True(t, ok)
	assert.Equal(t, []interface{}{int64(1), 2.1, "val"}, v.SliceVal().AsRaw())
	v, ok = am.Get("k_map")
	assert.True(t, ok)
	assert.Equal(t, map[string]interface{}{
		"k_int":    int64(1),
		"k_string": "val",
	}, v.MapVal().AsRaw())
}

func TestValue_CopyTo(t *testing.T) {
	// Test nil KvlistValue case for MapVal() func.
	dest := NewValueEmpty()
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: nil}}
	newValue(orig).CopyTo(dest)
	assert.Nil(t, dest.getOrig().Value.(*otlpcommon.AnyValue_KvlistValue).KvlistValue)

	// Test nil ArrayValue case for SliceVal() func.
	dest = NewValueEmpty()
	orig = &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: nil}}
	newValue(orig).CopyTo(dest)
	assert.Nil(t, dest.getOrig().Value.(*otlpcommon.AnyValue_ArrayValue).ArrayValue)

	// Test copy empty value.
	orig = &otlpcommon.AnyValue{}
	newValue(orig).CopyTo(dest)
	assert.Nil(t, dest.getOrig().Value)
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

func TestValue_copyTo(t *testing.T) {
	av := NewValueEmpty()
	destVal := otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{}}
	av.copyTo(&destVal)
	assert.EqualValues(t, nil, destVal.Value)
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
	am.PutString("k_string", "123")
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
		"k": nil,
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
	assert.EqualValues(t, NewSlice(), a1.SliceVal())
	assert.EqualValues(t, 0, a1.SliceVal().Len())

	a1.SliceVal().AppendEmpty().SetDoubleVal(123)
	assert.EqualValues(t, 1, a1.SliceVal().Len())
	assert.EqualValues(t, NewValueDouble(123), a1.SliceVal().At(0))
	// Create a second array.
	a2 := NewValueSlice()
	assert.EqualValues(t, 0, a2.SliceVal().Len())

	a2.SliceVal().AppendEmpty().SetStringVal("somestr")
	assert.EqualValues(t, 1, a2.SliceVal().Len())
	assert.EqualValues(t, NewValueString("somestr"), a2.SliceVal().At(0))

	// Insert the second array as a child.
	a2.CopyTo(a1.SliceVal().AppendEmpty())
	assert.EqualValues(t, 2, a1.SliceVal().Len())
	assert.EqualValues(t, NewValueDouble(123), a1.SliceVal().At(0))
	assert.EqualValues(t, a2, a1.SliceVal().At(1))

	// Check that the array was correctly inserted.
	childArray := a1.SliceVal().At(1)
	assert.EqualValues(t, ValueTypeSlice, childArray.Type())
	assert.EqualValues(t, 1, childArray.SliceVal().Len())

	v := childArray.SliceVal().At(0)
	assert.EqualValues(t, ValueTypeString, v.Type())
	assert.EqualValues(t, "somestr", v.StringVal())

	// Test nil values case for SliceVal() func.
	a1 = newValue(&otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: nil}})
	assert.EqualValues(t, newSlice(nil), a1.SliceVal())
}

func TestSliceWithNilValues(t *testing.T) {
	origWithNil := []otlpcommon.AnyValue{
		{},
		{Value: &otlpcommon.AnyValue_StringValue{StringValue: "test_value"}},
	}
	sm := newSlice(&origWithNil)

	val := sm.At(0)
	assert.EqualValues(t, ValueTypeEmpty, val.Type())
	assert.EqualValues(t, "", val.StringVal())

	val = sm.At(1)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "test_value", val.StringVal())

	sm.AppendEmpty().SetStringVal("other_value")
	val = sm.At(2)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())
}

func TestAsString(t *testing.T) {
	tests := []struct {
		name     string
		input    Value
		expected string
	}{
		{
			name:     "string",
			input:    NewValueString("string value"),
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
			input:    NewValueString("value"),
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
			actual := test.input.asRaw()
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
			expected: NewValueString("text"),
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
			name:     "bytes",
			input:    []byte{1, 2, 3},
			expected: NewValueBytes(NewImmutableByteSlice([]byte{1, 2, 3})),
		},
		{
			name: "map",
			input: map[string]interface{}{
				"k": "v",
			},
			expected: func() Value {
				m := NewValueMap()
				m.MapVal().FromRaw(map[string]interface{}{
					"k": "v",
				})
				return m
			}(),
		},
		{
			name:  "empty map",
			input: map[string]interface{}{},
			expected: func() Value {
				m := NewValueMap()
				m.MapVal().FromRaw(map[string]interface{}{})
				return m
			}(),
		},
		{
			name:  "slice",
			input: []interface{}{"v1", "v2"},
			expected: (func() Value {
				s := NewValueSlice()
				s.SliceVal().FromRaw([]interface{}{"v1", "v2"})
				return s
			})(),
		},
		{
			name:  "empty slice",
			input: []interface{}{},
			expected: (func() Value {
				s := NewValueSlice()
				s.SliceVal().FromRaw([]interface{}{})
				return s
			})(),
		},
		{
			name:  "invalid value",
			input: ValueTypeDouble,
			expected: (func() Value {
				return NewValueString("<Invalid value type pcommon.ValueType>")
			})(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := NewValueEmpty()
			actual.fromRaw(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func generateTestValueMap() Value {
	ret := NewValueMap()
	attrMap := ret.MapVal()
	attrMap.PutString("strKey", "strVal")
	attrMap.PutInt("intKey", 7)
	attrMap.PutDouble("floatKey", 18.6)
	attrMap.PutBool("boolKey", false)
	attrMap.PutEmpty("nullKey")

	m := attrMap.PutEmptyMap("mapKey")
	m.PutString("keyOne", "valOne")
	m.PutString("keyTwo", "valTwo")

	s := attrMap.PutEmptySlice("arrKey")
	s.AppendEmpty().SetStringVal("strOne")
	s.AppendEmpty().SetStringVal("strTwo")

	return ret
}

func generateTestValueSlice() Value {
	ret := NewValueSlice()
	attrArr := ret.SliceVal()
	attrArr.AppendEmpty().SetStringVal("strVal")
	attrArr.AppendEmpty().SetIntVal(7)
	attrArr.AppendEmpty().SetDoubleVal(18.6)
	attrArr.AppendEmpty().SetBoolVal(false)
	attrArr.AppendEmpty()
	return ret
}

func generateTestValueBytes() Value {
	v := NewValueBytesEmpty()
	v.BytesVal().FromRaw([]byte("String bytes"))
	return v
}
