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

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

type TestReceiverFactory struct {
	name config.Type
}

// Type gets the type of the Receiver config created by this factory.
func (f *TestReceiverFactory) Type() config.Type {
	return f.name
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *TestReceiverFactory) CreateDefaultConfig() config.Receiver {
	return nil
}

// CreateTracesReceiver creates a trace receiver based on this config.
func (f *TestReceiverFactory) CreateTracesReceiver(context.Context, ReceiverCreateSettings, config.Receiver, consumer.Traces) (TracesReceiver, error) {
	return nil, componenterror.ErrDataTypeIsNotSupported
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *TestReceiverFactory) CreateMetricsReceiver(context.Context, ReceiverCreateSettings, config.Receiver, consumer.Metrics) (MetricsReceiver, error) {
	return nil, componenterror.ErrDataTypeIsNotSupported
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *TestReceiverFactory) CreateLogsReceiver(context.Context, ReceiverCreateSettings, config.Receiver, consumer.Logs) (LogsReceiver, error) {
	return nil, componenterror.ErrDataTypeIsNotSupported
}

func TestBuildReceivers(t *testing.T) {
	type testCase struct {
		in  []ReceiverFactory
		out map[config.Type]ReceiverFactory
	}

	testCases := []testCase{
		{
			in: []ReceiverFactory{
				&TestReceiverFactory{"e1"},
				&TestReceiverFactory{"e2"},
			},
			out: map[config.Type]ReceiverFactory{
				"e1": &TestReceiverFactory{"e1"},
				"e2": &TestReceiverFactory{"e2"},
			},
		},
		{
			in: []ReceiverFactory{
				&TestReceiverFactory{"e1"},
				&TestReceiverFactory{"e1"},
			},
		},
	}

	for _, c := range testCases {
		out, err := MakeReceiverFactoryMap(c.in...)
		if c.out == nil {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, c.out, out)
	}
}
