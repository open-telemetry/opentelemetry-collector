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

	v.SetString("abc")
	assert.EqualValues(t, AttributeValueSTRING, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v.SetInt(123)
	assert.EqualValues(t, AttributeValueINT, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v.SetDouble(3.4)
	assert.EqualValues(t, AttributeValueDOUBLE, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v.SetBool(true)
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

func TestAttributeKeyValue(t *testing.T) {
	v := NewAttributeKeyValueString("key_string", "abc")
	assert.EqualValues(t, "key_string", v.Key())
	assert.EqualValues(t, AttributeValueSTRING, v.ValType())
	assert.EqualValues(t, "abc", v.StringVal())

	v = NewAttributeKeyValueInt("int_string", 123)
	assert.EqualValues(t, "int_string", v.Key())
	assert.EqualValues(t, AttributeValueINT, v.ValType())
	assert.EqualValues(t, 123, v.IntVal())

	v = NewAttributeKeyValueDouble("double_string", 3.4)
	assert.EqualValues(t, "double_string", v.Key())
	assert.EqualValues(t, AttributeValueDOUBLE, v.ValType())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v = NewAttributeKeyValueBool("bool_string", true)
	assert.EqualValues(t, "bool_string", v.Key())
	assert.EqualValues(t, AttributeValueBOOL, v.ValType())
	assert.EqualValues(t, true, v.BoolVal())

	v = NewAttributeKeyValueBool("other_key", true)
	v.SetStringVal("abc")
	assert.EqualValues(t, "other_key", v.Key())
	assert.EqualValues(t, AttributeValueSTRING, v.ValType())
	assert.EqualValues(t, "abc", v.StringVal())

	v.SetIntVal(123)
	assert.EqualValues(t, "other_key", v.Key())
	assert.EqualValues(t, AttributeValueINT, v.ValType())
	assert.EqualValues(t, 123, v.IntVal())

	v.SetDoubleVal(3.4)
	assert.EqualValues(t, "other_key", v.Key())
	assert.EqualValues(t, AttributeValueDOUBLE, v.ValType())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v.SetBoolVal(true)
	assert.EqualValues(t, "other_key", v.Key())
	assert.EqualValues(t, AttributeValueBOOL, v.ValType())
	assert.EqualValues(t, true, v.BoolVal())
}

func TestNilStringMap(t *testing.T) {
	val, exist := NewStringMap(nil).Get("test_key")
	assert.EqualValues(t, false, exist)
	assert.EqualValues(t, StringKeyValue{nil}, val)
	insertMap := NewStringMap(nil)
	insertMap.Insert("key", "value")
	assert.EqualValues(t, NewStringMap(map[string]string{"key": "value"}), insertMap)
	updateMap := NewStringMap(nil)
	updateMap.Update("key", "value")
	assert.EqualValues(t, NewStringMap(nil), updateMap)
	upsertMap := NewStringMap(nil)
	upsertMap.Upsert("key", "value")
	assert.EqualValues(t, NewStringMap(map[string]string{"key": "value"}), upsertMap)
	deleteMap := NewStringMap(nil)
	assert.EqualValues(t, false, deleteMap.Delete("key"))
	assert.EqualValues(t, NewStringMap(nil), deleteMap)
	assert.EqualValues(t, 0, NewStringMap(nil).Len())
	assert.EqualValues(t, NewStringMap(nil), NewStringMap(nil).Sort())
}

func TestStringMap(t *testing.T) {
	origMap := map[string]string{"k0": "v0", "k1": "v1", "k2": "v2"}
	sm := NewStringMap(origMap)
	assert.EqualValues(t, 3, sm.Len())

	val, exist := sm.Get("k2")
	assert.EqualValues(t, true, exist)
	assert.EqualValues(t, NewStringKeyValue("k2", "v2"), val)

	val, exist = sm.Get("k3")
	assert.EqualValues(t, false, exist)
	assert.EqualValues(t, StringKeyValue{nil}, val)

	sm.Insert("k1", "v1")
	assert.EqualValues(t, NewStringMap(origMap).Sort(), sm.Sort())
	sm.Insert("k3", "v3")
	assert.EqualValues(t, 4, sm.Len())
	assert.EqualValues(t, NewStringMap(map[string]string{"k0": "v0", "k1": "v1", "k2": "v2", "k3": "v3"}).Sort(), sm.Sort())
	assert.EqualValues(t, true, sm.Delete("k3"))
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, NewStringMap(origMap).Sort(), sm.Sort())

	sm.Update("k3", "v3")
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, NewStringMap(origMap).Sort(), sm.Sort())
	sm.Update("k2", "v3")
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, NewStringMap(map[string]string{"k0": "v0", "k1": "v1", "k2": "v3"}).Sort(), sm.Sort())
	sm.Update("k2", "v2")
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, NewStringMap(origMap).Sort(), sm.Sort())

	sm.Upsert("k3", "v3")
	assert.EqualValues(t, 4, sm.Len())
	assert.EqualValues(t, NewStringMap(map[string]string{"k0": "v0", "k1": "v1", "k2": "v2", "k3": "v3"}).Sort(), sm.Sort())
	sm.Upsert("k1", "v5")
	assert.EqualValues(t, 4, sm.Len())
	assert.EqualValues(t, NewStringMap(map[string]string{"k0": "v0", "k1": "v5", "k2": "v2", "k3": "v3"}).Sort(), sm.Sort())
	sm.Upsert("k1", "v1")
	assert.EqualValues(t, 4, sm.Len())
	assert.EqualValues(t, NewStringMap(map[string]string{"k0": "v0", "k1": "v1", "k2": "v2", "k3": "v3"}).Sort(), sm.Sort())
	assert.EqualValues(t, true, sm.Delete("k3"))
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, NewStringMap(origMap).Sort(), sm.Sort())

	assert.EqualValues(t, false, sm.Delete("k3"))
	assert.EqualValues(t, 3, sm.Len())
	assert.EqualValues(t, NewStringMap(origMap).Sort(), sm.Sort())

	assert.EqualValues(t, true, sm.Delete("k0"))
	assert.EqualValues(t, 2, sm.Len())
	assert.EqualValues(t, NewStringMap(map[string]string{"k1": "v1", "k2": "v2"}).Sort(), sm.Sort())
	assert.EqualValues(t, true, sm.Delete("k2"))
	assert.EqualValues(t, 1, sm.Len())
	assert.EqualValues(t, NewStringMap(map[string]string{"k1": "v1"}).Sort(), sm.Sort())
	assert.EqualValues(t, true, sm.Delete("k1"))
	assert.EqualValues(t, 0, sm.Len())
}

func TestStringMapIteration(t *testing.T) {
	sm := NewStringMap(map[string]string{"k0": "v0", "k1": "v1", "k2": "v2"})
	assert.EqualValues(t, 3, sm.Len())
	sm.Sort()
	for i := 0; i < sm.Len(); i++ {
		skv := sm.GetStringKeyValue(i)
		assert.EqualValues(t, "k"+strconv.Itoa(i), skv.Key())
		assert.EqualValues(t, "v"+strconv.Itoa(i), skv.Value())
	}
}

func generateTestStringMap() StringMap {
	return NewStringMap(map[string]string{
		"k": "v",
	})
}

func generateTestAttributeMap() AttributeMap {
	return NewAttributeMap(map[string]AttributeValue{
		"k": NewAttributeValueString("v"),
	})
}
