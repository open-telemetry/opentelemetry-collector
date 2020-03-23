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

	otlp "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	"github.com/stretchr/testify/assert"
)

func TestAttributeValue(t *testing.T) {
	v := NewAttributeValueString("abc")
	assert.EqualValues(t, otlp.AttributeKeyValue_STRING, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v = NewAttributeValueInt(123)
	assert.EqualValues(t, otlp.AttributeKeyValue_INT, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v = NewAttributeValueDouble(3.4)
	assert.EqualValues(t, otlp.AttributeKeyValue_DOUBLE, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v = NewAttributeValueBool(true)
	assert.EqualValues(t, otlp.AttributeKeyValue_BOOL, v.Type())
	assert.EqualValues(t, true, v.BoolVal())

	v.SetString("abc")
	assert.EqualValues(t, otlp.AttributeKeyValue_STRING, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v.SetInt(123)
	assert.EqualValues(t, otlp.AttributeKeyValue_INT, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v.SetDouble(3.4)
	assert.EqualValues(t, otlp.AttributeKeyValue_DOUBLE, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v.SetBool(true)
	assert.EqualValues(t, otlp.AttributeKeyValue_BOOL, v.Type())
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
