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
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	Flags(fs)
	assert.Equal(t, gatesList, fs.Lookup(gatesListCfg).Value)
}

func TestGetFlags(t *testing.T) {
	assert.Equal(t, gatesList, GetFlags())
}

func TestFlagValue_basic(t *testing.T) {
	for _, tc := range []struct {
		name     string
		expected string
		input    FlagValue
	}{
		{
			name:     "single item",
			input:    FlagValue{"foo": true},
			expected: "foo",
		},
		{
			name:     "single disabled item",
			input:    FlagValue{"foo": false},
			expected: "-foo",
		},
		{
			name:     "multiple items",
			input:    FlagValue{"foo": true, "bar": false},
			expected: "-bar,foo",
		},
		{
			name:     "multiple positive items with namespaces",
			input:    FlagValue{"foo.bar": true, "bar.baz": true},
			expected: "bar.baz,foo.bar",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.input.String())
			v := FlagValue{}
			assert.NoError(t, v.Set(tc.expected))
			assert.Equal(t, tc.input, v)
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

func TestFlagValue_SetSlice(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    []string
		expected FlagValue
	}{
		{
			name:     "single item",
			input:    []string{"foo"},
			expected: FlagValue{"foo": true},
		},
		{
			name:     "multiple items",
			input:    []string{"foo", "-bar", "+baz"},
			expected: FlagValue{"foo": true, "bar": false, "baz": true},
		},
		{
			name:     "repeated items",
			input:    []string{"foo", "-bar", "-foo"},
			expected: FlagValue{"foo": true, "bar": false},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := FlagValue{}
			assert.NoError(t, v.SetSlice(tc.input))
			assert.Equal(t, tc.expected, v)
		})
	}
}
