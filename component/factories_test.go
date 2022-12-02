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
)

func TestMakeReceiverFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []ReceiverFactory
		out  map[Type]ReceiverFactory
	}

	p1 := NewReceiverFactory("p1", nil)
	p2 := NewReceiverFactory("p2", nil)
	testCases := []testCase{
		{
			name: "different names",
			in:   []ReceiverFactory{p1, p2},
			out: map[Type]ReceiverFactory{
				p1.Type(): p1,
				p2.Type(): p2,
			},
		},
		{
			name: "same name",
			in:   []ReceiverFactory{p1, p2, NewReceiverFactory("p1", nil)},
		},
	}

	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeReceiverFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestMakeProcessorFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []ProcessorFactory
		out  map[Type]ProcessorFactory
	}

	p1 := NewProcessorFactory("p1", nil)
	p2 := NewProcessorFactory("p2", nil)
	testCases := []testCase{
		{
			name: "different names",
			in:   []ProcessorFactory{p1, p2},
			out: map[Type]ProcessorFactory{
				p1.Type(): p1,
				p2.Type(): p2,
			},
		},
		{
			name: "same name",
			in:   []ProcessorFactory{p1, p2, NewProcessorFactory("p1", nil)},
		},
	}

	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeProcessorFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}
