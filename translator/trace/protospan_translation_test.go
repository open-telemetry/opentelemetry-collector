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

	"go.opentelemetry.io/collector/model/pdata"
)

func TestAttributeValueToString(t *testing.T) {
	tests := []struct {
		name     string
		input    pdata.AttributeValue
		expected string
	}{
		{
			name:     "string",
			input:    pdata.NewAttributeValueString("string value"),
			expected: "string value",
		},
		{
			name:     "int64",
			input:    pdata.NewAttributeValueInt(42),
			expected: "42",
		},
		{
			name:     "float64",
			input:    pdata.NewAttributeValueDouble(1.61803399),
			expected: "1.61803399",
		},
		{
			name:     "boolean",
			input:    pdata.NewAttributeValueBool(true),
			expected: "true",
		},
		{
			name:     "empty_map",
			input:    pdata.NewAttributeValueMap(),
			expected: "{}",
		},
		{
			name:     "simple_map",
			input:    simpleAttributeValueMap(),
			expected: "{\"arrKey\":[\"strOne\",\"strTwo\"],\"boolKey\":false,\"floatKey\":18.6,\"intKey\":7,\"mapKey\":{\"keyOne\":\"valOne\",\"keyTwo\":\"valTwo\"},\"nullKey\":null,\"strKey\":\"strVal\"}",
		},
		{
			name:     "empty_array",
			input:    pdata.NewAttributeValueArray(),
			expected: "[]",
		},
		{
			name:     "simple_array",
			input:    simpleAttributeValueArray(),
			expected: "[\"strVal\",7,18.6,false,null]",
		},
		{
			name:     "null",
			input:    pdata.NewAttributeValueNull(),
			expected: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := AttributeValueToString(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func simpleAttributeValueMap() pdata.AttributeValue {
	ret := pdata.NewAttributeValueMap()
	attrMap := ret.MapVal()
	attrMap.UpsertString("strKey", "strVal")
	attrMap.UpsertInt("intKey", 7)
	attrMap.UpsertDouble("floatKey", 18.6)
	attrMap.UpsertBool("boolKey", false)
	attrMap.Upsert("nullKey", pdata.NewAttributeValueNull())
	attrMap.Upsert("mapKey", constructTestAttributeSubmap())
	attrMap.Upsert("arrKey", constructTestAttributeSubarray())
	return ret
}

func simpleAttributeValueArray() pdata.AttributeValue {
	ret := pdata.NewAttributeValueArray()
	attrArr := ret.ArrayVal()
	attrArr.AppendEmpty().SetStringVal("strVal")
	attrArr.AppendEmpty().SetIntVal(7)
	attrArr.AppendEmpty().SetDoubleVal(18.6)
	attrArr.AppendEmpty().SetBoolVal(false)
	attrArr.AppendEmpty()
	return ret
}

func constructTestAttributeSubmap() pdata.AttributeValue {
	value := pdata.NewAttributeValueMap()
	value.MapVal().UpsertString("keyOne", "valOne")
	value.MapVal().UpsertString("keyTwo", "valTwo")
	return value
}

func constructTestAttributeSubarray() pdata.AttributeValue {
	value := pdata.NewAttributeValueArray()
	value.ArrayVal().AppendEmpty().SetStringVal("strOne")
	value.ArrayVal().AppendEmpty().SetStringVal("strTwo")
	return value
}
