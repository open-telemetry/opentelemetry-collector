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

package exporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
)

func TestNewFactory(t *testing.T) {
	const typeStr = "test"
	defaultCfg := struct{}{}
	factory := NewFactory(
		typeStr,
		func() component.Config { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesExporter(context.Background(), CreateSettings{}, &defaultCfg)
	assert.Error(t, err)
	_, err = factory.CreateMetricsExporter(context.Background(), CreateSettings{}, &defaultCfg)
	assert.Error(t, err)
	_, err = factory.CreateLogsExporter(context.Background(), CreateSettings{}, &defaultCfg)
	assert.Error(t, err)
}

func TestNewFactoryWithOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := struct{}{}
	factory := NewFactory(
		typeStr,
		func() component.Config { return &defaultCfg },
		WithTraces(createTracesExporter, component.StabilityLevelDevelopment),
		WithMetrics(createMetricsExporter, component.StabilityLevelAlpha),
		WithLogs(createLogsExporter, component.StabilityLevelDeprecated))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDevelopment, factory.TracesExporterStability())
	_, err := factory.CreateTracesExporter(context.Background(), CreateSettings{}, &defaultCfg)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.MetricsExporterStability())
	_, err = factory.CreateMetricsExporter(context.Background(), CreateSettings{}, &defaultCfg)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.LogsExporterStability())
	_, err = factory.CreateLogsExporter(context.Background(), CreateSettings{}, &defaultCfg)
	assert.NoError(t, err)
}

func TestMakeFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []Factory
		out  map[component.Type]Factory
	}

	p1 := NewFactory("p1", nil)
	p2 := NewFactory("p2", nil)
	testCases := []testCase{
		{
			name: "different names",
			in:   []Factory{p1, p2},
			out: map[component.Type]Factory{
				p1.Type(): p1,
				p2.Type(): p2,
			},
		},
		{
			name: "same name",
			in:   []Factory{p1, p2, NewFactory("p1", nil)},
		},
	}

	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}

func createTracesExporter(context.Context, CreateSettings, component.Config) (Traces, error) {
	return nil, nil
}

func createMetricsExporter(context.Context, CreateSettings, component.Config) (Metrics, error) {
	return nil, nil
}

func createLogsExporter(context.Context, CreateSettings, component.Config) (Logs, error) {
	return nil, nil
}
