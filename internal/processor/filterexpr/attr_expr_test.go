// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterexpr

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestAttributesMatching(t *testing.T) {
	attr := pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
		"keyString": pdata.NewAttributeValueString("value"),
		"keyInt":    pdata.NewAttributeValueInt(123),
		"keyDouble": pdata.NewAttributeValueDouble(12.3),
		"keyBool":   pdata.NewAttributeValueBool(true),
		"keyExists": pdata.NewAttributeValueString("present"),
	})
	testcases := []struct {
		name     string
		expr     AttributeExpr
		expected bool
	}{
		{
			name:     "has_attribute_true",
			expr:     newHasAttributeExpr("keyExists"),
			expected: true,
		},
		{
			name:     "has_attribute_false",
			expr:     newHasAttributeExpr("keyExists2"),
			expected: false,
		},
		{
			name:     "int_attribute_true",
			expr:     newIntAttributeExpr("keyInt", newEqualIntExpr(123)),
			expected: true,
		},
		{
			name:     "int_attribute_no_key",
			expr:     newIntAttributeExpr("keyInt2", newEqualIntExpr(123)),
			expected: false,
		},
		{
			name:     "int_attribute_different_value",
			expr:     newIntAttributeExpr("keyInt", newEqualIntExpr(234)),
			expected: false,
		},
		{
			name:     "double_attribute_true",
			expr:     newDoubleAttributeExpr("keyDouble", newEqualDoubleExpr(12.3)),
			expected: true,
		},
		{
			name:     "double_attribute_no_key",
			expr:     newDoubleAttributeExpr("keyDouble2", newEqualDoubleExpr(12.3)),
			expected: false,
		},
		{
			name:     "double_attribute_different_value",
			expr:     newDoubleAttributeExpr("keyDouble", newEqualDoubleExpr(23.4)),
			expected: false,
		},
		{
			name:     "string_attribute_true",
			expr:     newStringAttributeExpr("keyString", newEqualStringExpr("value")),
			expected: true,
		},
		{
			name:     "string_attribute_no_key",
			expr:     newStringAttributeExpr("keyString2", newEqualStringExpr("value")),
			expected: false,
		},
		{
			name:     "string_attribute_different_value",
			expr:     newStringAttributeExpr("keyString", newEqualStringExpr("other_value")),
			expected: false,
		},
		{
			name:     "bool_attribute_true",
			expr:     newBoolAttributeExpr("keyBool", true),
			expected: true,
		},
		{
			name:     "bool_attribute_no_key",
			expr:     newBoolAttributeExpr("keyBool2", true),
			expected: false,
		},
		{
			name:     "bool_attribute_different_value",
			expr:     newBoolAttributeExpr("keyBool", false),
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(attr))
		})
	}
}

func TestEmptyAttributesMatching(t *testing.T) {
	attr := pdata.NewAttributeMap()
	testcases := []struct {
		name     string
		expr     AttributeExpr
		expected bool
	}{
		{
			name:     "has_attribute",
			expr:     newHasAttributeExpr("key"),
			expected: false,
		},
		{
			name:     "int_attribute",
			expr:     newIntAttributeExpr("key", newEqualIntExpr(123)),
			expected: false,
		},
		{
			name:     "double_attribute",
			expr:     newDoubleAttributeExpr("key", newEqualDoubleExpr(12.3)),
			expected: false,
		},
		{
			name:     "string_attribute",
			expr:     newStringAttributeExpr("key", newEqualStringExpr("value")),
			expected: false,
		},
		{
			name:     "bool_attribute",
			expr:     newBoolAttributeExpr("key", true),
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(attr))
		})
	}
}
