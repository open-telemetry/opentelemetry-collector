// Copyright 2019, OpenTelemetry Authors
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

package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
)

type TestFactory struct {
	name string
}

// Type gets the type of the Processor config created by this factory.
func (f *TestFactory) Type() string {
	return f.name
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (f *TestFactory) CreateDefaultConfig() configmodels.Processor {
	return nil
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *TestFactory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor,
) (TraceProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *TestFactory) CreateMetricsProcessor(
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (MetricsProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

func TestFactoriesBuilder(t *testing.T) {
	type testCase struct {
		in  []Factory
		out map[string]Factory
		err bool
	}

	testCases := []testCase{
		{
			in: []Factory{
				&TestFactory{"p1"},
				&TestFactory{"p2"},
			},
			out: map[string]Factory{
				"p1": &TestFactory{"p1"},
				"p2": &TestFactory{"p2"},
			},
			err: false,
		},
		{
			in: []Factory{
				&TestFactory{"p1"},
				&TestFactory{"p1"},
			},
			err: true,
		},
	}

	for _, c := range testCases {
		out, err := Build(c.in...)
		if c.err {
			assert.NotNil(t, err)
			continue
		}
		assert.Nil(t, err)
		assert.Equal(t, c.out, out)
	}
}
