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

var _ ProcessorFactory = (*TestProcessorFactory)(nil)

type TestProcessorFactory struct {
	ProcessorFactory
	name string
}

// Type gets the type of the Processor config created by this factory.
func (f *TestProcessorFactory) Type() config.Type {
	return config.Type(f.name)
}

func TestMakeProcessorFactoryMap(t *testing.T) {
	type testCase struct {
		in  []ProcessorFactory
		out map[config.Type]ProcessorFactory
	}

	testCases := []testCase{
		{
			in: []ProcessorFactory{
				&TestProcessorFactory{name: "p1"},
				&TestProcessorFactory{name: "p2"},
			},
			out: map[config.Type]ProcessorFactory{
				"p1": &TestProcessorFactory{name: "p1"},
				"p2": &TestProcessorFactory{name: "p2"},
			},
		},
		{
			in: []ProcessorFactory{
				&TestProcessorFactory{name: "p1"},
				&TestProcessorFactory{name: "p1"},
			},
		},
	}

	for _, c := range testCases {
		out, err := MakeProcessorFactoryMap(c.in...)
		if c.out == nil {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, c.out, out)
	}
}
