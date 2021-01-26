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
)

func TestIntMatching(t *testing.T) {
	ils := pdata.NewInstrumentationLibrary()
	ils.SetName("name")
	ils.SetVersion("version")
	testcases := []struct {
		name     string
		value    int64
		expr     IntExpr
		expected bool
	}{
		{
			name:     "not_equal",
			value:    123,
			expr:     newNotEqualIntExpr(234),
			expected: true,
		},
		{
			name:     "not_equal/same_value",
			value:    123,
			expr:     newNotEqualIntExpr(123),
			expected: false,
		},
		{
			name:     "equal",
			value:    123,
			expr:     newEqualIntExpr(123),
			expected: true,
		},
		{
			name:     "equal/other_value",
			value:    123,
			expr:     newEqualIntExpr(234),
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(tc.value))
		})
	}
}

func TestDoubleMatching(t *testing.T) {
	ils := pdata.NewInstrumentationLibrary()
	ils.SetName("name")
	ils.SetVersion("version")
	testcases := []struct {
		name     string
		value    float64
		expr     DoubleExpr
		expected bool
	}{
		{
			name:     "not_equal",
			value:    12.3,
			expr:     newNotEqualDoubleExpr(23.4),
			expected: true,
		},
		{
			name:     "not_equal/same_value",
			value:    12.3,
			expr:     newNotEqualDoubleExpr(12.3),
			expected: false,
		},
		{
			name:     "equal",
			value:    12.3,
			expr:     newEqualDoubleExpr(12.3),
			expected: true,
		},
		{
			name:     "equal/other_value",
			value:    12.3,
			expr:     newEqualDoubleExpr(23.4),
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(tc.value))
		})
	}
}

func TestStringMatching(t *testing.T) {
	ils := pdata.NewInstrumentationLibrary()
	ils.SetName("name")
	ils.SetVersion("version")
	testcases := []struct {
		name     string
		value    string
		expr     StringExpr
		expected bool
	}{
		{
			name:     "not_equal",
			value:    "value",
			expr:     newNotEqualStringExpr("other"),
			expected: true,
		},
		{
			name:     "not_equal/empty_value",
			value:    "",
			expr:     newNotEqualStringExpr("value"),
			expected: true,
		},
		{
			name:     "not_equal/same_value",
			value:    "value",
			expr:     newNotEqualStringExpr("value"),
			expected: false,
		},
		{
			name:     "equal",
			value:    "value",
			expr:     newEqualStringExpr("value"),
			expected: true,
		},
		{
			name:     "equal/empty_value",
			value:    "",
			expr:     newEqualStringExpr("value"),
			expected: false,
		},
		{
			name:     "equal/other_value",
			value:    "other_value",
			expr:     newEqualStringExpr("value"),
			expected: false,
		},
		{
			name:     "regexp",
			value:    "value",
			expr:     newMustRegexpStringExpr(t, "^val.*"),
			expected: true,
		},
		{
			name:     "regexp/empty_value",
			value:    "",
			expr:     newMustRegexpStringExpr(t, "^val.*"),
			expected: false,
		},
		{
			name:     "regexp/other_value",
			value:    "other_value",
			expr:     newMustRegexpStringExpr(t, "^val.*"),
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(tc.value))
		})
	}
}

func newMustRegexpStringExpr(t *testing.T, str string) StringExpr {
	se, err := newRegexpStringExpr(str)
	require.NoError(t, err)
	return se
}
