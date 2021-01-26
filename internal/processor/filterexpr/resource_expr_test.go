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
	"go.opentelemetry.io/collector/translator/conventions"
)

func TestResourceMatching(t *testing.T) {
	resource := pdata.NewResource()
	resource.Attributes().InitFromMap(map[string]pdata.AttributeValue{
		"key": pdata.NewAttributeValueString("value"),
	})
	testcases := []struct {
		name     string
		expr     ResourceExpr
		expected bool
	}{
		{
			name:     "has_attribute",
			expr:     newAttributesResourceExpr(newHasAttributeExpr("key")),
			expected: true,
		},
		{
			name:     "no_attribute",
			expr:     newAttributesResourceExpr(newHasAttributeExpr("")),
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(resource))
		})
	}
}

func TestResourceLogicalOperators(t *testing.T) {
	testcases := []struct {
		name     string
		expr     ResourceExpr
		expected bool
	}{
		{
			name:     "or_attribute_true_true",
			expr:     newOrResourceExpr(trueResourceExpr, falseResourceExpr),
			expected: true,
		},
		{
			name:     "or_attribute_true_false",
			expr:     newOrResourceExpr(trueResourceExpr, falseResourceExpr),
			expected: true,
		},
		{
			name:     "or_attribute_false_true",
			expr:     newOrResourceExpr(falseResourceExpr, trueResourceExpr),
			expected: true,
		},
		{
			name:     "or_attribute_false_false",
			expr:     newOrResourceExpr(falseResourceExpr, falseResourceExpr),
			expected: false,
		},
		{
			name:     "and_attribute_true_true",
			expr:     newAndResourceExpr(trueResourceExpr, trueResourceExpr),
			expected: true,
		},
		{
			name:     "and_attribute_true_false",
			expr:     newAndResourceExpr(trueResourceExpr, falseResourceExpr),
			expected: false,
		},
		{
			name:     "and_attribute_false_true",
			expr:     newAndResourceExpr(falseResourceExpr, trueResourceExpr),
			expected: false,
		},
		{
			name:     "and_attribute_false_false",
			expr:     newAndResourceExpr(falseResourceExpr, falseResourceExpr),
			expected: false,
		},
		{
			name:     "not_attribute_true",
			expr:     newNotResourceExpr(trueResourceExpr),
			expected: false,
		},
		{
			name:     "not_attribute_false",
			expr:     newNotResourceExpr(falseResourceExpr),
			expected: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(pdata.NewResource()))
		})
	}
}

type boolResourceExpr bool

func (bae boolResourceExpr) Evaluate(_ pdata.Resource) bool {
	return bool(bae)
}

var trueResourceExpr = boolResourceExpr(true)
var falseResourceExpr = boolResourceExpr(false)

func BenchmarkEvaluateResource(b *testing.B) {
	resource := pdata.NewResource()
	resource.Attributes().InitFromMap(map[string]pdata.AttributeValue{
		conventions.AttributeServiceName: pdata.NewAttributeValueString("svcA"),
		"resString":                      pdata.NewAttributeValueString("arithmetic"),
	})

	res := newAndResourceExpr(
		newAttributesResourceExpr(newStringAttributeExpr(conventions.AttributeServiceName, newEqualStringExpr("svcA"))),
		newAttributesResourceExpr(newHasAttributeExpr("resString")))

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if !res.Evaluate(resource) {
			b.Fail()
		}
	}
}
