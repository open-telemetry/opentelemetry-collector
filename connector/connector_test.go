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

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

func TestNewFactoryNoOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := struct{}{}
	factory := NewFactory(typeStr, func() component.Config { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errTracesToTraces)
	_, err = factory.CreateTracesToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errTracesToMetrics)
	_, err = factory.CreateTracesToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errTracesToLogs)

	_, err = factory.CreateMetricsToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errMetricsToTraces)
	_, err = factory.CreateMetricsToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errMetricsToMetrics)
	_, err = factory.CreateMetricsToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errMetricsToLogs)

	_, err = factory.CreateLogsToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errLogsToTraces)
	_, err = factory.CreateLogsToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errLogsToMetrics)
	_, err = factory.CreateLogsToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errLogsToLogs)
}

func TestNewFactoryWithSameTypes(t *testing.T) {
	const typeStr = "test"
	defaultCfg := struct{}{}
	factory := NewFactory(typeStr, func() component.Config { return &defaultCfg },
		WithTracesToTraces(createTracesToTraces, component.StabilityLevelAlpha),
		WithMetricsToMetrics(createMetricsToMetrics, component.StabilityLevelBeta),
		WithLogsToLogs(createLogsToLogs, component.StabilityLevelUnmaintained))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToTracesStability())
	_, err := factory.CreateTracesToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToMetricsStability())
	_, err = factory.CreateMetricsToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToLogsStability())
	_, err = factory.CreateLogsToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateTracesToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errTracesToMetrics)
	_, err = factory.CreateTracesToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errTracesToLogs)

	_, err = factory.CreateMetricsToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errMetricsToTraces)
	_, err = factory.CreateMetricsToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errMetricsToLogs)

	_, err = factory.CreateLogsToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errLogsToTraces)
	_, err = factory.CreateLogsToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errLogsToMetrics)
}

func TestNewFactoryWithTranslateTypes(t *testing.T) {
	const typeStr = "test"
	defaultCfg := struct{}{}
	factory := NewFactory(typeStr, func() component.Config { return &defaultCfg },
		WithTracesToMetrics(createTracesToMetrics, component.StabilityLevelDevelopment),
		WithTracesToLogs(createTracesToLogs, component.StabilityLevelAlpha),
		WithMetricsToTraces(createMetricsToTraces, component.StabilityLevelBeta),
		WithMetricsToLogs(createMetricsToLogs, component.StabilityLevelStable),
		WithLogsToTraces(createLogsToTraces, component.StabilityLevelDeprecated),
		WithLogsToMetrics(createLogsToMetrics, component.StabilityLevelUnmaintained))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errTracesToTraces)
	_, err = factory.CreateMetricsToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errMetricsToMetrics)
	_, err = factory.CreateLogsToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, errLogsToLogs)

	assert.Equal(t, component.StabilityLevelDevelopment, factory.TracesToMetricsStability())
	_, err = factory.CreateTracesToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToLogsStability())
	_, err = factory.CreateTracesToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToTracesStability())
	_, err = factory.CreateMetricsToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelStable, factory.MetricsToLogsStability())
	_, err = factory.CreateMetricsToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.LogsToTracesStability())
	_, err = factory.CreateLogsToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToMetricsStability())
	_, err = factory.CreateLogsToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func TestNewFactoryWithAllTypes(t *testing.T) {
	const typeStr = "test"
	defaultCfg := struct{}{}
	factory := NewFactory(typeStr, func() component.Config { return &defaultCfg },
		WithTracesToTraces(createTracesToTraces, component.StabilityLevelAlpha),
		WithTracesToMetrics(createTracesToMetrics, component.StabilityLevelDevelopment),
		WithTracesToLogs(createTracesToLogs, component.StabilityLevelAlpha),
		WithMetricsToTraces(createMetricsToTraces, component.StabilityLevelBeta),
		WithMetricsToMetrics(createMetricsToMetrics, component.StabilityLevelBeta),
		WithMetricsToLogs(createMetricsToLogs, component.StabilityLevelStable),
		WithLogsToTraces(createLogsToTraces, component.StabilityLevelDeprecated),
		WithLogsToMetrics(createLogsToMetrics, component.StabilityLevelUnmaintained),
		WithLogsToLogs(createLogsToLogs, component.StabilityLevelUnmaintained))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToTracesStability())
	_, err := factory.CreateTracesToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelDevelopment, factory.TracesToMetricsStability())
	_, err = factory.CreateTracesToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToLogsStability())
	_, err = factory.CreateTracesToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToTracesStability())
	_, err = factory.CreateMetricsToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToMetricsStability())
	_, err = factory.CreateMetricsToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelStable, factory.MetricsToLogsStability())
	_, err = factory.CreateMetricsToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.LogsToTracesStability())
	_, err = factory.CreateLogsToTraces(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToMetricsStability())
	_, err = factory.CreateLogsToMetrics(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToLogsStability())
	_, err = factory.CreateLogsToLogs(context.Background(), CreateSettings{}, &defaultCfg, nil)
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

func createTracesToTraces(context.Context, CreateSettings, component.Config, consumer.Traces) (Traces, error) {
	return nil, nil
}
func createTracesToMetrics(context.Context, CreateSettings, component.Config, consumer.Metrics) (Traces, error) {
	return nil, nil
}
func createTracesToLogs(context.Context, CreateSettings, component.Config, consumer.Logs) (Traces, error) {
	return nil, nil
}

func createMetricsToTraces(context.Context, CreateSettings, component.Config, consumer.Traces) (Metrics, error) {
	return nil, nil
}
func createMetricsToMetrics(context.Context, CreateSettings, component.Config, consumer.Metrics) (Metrics, error) {
	return nil, nil
}
func createMetricsToLogs(context.Context, CreateSettings, component.Config, consumer.Logs) (Metrics, error) {
	return nil, nil
}

func createLogsToTraces(context.Context, CreateSettings, component.Config, consumer.Traces) (Logs, error) {
	return nil, nil
}
func createLogsToMetrics(context.Context, CreateSettings, component.Config, consumer.Metrics) (Logs, error) {
	return nil, nil
}
func createLogsToLogs(context.Context, CreateSettings, component.Config, consumer.Logs) (Logs, error) {
	return nil, nil
}
