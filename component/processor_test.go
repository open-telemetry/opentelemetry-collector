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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

type TestProcessorFactory struct {
	name string
}

// Type gets the type of the Processor config created by this factory.
func (f *TestProcessorFactory) Type() configmodels.Type {
	return configmodels.Type(f.name)
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (f *TestProcessorFactory) CreateDefaultConfig() configmodels.Processor {
	return nil
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *TestProcessorFactory) CreateTracesProcessor(context.Context, ProcessorCreateParams, configmodels.Processor, consumer.TracesConsumer) (TracesProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *TestProcessorFactory) CreateMetricsProcessor(context.Context, ProcessorCreateParams, configmodels.Processor, consumer.MetricsConsumer) (MetricsProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *TestProcessorFactory) CreateLogsProcessor(context.Context, ProcessorCreateParams, configmodels.Processor, consumer.LogsConsumer) (LogsProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

func TestFactoriesBuilder(t *testing.T) {
	type testCase struct {
		in  []ProcessorFactory
		out map[configmodels.Type]ProcessorFactory
	}

	testCases := []testCase{
		{
			in: []ProcessorFactory{
				&TestProcessorFactory{"p1"},
				&TestProcessorFactory{"p2"},
			},
			out: map[configmodels.Type]ProcessorFactory{
				"p1": &TestProcessorFactory{"p1"},
				"p2": &TestProcessorFactory{"p2"},
			},
		},
		{
			in: []ProcessorFactory{
				&TestProcessorFactory{"p1"},
				&TestProcessorFactory{"p1"},
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
