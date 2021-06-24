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
)

type TestExporterFactory struct {
	name string
}

// Type gets the type of the Exporter config created by this factory.
func (f *TestExporterFactory) Type() config.Type {
	return config.Type(f.name)
}

// CreateDefaultConfig creates the default configuration for the Exporter.
func (f *TestExporterFactory) CreateDefaultConfig() config.Exporter {
	return nil
}

// CreateTracesExporter creates a trace exporter based on this config.
func (f *TestExporterFactory) CreateTracesExporter(context.Context, ExporterCreateSettings, config.Exporter) (TracesExporter, error) {
	return nil, componenterror.ErrDataTypeIsNotSupported
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *TestExporterFactory) CreateMetricsExporter(context.Context, ExporterCreateSettings, config.Exporter) (MetricsExporter, error) {
	return nil, componenterror.ErrDataTypeIsNotSupported
}

// CreateLogsExporter creates a logs exporter based on this config.
func (f *TestExporterFactory) CreateLogsExporter(context.Context, ExporterCreateSettings, config.Exporter) (LogsExporter, error) {
	return nil, componenterror.ErrDataTypeIsNotSupported
}

func TestBuildExporters(t *testing.T) {
	type testCase struct {
		in  []ExporterFactory
		out map[config.Type]ExporterFactory
	}

	testCases := []testCase{
		{
			in: []ExporterFactory{
				&TestExporterFactory{"exp1"},
				&TestExporterFactory{"exp2"},
			},
			out: map[config.Type]ExporterFactory{
				"exp1": &TestExporterFactory{"exp1"},
				"exp2": &TestExporterFactory{"exp2"},
			},
		},
		{
			in: []ExporterFactory{
				&TestExporterFactory{"exp1"},
				&TestExporterFactory{"exp1"},
			},
		},
	}

	for _, c := range testCases {
		out, err := MakeExporterFactoryMap(c.in...)
		if c.out == nil {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, c.out, out)
	}
}
