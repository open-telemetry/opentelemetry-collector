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

package configschema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadFieldsWithDefaults(t *testing.T) {
	defaults := map[string]interface{}{
		"one":           "1",
		"two":           int64(2),
		"three":         uint64(3),
		"four":          true,
		"duration":      "42ns",
		"name":          "squashed",
		"person_ptr":    "foo",
		"person_struct": "bar",
	}
	s := testStruct{
		One:      "1",
		Two:      2,
		Three:    3,
		Four:     true,
		Duration: 42,
		Squashed: testPerson{"squashed"},
		PersonPtr: &testPerson{
			Name: "foo",
		},
		PersonStruct: testPerson{
			Name: "bar",
		},
	}
	testReadFields(t, s, defaults)
}

func TestReadFieldsWithoutDefaults(t *testing.T) {
	testReadFields(t, testStruct{}, map[string]interface{}{})
}

func getField(fields []*Field, name string) *Field {
	for _, f := range fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func testReadFields(t *testing.T, s testStruct, defaults map[string]interface{}) {
	root := ReadFields(
		reflect.ValueOf(s),
		testDR(),
	)

	assert.Equal(t, "testStruct comment\n", root.Doc)

	assert.Equal(t, "configschema.testStruct", root.Type)

	assert.Equal(t, 10, len(root.Fields))

	assert.Equal(t, &Field{
		Name:    "one",
		Kind:    "string",
		Default: defaults["one"],
	}, getField(root.Fields, "one"))

	assert.Equal(t, &Field{
		Name:    "two",
		Kind:    "int",
		Default: defaults["two"],
	}, getField(root.Fields, "two"))

	assert.Equal(t, &Field{
		Name:    "three",
		Kind:    "uint",
		Default: defaults["three"],
	}, getField(root.Fields, "three"))

	assert.Equal(t, &Field{
		Name:    "four",
		Kind:    "bool",
		Default: defaults["four"],
	}, getField(root.Fields, "four"))

	assert.Equal(t, &Field{
		Name:    "duration",
		Type:    "time.Duration",
		Kind:    "int64",
		Default: defaults["duration"],
		Doc:     "embedded, package qualified comment\n",
	}, getField(root.Fields, "duration"))

	assert.Equal(t, &Field{
		Name:    "name",
		Kind:    "string",
		Default: defaults["name"],
	}, getField(root.Fields, "name"))

	personPtr := getField(root.Fields, "person_ptr")
	assert.Equal(t, "*configschema.testPerson", personPtr.Type)
	assert.Equal(t, "ptr", personPtr.Kind)
	assert.Equal(t, 1, len(personPtr.Fields))
	assert.Equal(t, &Field{
		Name:    "name",
		Kind:    "string",
		Default: defaults["person_ptr"],
	}, getField(personPtr.Fields, "name"))

	personStruct := getField(root.Fields, "person_struct")
	assert.Equal(t, "configschema.testPerson", personStruct.Type)
	assert.Equal(t, "struct", personStruct.Kind)
	assert.Equal(t, 1, len(personStruct.Fields))
	assert.Equal(t, &Field{
		Name:    "name",
		Kind:    "string",
		Default: defaults["person_struct"],
	}, getField(personStruct.Fields, "name"))

	persons := getField(root.Fields, "persons")
	assert.Equal(t, "[]configschema.testPerson", persons.Type)
	assert.Equal(t, "slice", persons.Kind)
	assert.Equal(t, 1, len(persons.Fields))
	assert.Equal(t, &Field{
		Name: "name",
		Kind: "string",
	}, getField(persons.Fields, "name"))

	personPtrs := getField(root.Fields, "person_ptrs")
	assert.Equal(t, "[]*configschema.testPerson", personPtrs.Type)
	assert.Equal(t, "slice", personPtrs.Kind)
	assert.Equal(t, 1, len(personPtrs.Fields))
	assert.Equal(t, &Field{
		Name: "name",
		Kind: "string",
	}, getField(personPtrs.Fields, "name"))
}
