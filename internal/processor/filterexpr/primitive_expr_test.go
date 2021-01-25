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
			name:     "strict",
			value:    "value",
			expr:     NewStrictStringExpr("value"),
			expected: true,
		},
		{
			name:     "strict/empty_value",
			value:    "",
			expr:     NewStrictStringExpr("value"),
			expected: false,
		},
		{
			name:     "strict/other_value",
			value:    "other_value",
			expr:     NewStrictStringExpr("value"),
			expected: false,
		},
		{
			name:     "regexp",
			value:    "value",
			expr:     newRegexpStringExpr(t, "^val.*"),
			expected: true,
		},
		{
			name:     "regexp/empty_value",
			value:    "",
			expr:     newRegexpStringExpr(t, "^val.*"),
			expected: false,
		},
		{
			name:     "regexp/other_value",
			value:    "other_value",
			expr:     newRegexpStringExpr(t, "^val.*"),
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.Evaluate(tc.value))
		})
	}
}

func newRegexpStringExpr(t *testing.T, str string) StringExpr {
	se, err := NewRegexpStringExpr(str)
	require.NoError(t, err)
	return se
}
