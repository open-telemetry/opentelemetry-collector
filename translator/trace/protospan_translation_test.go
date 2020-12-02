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

package tracetranslator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestAttributeValueToString(t *testing.T) {
	tests := []struct {
		name     string
		input    pdata.AttributeValue
		jsonLike bool
		expected string
	}{
		{
			name:     "string",
			input:    pdata.NewAttributeValueString("string value"),
			jsonLike: false,
			expected: "string value",
		},
		{
			name:     "json string",
			input:    pdata.NewAttributeValueString("string value"),
			jsonLike: true,
			expected: "\"string value\"",
		},
		{
			name:     "int64",
			input:    pdata.NewAttributeValueInt(42),
			jsonLike: false,
			expected: "42",
		},
		{
			name:     "float64",
			input:    pdata.NewAttributeValueDouble(1.61803399),
			jsonLike: false,
			expected: "1.61803399",
		},
		{
			name:     "boolean",
			input:    pdata.NewAttributeValueBool(true),
			jsonLike: false,
			expected: "true",
		},
		{
			name:     "null",
			input:    pdata.NewAttributeValueNull(),
			jsonLike: false,
			expected: "",
		},
		{
			name:     "null",
			input:    pdata.NewAttributeValueNull(),
			jsonLike: true,
			expected: "null",
		},
		{
			name:     "map",
			input:    pdata.NewAttributeValueMap(),
			jsonLike: false,
			expected: "{}",
		},
		{
			name:     "array",
			input:    pdata.NewAttributeValueArray(),
			jsonLike: false,
			expected: "[]",
		},
		{
			name:     "array",
			input:    pdata.NewAttributeValueNull(),
			jsonLike: false,
			expected: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := AttributeValueToString(test.input, test.jsonLike)
			assert.Equal(t, test.expected, actual)
			dest := pdata.NewAttributeMap()
			key := "keyOne"
			UpsertStringToAttributeMap(key, actual, dest, false)
			val, ok := dest.Get(key)
			assert.True(t, ok)
			if !test.jsonLike {
				switch test.input.Type() {
				case pdata.AttributeValueINT, pdata.AttributeValueDOUBLE, pdata.AttributeValueBOOL:
					assert.EqualValues(t, test.input, val)
				case pdata.AttributeValueARRAY:
					assert.NotNil(t, val)
				default:
					assert.Equal(t, test.expected, val.StringVal())
				}
			}
		})
	}
}

func TestAttributeMapToStringAndBack(t *testing.T) {
	expected := pdata.NewAttributeValueMap()
	attrMap := expected.MapVal()
	attrMap.UpsertString("strKey", "strVal")
	attrMap.UpsertInt("intKey", 7)
	attrMap.UpsertDouble("floatKey", 18.6)
	attrMap.UpsertBool("boolKey", false)
	attrMap.Upsert("nullKey", pdata.NewAttributeValueNull())
	attrMap.Upsert("mapKey", constructTestAttributeSubmap())
	attrMap.Upsert("arrKey", constructTestAttributeSubarray())
	strVal := AttributeValueToString(expected, false)
	dest := pdata.NewAttributeMap()
	UpsertStringToAttributeMap("parent", strVal, dest, false)
	actual, ok := dest.Get("parent")
	assert.True(t, ok)
	compareMaps(t, attrMap, actual.MapVal())
}

func TestAttributeArrayToStringAndBack(t *testing.T) {
	expected := pdata.NewAttributeValueArray()
	attrArr := expected.ArrayVal()
	attrArr.Append(pdata.NewAttributeValueString("strVal"))
	attrArr.Append(pdata.NewAttributeValueInt(7))
	attrArr.Append(pdata.NewAttributeValueDouble(18.6))
	attrArr.Append(pdata.NewAttributeValueBool(false))
	attrArr.Append(pdata.NewAttributeValueNull())
	strVal := AttributeValueToString(expected, false)
	dest := pdata.NewAttributeMap()
	UpsertStringToAttributeMap("parent", strVal, dest, false)
	actual, ok := dest.Get("parent")
	assert.True(t, ok)
	compareArrays(t, attrArr, actual.ArrayVal())
}

func compareMaps(t *testing.T, expected pdata.AttributeMap, actual pdata.AttributeMap) {
	expected.ForEach(func(k string, e pdata.AttributeValue) {
		a, ok := actual.Get(k)
		assert.True(t, ok)
		if ok {
			if e.Type() == pdata.AttributeValueMAP {
				compareMaps(t, e.MapVal(), a.MapVal())
			} else {
				assert.Equal(t, e, a)
			}
		}
	})
}

func compareArrays(t *testing.T, expected pdata.AnyValueArray, actual pdata.AnyValueArray) {
	for i := 0; i < expected.Len(); i++ {
		e := expected.At(i)
		a := actual.At(i)
		if e.Type() == pdata.AttributeValueMAP {
			compareMaps(t, e.MapVal(), a.MapVal())
		} else {
			assert.Equal(t, e, a)
		}
	}
}

func constructTestAttributeSubmap() pdata.AttributeValue {
	value := pdata.NewAttributeValueMap()
	value.MapVal().UpsertString("keyOne", "valOne")
	value.MapVal().UpsertString("keyTwo", "valTwo")
	return value
}

func constructTestAttributeSubarray() pdata.AttributeValue {
	value := pdata.NewAttributeValueArray()
	a1 := pdata.NewAttributeValueString("strOne")
	value.ArrayVal().Append(a1)
	a2 := pdata.NewAttributeValueString("strTwo")
	value.ArrayVal().Append(a2)
	return value
}
