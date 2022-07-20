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

package featuregate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlagValue_basic(t *testing.T) {
	for _, tc := range []struct {
		name       string
		input      string
		output     FlagValue
		skipString bool
	}{
		{
			name:   "empty item",
			input:  "",
			output: FlagValue{},
		},
		{
			name:   "single item",
			input:  "foo",
			output: FlagValue{"foo": true},
		},
		{
			name:   "single disabled item",
			input:  "-foo",
			output: FlagValue{"foo": false},
		},
		{
			name:   "multiple items",
			input:  "-bar,foo",
			output: FlagValue{"foo": true, "bar": false},
		},
		{
			name:   "multiple positive items with namespaces",
			input:  "bar.baz,foo.bar",
			output: FlagValue{"foo.bar": true, "bar.baz": true},
		},
		{
			name:       "repeated items",
			input:      "foo,-bar,-foo",
			output:     FlagValue{"foo": true, "bar": false},
			skipString: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := FlagValue{}
			assert.NoError(t, v.Set(tc.input))
			assert.Equal(t, tc.output, v)

			if tc.skipString {
				return
			}

			// Check that the String() func returns the result as the input.
			assert.Equal(t, tc.input, tc.output.String())
		})
	}
}

func TestFlagValue_withPlus(t *testing.T) {
	for _, tc := range []struct {
		name     string
		expected string
		input    FlagValue
	}{
		{
			name:     "single item",
			input:    FlagValue{"foo": true},
			expected: "+foo",
		},
		{
			name:     "multiple items",
			input:    FlagValue{"foo": true, "bar": false},
			expected: "-bar,+foo",
		},
		{
			name:     "multiple positive items with namespaces",
			input:    FlagValue{"foo.bar": true, "bar.baz": true},
			expected: "+bar.baz,+foo.bar",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := FlagValue{}
			assert.NoError(t, v.Set(tc.expected))
			assert.Equal(t, tc.input, v)
		})
	}
}
