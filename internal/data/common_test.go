// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package data

import (
	"math/rand"
	"strconv"
	"testing"

	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	"github.com/stretchr/testify/assert"
)

func TestAttributeValue(t *testing.T) {
	v := NewAttributeValueString("abc")
	assert.EqualValues(t, AttributeValueSTRING, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v = NewAttributeValueInt(123)
	assert.EqualValues(t, AttributeValueINT, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v = NewAttributeValueDouble(3.4)
	assert.EqualValues(t, AttributeValueDOUBLE, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v = NewAttributeValueBool(true)
	assert.EqualValues(t, AttributeValueBOOL, v.Type())
	assert.EqualValues(t, true, v.BoolVal())

	v.SetStringVal("abc")
	assert.EqualValues(t, AttributeValueSTRING, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v.SetIntVal(123)
	assert.EqualValues(t, AttributeValueINT, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v.SetDoubleVal(3.4)
	assert.EqualValues(t, AttributeValueDOUBLE, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v.SetBoolVal(true)
	assert.EqualValues(t, AttributeValueBOOL, v.Type())
	assert.EqualValues(t, true, v.BoolVal())
}

func TestNewAttributeValueSlice(t *testing.T) {
	events := NewAttributeValueSlice(0)
	assert.EqualValues(t, 0, len(events))

	n := rand.Intn(10)
	events = NewAttributeValueSlice(n)
	assert.EqualValues(t, n, len(events))
	for event := range events {
		assert.NotNil(t, event)
	}
}

func TestNilAttributeMap(t *testing.T) {
	val, exist := NewAttributeMap().Get("test_key")
	assert.EqualValues(t, false, exist)
	assert.EqualValues(t, AttributeValue{nil}, val)

	insertMap := NewAttributeMap()
	insertMap.Insert("k", NewAttributeValueString("v"))
	assert.EqualValues(t, generateTestAttributeMap(), insertMap)

	insertMapString := NewAttributeMap()
	insertMapString.InsertString("k", "v")
	assert.EqualValues(t, generateTestAttributeMap(), insertMapString)

	insertMapInt := NewAttributeMap()
	insertMapInt.InsertInt("k", 123)
	assert.EqualValues(t, generateTestIntAttributeMap(), insertMapInt)

	insertMapDouble := NewAttributeMap()
	insertMapDouble.InsertDouble("k", 12.3)
	assert.EqualValues(t, generateTestDoubleAttributeMap(), insertMapDouble)

	insertMapBool := NewAttributeMap()
	insertMapBool.InsertBool("k", true)
	assert.EqualValues(t, generateTestBoolAttributeMap(), insertMapBool)

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

	deleteMap := NewAttributeMap()
	assert.EqualValues(t, false, deleteMap.Delete("k"))
	assert.EqualValues(t, NewAttributeMap(), deleteMap)
	assert.EqualValues(t, 0, NewAttributeMap().Len())

	// Test Sort
	assert.EqualValues(t, NewAttributeMap(), NewAttributeMap().Sort())
}

func TestAttributeMapWithNilValues(t *testing.T) {
	origWithNil := []*otlpcommon.AttributeKeyValue{
		nil,
		{
			Key:         "test_key",
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "test_value",
		},
		nil,
	}
	sm := AttributeMap{
		orig: &origWithNil,
	}
	val, exist := sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, "test_value", val.StringVal())

	sm.Insert("other_key", NewAttributeValueString("other_value"))
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.InsertString("other_key_string", "other_value")
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.InsertInt("other_key_int", 123)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, 123, val.IntVal())

	sm.InsertDouble("other_key_double", 1.23)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, 1.23, val.DoubleVal())

	sm.InsertBool("other_key_bool", true)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, true, val.BoolVal())

	sm.Update("other_key", NewAttributeValueString("yet_another_value"))
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpdateString("other_key_string", "yet_another_value")
	val, exist = sm.Get("other_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpdateInt("other_key_int", 456)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, 456, val.IntVal())

	sm.UpdateDouble("other_key_double", 4.56)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, 4.56, val.DoubleVal())

	sm.UpdateBool("other_key_bool", false)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, false, val.BoolVal())

	sm.Upsert("other_key", NewAttributeValueString("other_value"))
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.UpsertString("other_key_string", "other_value")
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.UpsertInt("other_key_int", 123)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, 123, val.IntVal())

	sm.UpsertDouble("other_key_double", 1.23)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, 1.23, val.DoubleVal())

	sm.UpsertBool("other_key_bool", true)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, true, val.BoolVal())

	sm.Upsert("yet_another_key", NewAttributeValueString("yet_another_value"))
	val, exist = sm.Get("yet_another_key")
	assert.True(t, exist)
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpsertString("yet_another_key_string", "yet_another_value")
	val, exist = sm.Get("yet_another_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpsertInt("yet_another_key_int", 456)
	val, exist = sm.Get("yet_another_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, 456, val.IntVal())

	sm.UpsertDouble("yet_another_key_double", 4.56)
	val, exist = sm.Get("yet_another_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, 4.56, val.DoubleVal())

	sm.UpsertBool("yet_another_key_bool", false)
	val, exist = sm.Get("yet_another_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, false, val.BoolVal())

	assert.True(t, sm.Delete("other_key"))
	assert.True(t, sm.Delete("other_key_string"))
	assert.True(t, sm.Delete("other_key_int"))
	assert.True(t, sm.Delete("other_key_double"))
	assert.True(t, sm.Delete("other_key_bool"))
	assert.True(t, sm.Delete("yet_another_key"))
	assert.True(t, sm.Delete("yet_another_key_string"))
	assert.True(t, sm.Delete("yet_another_key_int"))
	assert.True(t, sm.Delete("yet_another_key_double"))
	assert.True(t, sm.Delete("yet_another_key_bool"))
	assert.False(t, sm.Delete("other_key"))
	assert.False(t, sm.Delete("yet_another_key"))

	// Test that the initial key is still there.
	val, exist = sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, "test_value", val.StringVal())

	// Test Sort
	assert.EqualValues(t, AttributeMap{orig: &origWithNil}, sm.Sort())
}

func TestNilStringMap(t *testing.T) {
	val, exist := NewStringMap().Get("test_key")
	assert.EqualValues(t, false, exist)
	assert.EqualValues(t, StringKeyValue{nil}, val)

	insertMap := NewStringMap()
	insertMap.Insert("k", "v")
	assert.EqualValues(t, generateTestStringMap(), insertMap)

	updateMap := NewStringMap()
	updateMap.Update("k", "v")
	assert.EqualValues(t, NewStringMap(), updateMap)

	upsertMap := NewStringMap()
	upsertMap.Upsert("k", "v")
	assert.EqualValues(t, generateTestStringMap(), upsertMap)

	deleteMap := NewStringMap()
	assert.EqualValues(t, false, deleteMap.Delete("k"))
	assert.EqualValues(t, NewStringMap(), deleteMap)
	assert.EqualValues(t, 0, NewStringMap().Len())

	// Test Sort
	assert.EqualValues(t, NewStringMap(), NewStringMap().Sort())
}

func TestStringMapWithNilValues(t *testing.T) {
	origWithNil := []*otlpcommon.StringKeyValue{
		nil,
		{
			Key:   "test_key",
			Value: "test_value",
		},
		nil,
	}
	sm := StringMap{
		orig: &origWithNil,
	}
	val, exist := sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, "test_value", val.Value())

	sm.Insert("other_key", "other_value")
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, "other_value", val.Value())

	sm.Update("other_key", "yet_another_value")
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, "yet_another_value", val.Value())

	sm.Upsert("other_key", "other_value")
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, "other_value", val.Value())

	sm.Upsert("yet_another_key", "yet_another_value")
	val, exist = sm.Get("yet_another_key")
	assert.True(t, exist)
	assert.EqualValues(t, "yet_another_value", val.Value())

	assert.EqualValues(t, true, sm.Delete("other_key"))
	assert.EqualValues(t, true, sm.Delete("yet_another_key"))
	assert.EqualValues(t, false, sm.Delete("other_key"))
	assert.EqualValues(t, false, sm.Delete("yet_another_key"))

	// Test that the initial key is still there.
	val, exist = sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, "test_value", val.Value())

	// Test Sort
	assert.EqualValues(t, StringMap{orig: &origWithNil}, sm.Sort())
}

func TestStringMap(t *testing.T) {
	origRawMap := map[string]string{"k0": "v0", "k1": "v1", "k2": "v2"}
	origMap := NewStringMap().InitFromMap(origRawMap)
	sm := NewStringMap().InitFromMap(origRawMap)
	assert.EqualValues(t, 3, sm.Len())

	val, exist := sm.Get("k2")
	assert.True(t, exist)
	assert.EqualValues(t, NewStringKeyValue("k2", "v2"), val)

	val, exist = sm.Get("k3")
	assert.EqualValues(t, false, exist)
	assert.EqualValues(t, StringKeyValue{nil}, val)

	sm.Insert("k1", "v1")
	assert.EqualValues(t, origMap.Sort(), sm.Sort())
	sm.Insert("k3", "v3")
	assert.EqualValues(t, 4, sm.Len())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"k0": "v0", "k1": "v1", "k2": "v2", "k3": "v3"}).Sort(), sm.Sort())
	assert.EqualValues(t, true, sm.Delete("k3"))
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, origMap.Sort(), sm.Sort())

	sm.Update("k3", "v3")
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, origMap.Sort(), sm.Sort())
	sm.Update("k2", "v3")
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"k0": "v0", "k1": "v1", "k2": "v3"}).Sort(), sm.Sort())
	sm.Update("k2", "v2")
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, origMap.Sort(), sm.Sort())

	sm.Upsert("k3", "v3")
	assert.EqualValues(t, 4, sm.Len())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"k0": "v0", "k1": "v1", "k2": "v2", "k3": "v3"}).Sort(), sm.Sort())
	sm.Upsert("k1", "v5")
	assert.EqualValues(t, 4, sm.Len())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"k0": "v0", "k1": "v5", "k2": "v2", "k3": "v3"}).Sort(), sm.Sort())
	sm.Upsert("k1", "v1")
	assert.EqualValues(t, 4, sm.Len())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"k0": "v0", "k1": "v1", "k2": "v2", "k3": "v3"}).Sort(), sm.Sort())
	assert.EqualValues(t, true, sm.Delete("k3"))
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, origMap.Sort(), sm.Sort())

	assert.EqualValues(t, false, sm.Delete("k3"))
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, origMap.Sort(), sm.Sort())

	assert.EqualValues(t, true, sm.Delete("k0"))
	assert.EqualValues(t, 2, sm.Len())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"k1": "v1", "k2": "v2"}).Sort(), sm.Sort())
	assert.EqualValues(t, true, sm.Delete("k2"))
	assert.EqualValues(t, 1, sm.Len())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"k1": "v1"}).Sort(), sm.Sort())
	assert.EqualValues(t, true, sm.Delete("k1"))
	assert.EqualValues(t, 0, sm.Len())
}

func TestStringMapIteration(t *testing.T) {
	sm := NewStringMap().InitFromMap(map[string]string{"k0": "v0", "k1": "v1", "k2": "v2"})
	assert.EqualValues(t, 3, sm.Len())
	sm.Sort()
	for i := 0; i < sm.Len(); i++ {
		skv := sm.GetStringKeyValue(i)
		assert.EqualValues(t, "k"+strconv.Itoa(i), skv.Key())
		assert.EqualValues(t, "v"+strconv.Itoa(i), skv.Value())
	}
}

func BenchmarkSetValue(b *testing.B) {
	av := NewAttributeValueString("k")
	c := NewAttributeValueInt(123)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		av.SetValue(c)
	}
	if av.IntVal() != 123 {
		b.Fail()
	}
}

func BenchmarkSetIntVal(b *testing.B) {
	av := NewAttributeValueString("k")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		av.SetIntVal(int64(n))
	}
	if av.IntVal() != int64(b.N-1) {
		b.Fail()
	}
}

func generateTestStringMap() StringMap {
	sm := NewStringMap()
	fillTestStringMap(sm)
	return sm
}

func fillTestStringMap(dest StringMap) {
	dest.InitFromMap(map[string]string{
		"k": "v",
	})
}

func generateTestAttributeMap() AttributeMap {
	am := NewAttributeMap()
	fillTestAttributeMap(am)
	return am
}

func fillTestAttributeMap(dest AttributeMap) {
	dest.InitFromMap(map[string]AttributeValue{
		"k": NewAttributeValueString("v"),
	})
}

func generateTestIntAttributeMap() AttributeMap {
	am := NewAttributeMap()
	am.InitFromMap(map[string]AttributeValue{
		"k": NewAttributeValueInt(123),
	})
	return am
}

func generateTestDoubleAttributeMap() AttributeMap {
	am := NewAttributeMap()
	am.InitFromMap(map[string]AttributeValue{
		"k": NewAttributeValueDouble(12.3),
	})
	return am
}

func generateTestBoolAttributeMap() AttributeMap {
	am := NewAttributeMap()
	am.InitFromMap(map[string]AttributeValue{
		"k": NewAttributeValueBool(true),
	})
	return am
}
