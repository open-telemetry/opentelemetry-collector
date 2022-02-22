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

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config"
)

func TestMakeExtensionFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []ExtensionFactory
		out  map[config.Type]ExtensionFactory
	}

	p1 := NewExtensionFactory("p1", nil, nil)
	p2 := NewExtensionFactory("p2", nil, nil)
	testCases := []testCase{
		{
			name: "different names",
			in:   []ExtensionFactory{p1, p2},
			out: map[config.Type]ExtensionFactory{
				p1.Type(): p1,
				p2.Type(): p2,
			},
		},
		{
			name: "same name",
			in:   []ExtensionFactory{p1, p2, NewExtensionFactory("p1", nil, nil)},
		},
	}

	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeExtensionFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}
