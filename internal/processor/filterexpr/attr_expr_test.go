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
			expr:     NewHasAttributeExpr("keyExists"),
			expected: true,
		},
		{
			name:     "has_attribute_false",
			expr:     NewHasAttributeExpr("keyExists2"),
			expected: false,
		},
		{
			name:     "int_attribute_true",
			expr:     NewIntAttributeExpr("keyInt", 123),
			expected: true,
		},
		{
			name:     "int_attribute_no_key",
			expr:     NewIntAttributeExpr("keyInt2", 123),
			expected: false,
		},
		{
			name:     "int_attribute_different_value",
			expr:     NewIntAttributeExpr("keyInt", 234),
			expected: false,
		},
		{
			name:     "double_attribute_true",
			expr:     NewDoubleAttributeExpr("keyDouble", 12.3),
			expected: true,
		},
		{
			name:     "double_attribute_no_key",
			expr:     NewDoubleAttributeExpr("keyDouble2", 12.3),
			expected: false,
		},
		{
			name:     "double_attribute_different_value",
			expr:     NewDoubleAttributeExpr("keyDouble", 23.4),
			expected: false,
		},
		{
			name:     "string_attribute_true",
			expr:     NewStringAttributeExpr("keyString", NewStrictStringExpr("value")),
			expected: true,
		},
		{
			name:     "string_attribute_no_key",
			expr:     NewStringAttributeExpr("keyString2", NewStrictStringExpr("value")),
			expected: false,
		},
		{
			name:     "string_attribute_different_value",
			expr:     NewStringAttributeExpr("keyString", NewStrictStringExpr("other_value")),
			expected: false,
		},
		{
			name:     "bool_attribute_true",
			expr:     NewBoolAttributeExpr("keyBool", true),
			expected: true,
		},
		{
			name:     "bool_attribute_no_key",
			expr:     NewBoolAttributeExpr("keyBool2", true),
			expected: false,
		},
		{
			name:     "bool_attribute_different_value",
			expr:     NewBoolAttributeExpr("keyBool", false),
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
			expr:     NewHasAttributeExpr("key"),
			expected: false,
		},
		{
			name:     "int_attribute",
			expr:     NewIntAttributeExpr("key", 123),
			expected: false,
		},
		{
			name:     "double_attribute",
			expr:     NewDoubleAttributeExpr("key", 12.3),
			expected: false,
		},
		{
			name:     "string_attribute",
			expr:     NewStringAttributeExpr("key", NewStrictStringExpr("value")),
			expected: false,
		},
		{
			name:     "bool_attribute",
			expr:     NewBoolAttributeExpr("key", true),
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(attr))
		})
	}
}

func TestAttributesLogicalOperators(t *testing.T) {
	testcases := []struct {
		name     string
		expr     AttributeExpr
		expected bool
	}{
		{
			name:     "or_attribute_true_true",
			expr:     NewOrAttributeExpr(trueAttributeExpr, falseAttributeExpr),
			expected: true,
		},
		{
			name:     "or_attribute_true_false",
			expr:     NewOrAttributeExpr(trueAttributeExpr, falseAttributeExpr),
			expected: true,
		},
		{
			name:     "or_attribute_false_true",
			expr:     NewOrAttributeExpr(falseAttributeExpr, trueAttributeExpr),
			expected: true,
		},
		{
			name:     "or_attribute_false_false",
			expr:     NewOrAttributeExpr(falseAttributeExpr, falseAttributeExpr),
			expected: false,
		},
		{
			name:     "and_attribute_true_true",
			expr:     NewAndAttributeExpr(trueAttributeExpr, trueAttributeExpr),
			expected: true,
		},
		{
			name:     "and_attribute_true_false",
			expr:     NewAndAttributeExpr(trueAttributeExpr, falseAttributeExpr),
			expected: false,
		},
		{
			name:     "and_attribute_false_true",
			expr:     NewAndAttributeExpr(falseAttributeExpr, trueAttributeExpr),
			expected: false,
		},
		{
			name:     "and_attribute_false_false",
			expr:     NewAndAttributeExpr(falseAttributeExpr, falseAttributeExpr),
			expected: false,
		},
		{
			name:     "not_attribute_true",
			expr:     NewNotAttributeExpr(trueAttributeExpr),
			expected: false,
		},
		{
			name:     "not_attribute_false",
			expr:     NewNotAttributeExpr(falseAttributeExpr),
			expected: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(pdata.NewAttributeMap()))
		})
	}
}

type boolAttributeExpr bool

func (bae boolAttributeExpr) Evaluate(_ pdata.AttributeMap) bool {
	return bool(bae)
}

var trueAttributeExpr = boolAttributeExpr(true)
var falseAttributeExpr = boolAttributeExpr(false)
