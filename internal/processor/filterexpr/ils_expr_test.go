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

func TestIlsMatching(t *testing.T) {
	ils := pdata.NewInstrumentationLibrary()
	ils.SetName("name")
	ils.SetVersion("version")
	testcases := []struct {
		name     string
		expr     IlsExpr
		expected bool
	}{
		{
			name:     "name",
			expr:     NewNameIlsExpr(NewStrictStringExpr("name")),
			expected: true,
		},
		{
			name:     "empty_name",
			expr:     NewNameIlsExpr(NewStrictStringExpr("")),
			expected: false,
		},
		{
			name:     "version",
			expr:     NewVersionIlsExpr(NewStrictStringExpr("version")),
			expected: true,
		},
		{
			name:     "empty_version",
			expr:     NewVersionIlsExpr(NewStrictStringExpr("")),
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(ils))
		})
	}
}

func TestEmptyIlssMatching(t *testing.T) {
	ils := pdata.NewInstrumentationLibrary()
	testcases := []struct {
		name     string
		expr     IlsExpr
		expected bool
	}{
		{
			name:     "name",
			expr:     NewNameIlsExpr(NewStrictStringExpr("name")),
			expected: false,
		},
		{
			name:     "empty_name",
			expr:     NewNameIlsExpr(NewStrictStringExpr("")),
			expected: true,
		},
		{
			name:     "version",
			expr:     NewVersionIlsExpr(NewStrictStringExpr("version")),
			expected: false,
		},
		{
			name:     "empty_version",
			expr:     NewVersionIlsExpr(NewStrictStringExpr("")),
			expected: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(ils))
		})
	}
}

func TestIlsLogicalOperators(t *testing.T) {
	testcases := []struct {
		name     string
		expr     IlsExpr
		expected bool
	}{
		{
			name:     "or_attribute_true_true",
			expr:     NewOrIlsExpr(trueIlsExpr, falseIlsExpr),
			expected: true,
		},
		{
			name:     "or_attribute_true_false",
			expr:     NewOrIlsExpr(trueIlsExpr, falseIlsExpr),
			expected: true,
		},
		{
			name:     "or_attribute_false_true",
			expr:     NewOrIlsExpr(falseIlsExpr, trueIlsExpr),
			expected: true,
		},
		{
			name:     "or_attribute_false_false",
			expr:     NewOrIlsExpr(falseIlsExpr, falseIlsExpr),
			expected: false,
		},
		{
			name:     "and_attribute_true_true",
			expr:     NewAndIlsExpr(trueIlsExpr, trueIlsExpr),
			expected: true,
		},
		{
			name:     "and_attribute_true_false",
			expr:     NewAndIlsExpr(trueIlsExpr, falseIlsExpr),
			expected: false,
		},
		{
			name:     "and_attribute_false_true",
			expr:     NewAndIlsExpr(falseIlsExpr, trueIlsExpr),
			expected: false,
		},
		{
			name:     "and_attribute_false_false",
			expr:     NewAndIlsExpr(falseIlsExpr, falseIlsExpr),
			expected: false,
		},
		{
			name:     "not_attribute_true",
			expr:     NewNotIlsExpr(trueIlsExpr),
			expected: false,
		},
		{
			name:     "not_attribute_false",
			expr:     NewNotIlsExpr(falseIlsExpr),
			expected: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(pdata.NewInstrumentationLibrary()))
		})
	}
}

type boolIlsExpr bool

func (bae boolIlsExpr) Evaluate(_ pdata.InstrumentationLibrary) bool {
	return bool(bae)
}

var trueIlsExpr = boolIlsExpr(true)
var falseIlsExpr = boolIlsExpr(false)

func BenchmarkEvaluateIls(b *testing.B) {
	library := pdata.NewInstrumentationLibrary()
	library.SetName("lib")
	library.SetVersion("ver")

	ils := NewAndIlsExpr(
		NewNameIlsExpr(NewStrictStringExpr("lib")),
		NewVersionIlsExpr(NewStrictStringExpr("ver")))

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if !ils.Evaluate(library) {
			b.Fail()
		}
	}
}
