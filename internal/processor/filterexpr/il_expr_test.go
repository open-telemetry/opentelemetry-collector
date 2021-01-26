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

func TestIlMatching(t *testing.T) {
	ils := pdata.NewInstrumentationLibrary()
	ils.SetName("name")
	ils.SetVersion("version")
	testcases := []struct {
		name     string
		expr     IlExpr
		expected bool
	}{
		{
			name:     "name",
			expr:     newNameIlExpr(newEqualStringExpr("name")),
			expected: true,
		},
		{
			name:     "empty_name",
			expr:     newNameIlExpr(newEqualStringExpr("")),
			expected: false,
		},
		{
			name:     "version",
			expr:     newVersionIlExpr(newEqualStringExpr("version")),
			expected: true,
		},
		{
			name:     "empty_version",
			expr:     newVersionIlExpr(newEqualStringExpr("")),
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(ils))
		})
	}
}

func TestEmptyIlsMatching(t *testing.T) {
	ils := pdata.NewInstrumentationLibrary()
	testcases := []struct {
		name     string
		expr     IlExpr
		expected bool
	}{
		{
			name:     "name",
			expr:     newNameIlExpr(newEqualStringExpr("name")),
			expected: false,
		},
		{
			name:     "empty_name",
			expr:     newNameIlExpr(newEqualStringExpr("")),
			expected: true,
		},
		{
			name:     "version",
			expr:     newVersionIlExpr(newEqualStringExpr("version")),
			expected: false,
		},
		{
			name:     "empty_version",
			expr:     newVersionIlExpr(newEqualStringExpr("")),
			expected: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(ils))
		})
	}
}

func TestIlLogicalOperators(t *testing.T) {
	testcases := []struct {
		name     string
		expr     IlExpr
		expected bool
	}{
		{
			name:     "or_attribute_true_true",
			expr:     newOrIlExpr(trueIlExpr, falseIlExpr),
			expected: true,
		},
		{
			name:     "or_attribute_true_false",
			expr:     newOrIlExpr(trueIlExpr, falseIlExpr),
			expected: true,
		},
		{
			name:     "or_attribute_false_true",
			expr:     newOrIlExpr(falseIlExpr, trueIlExpr),
			expected: true,
		},
		{
			name:     "or_attribute_false_false",
			expr:     newOrIlExpr(falseIlExpr, falseIlExpr),
			expected: false,
		},
		{
			name:     "and_attribute_true_true",
			expr:     newAndIlExpr(trueIlExpr, trueIlExpr),
			expected: true,
		},
		{
			name:     "and_attribute_true_false",
			expr:     newAndIlExpr(trueIlExpr, falseIlExpr),
			expected: false,
		},
		{
			name:     "and_attribute_false_true",
			expr:     newAndIlExpr(falseIlExpr, trueIlExpr),
			expected: false,
		},
		{
			name:     "and_attribute_false_false",
			expr:     newAndIlExpr(falseIlExpr, falseIlExpr),
			expected: false,
		},
		{
			name:     "not_attribute_true",
			expr:     newNotIlExpr(trueIlExpr),
			expected: false,
		},
		{
			name:     "not_attribute_false",
			expr:     newNotIlExpr(falseIlExpr),
			expected: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(pdata.NewInstrumentationLibrary()))
		})
	}
}

type boolIlExpr bool

func (bae boolIlExpr) Evaluate(_ pdata.InstrumentationLibrary) bool {
	return bool(bae)
}

var trueIlExpr = boolIlExpr(true)
var falseIlExpr = boolIlExpr(false)

func BenchmarkEvaluateIl(b *testing.B) {
	library := pdata.NewInstrumentationLibrary()
	library.SetName("lib")
	library.SetVersion("ver")

	ils := newAndIlExpr(
		newNameIlExpr(newEqualStringExpr("lib")),
		newVersionIlExpr(newEqualStringExpr("ver")))

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if !ils.Evaluate(library) {
			b.Fail()
		}
	}
}
