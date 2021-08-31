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

package pdata

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	otlpcommon "go.opentelemetry.io/collector/model/internal/data/protogen/common/v1"
)

func TestAttributeValue(t *testing.T) {
	v := NewAttributeValueString("abc")
	assert.EqualValues(t, AttributeValueTypeString, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v = NewAttributeValueInt(123)
	assert.EqualValues(t, AttributeValueTypeInt, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v = NewAttributeValueDouble(3.4)
	assert.EqualValues(t, AttributeValueTypeDouble, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v = NewAttributeValueBool(true)
	assert.EqualValues(t, AttributeValueTypeBool, v.Type())
	assert.True(t, v.BoolVal())

	v = NewAttributeValueNull()
	assert.EqualValues(t, AttributeValueTypeNull, v.Type())

	v.SetStringVal("abc")
	assert.EqualValues(t, AttributeValueTypeString, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v.SetIntVal(123)
	assert.EqualValues(t, AttributeValueTypeInt, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v.SetDoubleVal(3.4)
	assert.EqualValues(t, AttributeValueTypeDouble, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v.SetBoolVal(true)
	assert.EqualValues(t, AttributeValueTypeBool, v.Type())
	assert.True(t, v.BoolVal())

	bytesValue := []byte{1, 2, 3, 4}
	v = NewAttributeValueBytes(bytesValue)
	assert.EqualValues(t, AttributeValueTypeBytes, v.Type())
	assert.EqualValues(t, bytesValue, v.BytesVal())
}

func TestAttributeValueType(t *testing.T) {
	assert.EqualValues(t, "NULL", AttributeValueTypeNull.String())
	assert.EqualValues(t, "STRING", AttributeValueTypeString.String())
	assert.EqualValues(t, "BOOL", AttributeValueTypeBool.String())
	assert.EqualValues(t, "INT", AttributeValueTypeInt.String())
	assert.EqualValues(t, "DOUBLE", AttributeValueTypeDouble.String())
	assert.EqualValues(t, "MAP", AttributeValueTypeMap.String())
	assert.EqualValues(t, "ARRAY", AttributeValueTypeArray.String())
	assert.EqualValues(t, "BYTES", AttributeValueTypeBytes.String())
}

func TestAttributeValueMap(t *testing.T) {
	m1 := NewAttributeValueMap()
	assert.Equal(t, AttributeValueTypeMap, m1.Type())
	assert.Equal(t, NewAttributeMap(), m1.MapVal())
	assert.Equal(t, 0, m1.MapVal().Len())

	m1.MapVal().InsertDouble("double_key", 123)
	assert.Equal(t, 1, m1.MapVal().Len())
	got, exists := m1.MapVal().Get("double_key")
	assert.True(t, exists)
	assert.Equal(t, NewAttributeValueDouble(123), got)

	// Create a second map.
	m2 := NewAttributeValueMap()
	assert.Equal(t, 0, m2.MapVal().Len())

	// Modify the source map that was inserted.
	m2.MapVal().UpsertString("key_in_child", "somestr")
	assert.Equal(t, 1, m2.MapVal().Len())
	got, exists = m2.MapVal().Get("key_in_child")
	assert.True(t, exists)
	assert.Equal(t, NewAttributeValueString("somestr"), got)

	// Insert the second map as a child. This should perform a deep copy.
	m1.MapVal().Insert("child_map", m2)
	assert.EqualValues(t, 2, m1.MapVal().Len())
	got, exists = m1.MapVal().Get("double_key")
	assert.True(t, exists)
	assert.Equal(t, NewAttributeValueDouble(123), got)
	got, exists = m1.MapVal().Get("child_map")
	assert.True(t, exists)
	assert.Equal(t, m2, got)

	// Modify the source map m2 that was inserted into m1.
	m2.MapVal().UpdateString("key_in_child", "somestr2")
	assert.EqualValues(t, 1, m2.MapVal().Len())
	got, exists = m2.MapVal().Get("key_in_child")
	assert.True(t, exists)
	assert.Equal(t, NewAttributeValueString("somestr2"), got)

	// The child map inside m1 should not be modified.
	childMap, childMapExists := m1.MapVal().Get("child_map")
	require.True(t, childMapExists)
	got, exists = childMap.MapVal().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewAttributeValueString("somestr"), got)

	// Now modify the inserted map (not the source)
	childMap.MapVal().UpdateString("key_in_child", "somestr3")
	assert.EqualValues(t, 1, childMap.MapVal().Len())
	got, exists = childMap.MapVal().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewAttributeValueString("somestr3"), got)

	// The source child map should not be modified.
	got, exists = m2.MapVal().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewAttributeValueString("somestr2"), got)

	deleted := m1.MapVal().Delete("double_key")
	assert.True(t, deleted)
	assert.EqualValues(t, 1, m1.MapVal().Len())
	_, exists = m1.MapVal().Get("double_key")
	assert.False(t, exists)

	deleted = m1.MapVal().Delete("child_map")
	assert.True(t, deleted)
	assert.EqualValues(t, 0, m1.MapVal().Len())
	_, exists = m1.MapVal().Get("child_map")
	assert.False(t, exists)

	// Test nil KvlistValue case for MapVal() func.
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: nil}}
	m1 = AttributeValue{orig: orig}
	assert.EqualValues(t, NewAttributeMap(), m1.MapVal())
}

func TestNilOrigSetAttributeValue(t *testing.T) {
	av := NewAttributeValueNull()
	av.SetStringVal("abc")
	assert.EqualValues(t, "abc", av.StringVal())

	av = NewAttributeValueNull()
	av.SetIntVal(123)
	assert.EqualValues(t, 123, av.IntVal())

	av = NewAttributeValueNull()
	av.SetBoolVal(true)
	assert.True(t, av.BoolVal())

	av = NewAttributeValueNull()
	av.SetDoubleVal(1.23)
	assert.EqualValues(t, 1.23, av.DoubleVal())

	av = NewAttributeValueNull()
	av.SetBytesVal([]byte{1, 2, 3})
	assert.Equal(t, []byte{1, 2, 3}, av.BytesVal())
}

func TestAttributeValueEqual(t *testing.T) {
	av1 := NewAttributeValueNull()
	av2 := NewAttributeValueNull()
	assert.True(t, av1.Equal(av2))

	av2 = NewAttributeValueString("abc")
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewAttributeValueString("abc")
	assert.True(t, av1.Equal(av2))

	av2 = NewAttributeValueString("edf")
	assert.False(t, av1.Equal(av2))

	av2 = NewAttributeValueInt(123)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewAttributeValueInt(234)
	assert.False(t, av1.Equal(av2))

	av1 = NewAttributeValueInt(123)
	assert.True(t, av1.Equal(av2))

	av2 = NewAttributeValueDouble(123)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewAttributeValueDouble(234)
	assert.False(t, av1.Equal(av2))

	av1 = NewAttributeValueDouble(123)
	assert.True(t, av1.Equal(av2))

	av2 = NewAttributeValueBool(false)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewAttributeValueBool(true)
	assert.False(t, av1.Equal(av2))

	av1 = NewAttributeValueBool(false)
	assert.True(t, av1.Equal(av2))

	av2 = NewAttributeValueBytes([]byte{1, 2, 3})
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewAttributeValueBytes([]byte{1, 2, 4})
	assert.False(t, av1.Equal(av2))

	av1 = NewAttributeValueBytes([]byte{1, 2, 3})
	assert.True(t, av1.Equal(av2))

	av1 = NewAttributeValueArray()
	av1.ArrayVal().AppendEmpty().SetIntVal(123)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av2 = NewAttributeValueArray()
	av2.ArrayVal().AppendEmpty().SetDoubleVal(123)
	assert.False(t, av1.Equal(av2))

	NewAttributeValueInt(123).CopyTo(av2.ArrayVal().At(0))
	assert.True(t, av1.Equal(av2))

	av1.CopyTo(av2.ArrayVal().AppendEmpty())
	assert.False(t, av1.Equal(av2))
	assert.True(t, av1.Equal(av1))

	av1 = NewAttributeValueMap()
	av1.MapVal().InitFromMap(map[string]AttributeValue{
		"foo": NewAttributeValueString("bar"),
	})
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av2 = NewAttributeValueMap()
	av2.MapVal().InitFromMap(map[string]AttributeValue{
		"foo": NewAttributeValueString("bar"),
	})
	assert.True(t, av1.Equal(av2))

	fooVal, ok := av2.MapVal().Get("foo")
	if !ok {
		assert.Fail(t, "expected to find value with key foo")
	}
	fooVal.SetStringVal("not-bar")
	assert.False(t, av1.Equal(av2))
}

func TestNilAttributeMap(t *testing.T) {
	assert.EqualValues(t, 0, NewAttributeMap().Len())

	val, exist := NewAttributeMap().Get("test_key")
	assert.False(t, exist)
	assert.EqualValues(t, AttributeValue{nil}, val)

	insertMap := NewAttributeMap()
	insertMap.Insert("k", NewAttributeValueString("v"))
	assert.EqualValues(t, generateTestAttributeMap(), insertMap)

	insertMapString := NewAttributeMap()
	insertMapString.InsertString("k", "v")
	assert.EqualValues(t, generateTestAttributeMap(), insertMapString)

	insertMapNull := NewAttributeMap()
	insertMapNull.InsertNull("k")
	assert.EqualValues(t, generateTestNullAttributeMap(), insertMapNull)

	insertMapInt := NewAttributeMap()
	insertMapInt.InsertInt("k", 123)
	assert.EqualValues(t, generateTestIntAttributeMap(), insertMapInt)

	insertMapDouble := NewAttributeMap()
	insertMapDouble.InsertDouble("k", 12.3)
	assert.EqualValues(t, generateTestDoubleAttributeMap(), insertMapDouble)

	insertMapBool := NewAttributeMap()
	insertMapBool.InsertBool("k", true)
	assert.EqualValues(t, generateTestBoolAttributeMap(), insertMapBool)

	insertMapBytes := NewAttributeMap()
	insertMapBytes.InsertBytes("k", []byte{1, 2, 3, 4, 5})
	assert.EqualValues(t, generateTestBytesAttributeMap(), insertMapBytes)

	updateMap := NewAttributeMap()
	updateMap.Update("k", NewAttributeValueString("v"))
	assert.EqualValues(t, NewAttributeMap(), updateMap)

	updateMapString := NewAttributeMap()
	updateMapString.UpdateString("k", "v")
	assert.EqualValues(t, NewAttributeMap(), updateMapString)

	updateMapInt := NewAttributeMap()
	updateMapInt.UpdateInt("k", 123)
	assert.EqualValues(t, NewAttributeMap(), updateMapInt)

	updateMapDouble := NewAttributeMap()
	updateMapDouble.UpdateDouble("k", 12.3)
	assert.EqualValues(t, NewAttributeMap(), updateMapDouble)

	updateMapBool := NewAttributeMap()
	updateMapBool.UpdateBool("k", true)
	assert.EqualValues(t, NewAttributeMap(), updateMapBool)

	updateMapBytes := NewAttributeMap()
	updateMapBytes.UpdateBytes("k", []byte{1, 2, 3})
	assert.EqualValues(t, NewAttributeMap(), updateMapBytes)

	upsertMap := NewAttributeMap()
	upsertMap.Upsert("k", NewAttributeValueString("v"))
	assert.EqualValues(t, generateTestAttributeMap(), upsertMap)

	upsertMapString := NewAttributeMap()
	upsertMapString.UpsertString("k", "v")
	assert.EqualValues(t, generateTestAttributeMap(), upsertMapString)

	upsertMapInt := NewAttributeMap()
	upsertMapInt.UpsertInt("k", 123)
	assert.EqualValues(t, generateTestIntAttributeMap(), upsertMapInt)

	upsertMapDouble := NewAttributeMap()
	upsertMapDouble.UpsertDouble("k", 12.3)
	assert.EqualValues(t, generateTestDoubleAttributeMap(), upsertMapDouble)

	upsertMapBool := NewAttributeMap()
	upsertMapBool.UpsertBool("k", true)
	assert.EqualValues(t, generateTestBoolAttributeMap(), upsertMapBool)

	upsertMapBytes := NewAttributeMap()
	upsertMapBytes.UpsertBytes("k", []byte{1, 2, 3, 4, 5})
	assert.EqualValues(t, generateTestBytesAttributeMap(), upsertMapBytes)

	deleteMap := NewAttributeMap()
	assert.False(t, deleteMap.Delete("k"))
	assert.EqualValues(t, NewAttributeMap(), deleteMap)

	// Test Sort
	assert.EqualValues(t, NewAttributeMap(), NewAttributeMap().Sort())
}

func TestAttributeMapWithEmpty(t *testing.T) {
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
	sm := AttributeMap{
		orig: &origWithNil,
	}
	val, exist := sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "test_value", val.StringVal())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeNull, val.Type())
	assert.EqualValues(t, "", val.StringVal())

	sm.Insert("other_key", NewAttributeValueString("other_value"))
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.InsertString("other_key_string", "other_value")
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.InsertInt("other_key_int", 123)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeInt, val.Type())
	assert.EqualValues(t, 123, val.IntVal())

	sm.InsertDouble("other_key_double", 1.23)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeDouble, val.Type())
	assert.EqualValues(t, 1.23, val.DoubleVal())

	sm.InsertBool("other_key_bool", true)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeBool, val.Type())
	assert.True(t, val.BoolVal())

	sm.InsertBytes("other_key_bytes", []byte{1, 2, 3})
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{1, 2, 3}, val.BytesVal())

	sm.Update("other_key", NewAttributeValueString("yet_another_value"))
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpdateString("other_key_string", "yet_another_value")
	val, exist = sm.Get("other_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpdateInt("other_key_int", 456)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeInt, val.Type())
	assert.EqualValues(t, 456, val.IntVal())

	sm.UpdateDouble("other_key_double", 4.56)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeDouble, val.Type())
	assert.EqualValues(t, 4.56, val.DoubleVal())

	sm.UpdateBool("other_key_bool", false)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeBool, val.Type())
	assert.False(t, val.BoolVal())

	sm.UpdateBytes("other_key_bytes", []byte{4, 5, 6})
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{4, 5, 6}, val.BytesVal())

	sm.Upsert("other_key", NewAttributeValueString("other_value"))
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.UpsertString("other_key_string", "other_value")
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.UpsertInt("other_key_int", 123)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeInt, val.Type())
	assert.EqualValues(t, 123, val.IntVal())

	sm.UpsertDouble("other_key_double", 1.23)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeDouble, val.Type())
	assert.EqualValues(t, 1.23, val.DoubleVal())

	sm.UpsertBool("other_key_bool", true)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeBool, val.Type())
	assert.True(t, val.BoolVal())

	sm.UpsertBytes("other_key_bytes", []byte{7, 8, 9})
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{7, 8, 9}, val.BytesVal())

	sm.Upsert("yet_another_key", NewAttributeValueString("yet_another_value"))
	val, exist = sm.Get("yet_another_key")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpsertString("yet_another_key_string", "yet_another_value")
	val, exist = sm.Get("yet_another_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpsertInt("yet_another_key_int", 456)
	val, exist = sm.Get("yet_another_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeInt, val.Type())
	assert.EqualValues(t, 456, val.IntVal())

	sm.UpsertDouble("yet_another_key_double", 4.56)
	val, exist = sm.Get("yet_another_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeDouble, val.Type())
	assert.EqualValues(t, 4.56, val.DoubleVal())

	sm.UpsertBool("yet_another_key_bool", false)
	val, exist = sm.Get("yet_another_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeBool, val.Type())
	assert.False(t, val.BoolVal())

	sm.UpsertBytes("yet_another_key_bytes", []byte{1})
	val, exist = sm.Get("yet_another_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{1}, val.BytesVal())

	assert.True(t, sm.Delete("other_key"))
	assert.True(t, sm.Delete("other_key_string"))
	assert.True(t, sm.Delete("other_key_int"))
	assert.True(t, sm.Delete("other_key_double"))
	assert.True(t, sm.Delete("other_key_bool"))
	assert.True(t, sm.Delete("other_key_bytes"))
	assert.True(t, sm.Delete("yet_another_key"))
	assert.True(t, sm.Delete("yet_another_key_string"))
	assert.True(t, sm.Delete("yet_another_key_int"))
	assert.True(t, sm.Delete("yet_another_key_double"))
	assert.True(t, sm.Delete("yet_another_key_bool"))
	assert.True(t, sm.Delete("yet_another_key_bytes"))
	assert.False(t, sm.Delete("other_key"))
	assert.False(t, sm.Delete("yet_another_key"))

	// Test that the initial key is still there.
	val, exist = sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "test_value", val.StringVal())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.EqualValues(t, AttributeValueTypeNull, val.Type())
	assert.EqualValues(t, "", val.StringVal())

	_, exist = sm.Get("test_key3")
	assert.False(t, exist)

	// Test Sort
	assert.EqualValues(t, AttributeMap{orig: &origWithNil}, sm.Sort())
}

func TestAttributeMapIterationNil(t *testing.T) {
	NewAttributeMap().Range(func(k string, v AttributeValue) bool {
		// Fail if any element is returned
		t.Fail()
		return true
	})
}

func TestAttributeMap_Range(t *testing.T) {
	rawMap := map[string]AttributeValue{
		"k_string": NewAttributeValueString("123"),
		"k_int":    NewAttributeValueInt(123),
		"k_double": NewAttributeValueDouble(1.23),
		"k_bool":   NewAttributeValueBool(true),
		"k_null":   NewAttributeValueNull(),
		"k_bytes":  NewAttributeValueBytes([]byte{}),
	}
	am := NewAttributeMapFromMap(rawMap)
	assert.Equal(t, 6, am.Len())

	calls := 0
	am.Range(func(k string, v AttributeValue) bool {
		calls++
		return false
	})
	assert.Equal(t, 1, calls)

	am.Range(func(k string, v AttributeValue) bool {
		assert.True(t, v.Equal(rawMap[k]))
		delete(rawMap, k)
		return true
	})
	assert.EqualValues(t, 0, len(rawMap))
}

func TestAttributeMap_InitFromMap(t *testing.T) {
	am := NewAttributeMapFromMap(map[string]AttributeValue(nil))
	assert.EqualValues(t, NewAttributeMap(), am)

	rawMap := map[string]AttributeValue{
		"k_string": NewAttributeValueString("123"),
		"k_int":    NewAttributeValueInt(123),
		"k_double": NewAttributeValueDouble(1.23),
		"k_bool":   NewAttributeValueBool(true),
		"k_null":   NewAttributeValueNull(),
		"k_bytes":  NewAttributeValueBytes([]byte{1, 2, 3}),
	}
	rawOrig := []otlpcommon.KeyValue{
		newAttributeKeyValueString("k_string", "123"),
		newAttributeKeyValueInt("k_int", 123),
		newAttributeKeyValueDouble("k_double", 1.23),
		newAttributeKeyValueBool("k_bool", true),
		newAttributeKeyValueNull("k_null"),
		newAttributeKeyValueBytes("k_bytes", []byte{1, 2, 3}),
	}
	am = NewAttributeMapFromMap(rawMap)
	assert.EqualValues(t, AttributeMap{orig: &rawOrig}.Sort(), am.Sort())
}

func TestAttributeValue_CopyTo(t *testing.T) {
	// Test nil KvlistValue case for MapVal() func.
	dest := NewAttributeValueNull()
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: nil}}
	AttributeValue{orig: orig}.CopyTo(dest)
	assert.Nil(t, dest.orig.Value.(*otlpcommon.AnyValue_KvlistValue).KvlistValue)

	// Test nil ArrayValue case for ArrayVal() func.
	dest = NewAttributeValueNull()
	orig = &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: nil}}
	AttributeValue{orig: orig}.CopyTo(dest)
	assert.Nil(t, dest.orig.Value.(*otlpcommon.AnyValue_ArrayValue).ArrayValue)

	// Test copy empty value.
	AttributeValue{orig: &otlpcommon.AnyValue{}}.CopyTo(dest)
	assert.Nil(t, dest.orig.Value)
}

func TestAttributeMap_CopyTo(t *testing.T) {
	dest := NewAttributeMap()
	// Test CopyTo to empty
	NewAttributeMap().CopyTo(dest)
	assert.EqualValues(t, 0, dest.Len())

	// Test CopyTo larger slice
	generateTestAttributeMap().CopyTo(dest)
	assert.EqualValues(t, generateTestAttributeMap(), dest)

	// Test CopyTo same size slice
	generateTestAttributeMap().CopyTo(dest)
	assert.EqualValues(t, generateTestAttributeMap(), dest)

	// Test CopyTo with an empty Value in the destination
	(*dest.orig)[0].Value = otlpcommon.AnyValue{}
	generateTestAttributeMap().CopyTo(dest)
	assert.EqualValues(t, generateTestAttributeMap(), dest)
}

func TestAttributeValue_copyTo(t *testing.T) {
	av := NewAttributeValueNull()
	destVal := otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{}}
	av.copyTo(&destVal)
	assert.EqualValues(t, nil, destVal.Value)
}

func TestAttributeMap_Update(t *testing.T) {
	origWithNil := []otlpcommon.KeyValue{
		{
			Key:   "test_key",
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "test_value"}},
		},
		{
			Key:   "test_key2",
			Value: otlpcommon.AnyValue{Value: nil},
		},
	}
	sm := AttributeMap{
		orig: &origWithNil,
	}

	av, exists := sm.Get("test_key")
	assert.True(t, exists)
	assert.EqualValues(t, AttributeValueTypeString, av.Type())
	assert.EqualValues(t, "test_value", av.StringVal())
	av.SetIntVal(123)

	av2, exists := sm.Get("test_key")
	assert.True(t, exists)
	assert.EqualValues(t, AttributeValueTypeInt, av2.Type())
	assert.EqualValues(t, 123, av2.IntVal())

	av, exists = sm.Get("test_key2")
	assert.True(t, exists)
	assert.EqualValues(t, AttributeValueTypeNull, av.Type())
	assert.EqualValues(t, "", av.StringVal())
	av.SetIntVal(123)

	av2, exists = sm.Get("test_key2")
	assert.True(t, exists)
	assert.EqualValues(t, AttributeValueTypeInt, av2.Type())
	assert.EqualValues(t, 123, av2.IntVal())
}

func TestAttributeMap_EnsureCapacity_Zero(t *testing.T) {
	am := NewAttributeMap()
	am.EnsureCapacity(0)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 0, cap(*am.orig))
}

func TestAttributeMap_EnsureCapacity(t *testing.T) {
	am := NewAttributeMap()
	am.EnsureCapacity(5)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 5, cap(*am.orig))
	am.EnsureCapacity(3)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 5, cap(*am.orig))
	am.EnsureCapacity(8)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 8, cap(*am.orig))
}

func TestAttributeMap_Clear(t *testing.T) {
	am := NewAttributeMap()
	assert.Nil(t, *am.orig)
	am.Clear()
	assert.Nil(t, *am.orig)
	am.EnsureCapacity(5)
	assert.NotNil(t, *am.orig)
	am.Clear()
	assert.Nil(t, *am.orig)
}

func BenchmarkAttributeValue_CopyTo(b *testing.B) {
	av := NewAttributeValueString("k")
	c := NewAttributeValueInt(123)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.copyTo(av.orig)
	}
	if av.IntVal() != 123 {
		b.Fail()
	}
}

func BenchmarkAttributeValue_SetIntVal(b *testing.B) {
	av := NewAttributeValueString("k")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		av.SetIntVal(int64(n))
	}
	if av.IntVal() != int64(b.N-1) {
		b.Fail()
	}
}

func BenchmarkAttributeMap_Range(b *testing.B) {
	const numElements = 20
	rawOrig := make([]otlpcommon.KeyValue, numElements)
	for i := 0; i < numElements; i++ {
		rawOrig[i] = otlpcommon.KeyValue{
			Key:   "k" + strconv.Itoa(i),
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v" + strconv.Itoa(i)}},
		}
	}
	am := AttributeMap{
		orig: &rawOrig,
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		numEls := 0
		am.Range(func(k string, v AttributeValue) bool {
			numEls++
			return true
		})
		if numEls != numElements {
			b.Fail()
		}
	}
}

func BenchmarkAttributeMap_RangeOverMap(b *testing.B) {
	const numElements = 20
	rawOrig := make(map[string]AttributeValue, numElements)
	for i := 0; i < numElements; i++ {
		key := "k" + strconv.Itoa(i)
		rawOrig[key] = NewAttributeValueString("v" + strconv.Itoa(i))
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		numEls := 0
		for _, v := range rawOrig {
			if v.orig == nil {
				continue
			}
			numEls++
		}
		if numEls != numElements {
			b.Fail()
		}
	}
}

func BenchmarkStringMap_RangeOverMap(b *testing.B) {
	const numElements = 20
	rawOrig := make(map[string]string, numElements)
	for i := 0; i < numElements; i++ {
		key := "k" + strconv.Itoa(i)
		rawOrig[key] = "v" + strconv.Itoa(i)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		numEls := 0
		for _, v := range rawOrig {
			if v == "" {
				continue
			}
			numEls++
		}
		if numEls != numElements {
			b.Fail()
		}
	}
}

func fillTestAttributeValue(dest AttributeValue) {
	dest.SetStringVal("v")
}

func generateTestAttributeValue() AttributeValue {
	av := NewAttributeValueNull()
	fillTestAttributeValue(av)
	return av
}

func generateTestAttributeMap() AttributeMap {
	am := NewAttributeMap()
	fillTestAttributeMap(am)
	return am
}

func fillTestAttributeMap(dest AttributeMap) {
	NewAttributeMapFromMap(map[string]AttributeValue{
		"k": NewAttributeValueString("v"),
	}).CopyTo(dest)
}

func generateTestNullAttributeMap() AttributeMap {
	return NewAttributeMapFromMap(map[string]AttributeValue{
		"k": NewAttributeValueNull(),
	})
}
func generateTestIntAttributeMap() AttributeMap {
	return NewAttributeMapFromMap(map[string]AttributeValue{
		"k": NewAttributeValueInt(123),
	})
}

func generateTestDoubleAttributeMap() AttributeMap {
	return NewAttributeMapFromMap(map[string]AttributeValue{
		"k": NewAttributeValueDouble(12.3),
	})
}

func generateTestBoolAttributeMap() AttributeMap {
	return NewAttributeMapFromMap(map[string]AttributeValue{
		"k": NewAttributeValueBool(true),
	})
}

func generateTestBytesAttributeMap() AttributeMap {
	return NewAttributeMapFromMap(map[string]AttributeValue{
		"k": NewAttributeValueBytes([]byte{1, 2, 3, 4, 5}),
	})
}

func TestAttributeValueArray(t *testing.T) {
	a1 := NewAttributeValueArray()
	assert.EqualValues(t, AttributeValueTypeArray, a1.Type())
	assert.EqualValues(t, NewAnyValueArray(), a1.ArrayVal())
	assert.EqualValues(t, 0, a1.ArrayVal().Len())

	v := a1.ArrayVal().AppendEmpty()
	v.SetDoubleVal(123)
	assert.EqualValues(t, 1, a1.ArrayVal().Len())
	assert.EqualValues(t, NewAttributeValueDouble(123), a1.ArrayVal().At(0))
	// Create a second array.
	a2 := NewAttributeValueArray()
	assert.EqualValues(t, 0, a2.ArrayVal().Len())

	a2.ArrayVal().AppendEmpty().SetStringVal("somestr")
	assert.EqualValues(t, 1, a2.ArrayVal().Len())
	assert.EqualValues(t, NewAttributeValueString("somestr"), a2.ArrayVal().At(0))

	// Insert the second array as a child.
	a2.CopyTo(a1.ArrayVal().AppendEmpty())
	assert.EqualValues(t, 2, a1.ArrayVal().Len())
	assert.EqualValues(t, NewAttributeValueDouble(123), a1.ArrayVal().At(0))
	assert.EqualValues(t, a2, a1.ArrayVal().At(1))

	// Check that the array was correctly inserted.
	childArray := a1.ArrayVal().At(1)
	assert.EqualValues(t, AttributeValueTypeArray, childArray.Type())
	assert.EqualValues(t, 1, childArray.ArrayVal().Len())

	v = childArray.ArrayVal().At(0)
	assert.EqualValues(t, AttributeValueTypeString, v.Type())
	assert.EqualValues(t, "somestr", v.StringVal())

	// Test nil values case for ArrayVal() func.
	a1 = AttributeValue{orig: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: nil}}}
	assert.EqualValues(t, NewAnyValueArray(), a1.ArrayVal())
}

func TestAnyValueArrayWithNilValues(t *testing.T) {
	origWithNil := []otlpcommon.AnyValue{
		{},
		{Value: &otlpcommon.AnyValue_StringValue{StringValue: "test_value"}},
	}
	sm := AnyValueArray{
		orig: &origWithNil,
	}

	val := sm.At(0)
	assert.EqualValues(t, AttributeValueTypeNull, val.Type())
	assert.EqualValues(t, "", val.StringVal())

	val = sm.At(1)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "test_value", val.StringVal())

	sm.AppendEmpty().SetStringVal("other_value")
	val = sm.At(2)
	assert.EqualValues(t, AttributeValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())
}

func TestAsString(t *testing.T) {
	tests := []struct {
		name     string
		input    AttributeValue
		expected string
	}{
		{
			name:     "string",
			input:    NewAttributeValueString("string value"),
			expected: "string value",
		},
		{
			name:     "int64",
			input:    NewAttributeValueInt(42),
			expected: "42",
		},
		{
			name:     "float64",
			input:    NewAttributeValueDouble(1.61803399),
			expected: "1.61803399",
		},
		{
			name:     "boolean",
			input:    NewAttributeValueBool(true),
			expected: "true",
		},
		{
			name:     "empty_map",
			input:    NewAttributeValueMap(),
			expected: "{}",
		},
		{
			name:     "simple_map",
			input:    simpleAttributeValueMap(),
			expected: "{\"arrKey\":[\"strOne\",\"strTwo\"],\"boolKey\":false,\"floatKey\":18.6,\"intKey\":7,\"mapKey\":{\"keyOne\":\"valOne\",\"keyTwo\":\"valTwo\"},\"nullKey\":null,\"strKey\":\"strVal\"}",
		},
		{
			name:     "empty_array",
			input:    NewAttributeValueArray(),
			expected: "[]",
		},
		{
			name:     "simple_array",
			input:    simpleAttributeValueArray(),
			expected: "[\"strVal\",7,18.6,false,null]",
		},
		{
			name:     "null",
			input:    NewAttributeValueNull(),
			expected: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.input.AsString()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func simpleAttributeValueMap() AttributeValue {
	ret := NewAttributeValueMap()
	attrMap := ret.MapVal()
	attrMap.UpsertString("strKey", "strVal")
	attrMap.UpsertInt("intKey", 7)
	attrMap.UpsertDouble("floatKey", 18.6)
	attrMap.UpsertBool("boolKey", false)
	attrMap.Upsert("nullKey", NewAttributeValueNull())
	attrMap.Upsert("mapKey", constructTestAttributeSubmap())
	attrMap.Upsert("arrKey", constructTestAttributeSubarray())
	return ret
}

func simpleAttributeValueArray() AttributeValue {
	ret := NewAttributeValueArray()
	attrArr := ret.ArrayVal()
	attrArr.AppendEmpty().SetStringVal("strVal")
	attrArr.AppendEmpty().SetIntVal(7)
	attrArr.AppendEmpty().SetDoubleVal(18.6)
	attrArr.AppendEmpty().SetBoolVal(false)
	attrArr.AppendEmpty()
	return ret
}

func constructTestAttributeSubmap() AttributeValue {
	value := NewAttributeValueMap()
	value.MapVal().UpsertString("keyOne", "valOne")
	value.MapVal().UpsertString("keyTwo", "valTwo")
	return value
}

func constructTestAttributeSubarray() AttributeValue {
	value := NewAttributeValueArray()
	value.ArrayVal().AppendEmpty().SetStringVal("strOne")
	value.ArrayVal().AppendEmpty().SetStringVal("strTwo")
	return value
}
