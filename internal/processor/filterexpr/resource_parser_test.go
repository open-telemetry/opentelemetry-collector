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
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
)

func TestToResourceExpr(t *testing.T) {
	expr, err := CompileResourceExpr(
		`Attribute("name") != "test" || Attribute("name") == "test" && !HasAttribute("version")`)
	require.NoError(t, err)
	assert.True(t, expr.Evaluate(newResource(map[string]pdata.AttributeValue{
		"name": pdata.NewAttributeValueString("test")})))
	assert.True(t, expr.Evaluate(newResource(map[string]pdata.AttributeValue{
		"name": pdata.NewAttributeValueString("other")})))
	assert.False(t, expr.Evaluate(newResource(map[string]pdata.AttributeValue{
		"name":    pdata.NewAttributeValueString("test"),
		"version": pdata.NewAttributeValueString("version")})))
}

func TestToResourceExpr_AllEqual(t *testing.T) {
	expr, err := CompileResourceExpr(
		`Attribute("string") == "test" && Attribute("int") == 123 && Attribute("double") == 12.3 && Attribute("bool") == true`)
	require.NoError(t, err)
	assert.True(t, expr.Evaluate(newResource(map[string]pdata.AttributeValue{
		"string": pdata.NewAttributeValueString("test"),
		"int":    pdata.NewAttributeValueInt(123),
		"double": pdata.NewAttributeValueDouble(12.3),
		"bool":   pdata.NewAttributeValueBool(true)})))
	assert.False(t, expr.Evaluate(newResource(map[string]pdata.AttributeValue{
		"string": pdata.NewAttributeValueString("test"),
		"int":    pdata.NewAttributeValueInt(234),
		"double": pdata.NewAttributeValueDouble(12.3),
		"bool":   pdata.NewAttributeValueBool(true)})))
	assert.False(t, expr.Evaluate(newResource(map[string]pdata.AttributeValue{
		"string": pdata.NewAttributeValueString("test"),
		"int":    pdata.NewAttributeValueInt(123),
		"double": pdata.NewAttributeValueDouble(23.4),
		"bool":   pdata.NewAttributeValueBool(true)})))
	assert.False(t, expr.Evaluate(newResource(map[string]pdata.AttributeValue{
		"string": pdata.NewAttributeValueString("test"),
		"int":    pdata.NewAttributeValueInt(123),
		"double": pdata.NewAttributeValueDouble(12.3),
		"bool":   pdata.NewAttributeValueBool(false)})))
}

func TestToResourceExpr_AllNotEqual(t *testing.T) {
	expr, err := CompileResourceExpr(
		`Attribute("string") != "test" || Attribute("int") != 123 || Attribute("double") != 12.3 || Attribute("bool") != true`)
	require.NoError(t, err)
	assert.False(t, expr.Evaluate(newResource(map[string]pdata.AttributeValue{
		"string": pdata.NewAttributeValueString("test"),
		"int":    pdata.NewAttributeValueInt(123),
		"double": pdata.NewAttributeValueDouble(12.3),
		"bool":   pdata.NewAttributeValueBool(true)})))
	assert.True(t, expr.Evaluate(newResource(map[string]pdata.AttributeValue{
		"string": pdata.NewAttributeValueString("test"),
		"int":    pdata.NewAttributeValueInt(234),
		"double": pdata.NewAttributeValueDouble(12.3),
		"bool":   pdata.NewAttributeValueBool(true)})))
	assert.True(t, expr.Evaluate(newResource(map[string]pdata.AttributeValue{
		"string": pdata.NewAttributeValueString("test"),
		"int":    pdata.NewAttributeValueInt(123),
		"double": pdata.NewAttributeValueDouble(23.4),
		"bool":   pdata.NewAttributeValueBool(true)})))
	assert.True(t, expr.Evaluate(newResource(map[string]pdata.AttributeValue{
		"string": pdata.NewAttributeValueString("test"),
		"int":    pdata.NewAttributeValueInt(123),
		"double": pdata.NewAttributeValueDouble(12.3),
		"bool":   pdata.NewAttributeValueBool(false)})))
}

func TestToResourceExpr_Empty(t *testing.T) {
	expr, err := CompileResourceExpr(``)
	require.NoError(t, err)
	assert.Nil(t, expr)
}

func TestCompileResourceError(t *testing.T) {
	testcases := []struct {
		name       string
		exprString string
	}{
		{
			name:       "And_FirstRegexp",
			exprString: `Attribute("test") ~= "[a-z)*" && Attribute("test") ~= "[a-z]*"`,
		},
		{
			name:       "And_SecondRegexp",
			exprString: `Attribute("test") ~= "[a-z]*" && Attribute("test") ~= "[a-z)*"`,
		},
		{
			name:       "Or_FirstRegexp",
			exprString: `Attribute("test") ~= "[a-z)*" || Attribute("test") ~= "[a-z]*"`,
		},
		{
			name:       "Or_SecondRegexp",
			exprString: `Attribute("test") ~= "[a-z]*" || Attribute("test") ~= "[a-z)*"`,
		},
		{
			name:       "Not_NameRegexp",
			exprString: `!(Attribute("test") ~= "[a-z)*")`,
		},
		{
			name:       "regexp_int",
			exprString: `Attribute("test") ~= 123`,
		},
		{
			name:       "regexp_double",
			exprString: `Attribute("test") ~= 12.3`,
		},
		{
			name:       "regexp_bool",
			exprString: `Attribute("test") ~= true`,
		},
		{
			name:       "invalid_attribute_call",
			exprString: `Attribute["test") != "test"`,
		},
		{
			name:       "invalid_attribute",
			exprString: `Atttribute("test") != "test"`,
		},
		{
			name:       "invalid_order",
			exprString: `"test" == Attribute("test")`,
		},
		{
			name:       "invalid_not",
			exprString: `!Attribute("test") == "test"`,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompileResourceExpr(tc.exprString)
			assert.Error(t, err)
		})
	}
}

func newResource(attrs map[string]pdata.AttributeValue) pdata.Resource {
	res := pdata.NewResource()
	res.Attributes().InitFromMap(attrs)
	return res
}

func BenchmarkEvaluateResourceExpr(b *testing.B) {
	resource := pdata.NewResource()
	resource.Attributes().InitFromMap(map[string]pdata.AttributeValue{
		conventions.AttributeServiceName: pdata.NewAttributeValueString("svcA"),
		"resString":                      pdata.NewAttributeValueString("arithmetic"),
	})

	res, err := CompileResourceExpr(`Attribute("service.name") == "svcA" && HasAttribute("resString")`)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if !res.Evaluate(resource) {
			b.Fail()
		}
	}
}
