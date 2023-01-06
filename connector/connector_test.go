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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

func TestNewFactoryNoOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := struct{}{}
	factory := NewFactory(typeStr, func() component.Config { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	testID := component.NewIDWithName("type", "name")

	_, err := factory.CreateTracesToTraces(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeTraces, component.DataTypeTraces))
	_, err = factory.CreateTracesToMetrics(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeTraces, component.DataTypeMetrics))
	_, err = factory.CreateTracesToLogs(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeTraces, component.DataTypeLogs))

	_, err = factory.CreateMetricsToTraces(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeMetrics, component.DataTypeTraces))
	_, err = factory.CreateMetricsToMetrics(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeMetrics, component.DataTypeMetrics))
	_, err = factory.CreateMetricsToLogs(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeMetrics, component.DataTypeLogs))

	_, err = factory.CreateLogsToTraces(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeLogs, component.DataTypeTraces))
	_, err = factory.CreateLogsToMetrics(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeLogs, component.DataTypeMetrics))
	_, err = factory.CreateLogsToLogs(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeLogs, component.DataTypeLogs))
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

	testID := component.NewIDWithName("type", "name")

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToTracesStability())
	_, err := factory.CreateTracesToTraces(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToMetricsStability())
	_, err = factory.CreateMetricsToMetrics(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToLogsStability())
	_, err = factory.CreateLogsToLogs(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateTracesToMetrics(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeTraces, component.DataTypeMetrics))
	_, err = factory.CreateTracesToLogs(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeTraces, component.DataTypeLogs))

	_, err = factory.CreateMetricsToTraces(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeMetrics, component.DataTypeTraces))
	_, err = factory.CreateMetricsToLogs(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeMetrics, component.DataTypeLogs))

	_, err = factory.CreateLogsToTraces(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeLogs, component.DataTypeTraces))
	_, err = factory.CreateLogsToMetrics(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeLogs, component.DataTypeMetrics))
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

	testID := component.NewIDWithName("type", "name")

	_, err := factory.CreateTracesToTraces(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeTraces, component.DataTypeTraces))
	_, err = factory.CreateMetricsToMetrics(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeMetrics, component.DataTypeMetrics))
	_, err = factory.CreateLogsToLogs(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.Equal(t, err, errDataTypes(testID, component.DataTypeLogs, component.DataTypeLogs))

	assert.Equal(t, component.StabilityLevelDevelopment, factory.TracesToMetricsStability())
	_, err = factory.CreateTracesToMetrics(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToLogsStability())
	_, err = factory.CreateTracesToLogs(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToTracesStability())
	_, err = factory.CreateMetricsToTraces(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelStable, factory.MetricsToLogsStability())
	_, err = factory.CreateMetricsToLogs(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.LogsToTracesStability())
	_, err = factory.CreateLogsToTraces(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToMetricsStability())
	_, err = factory.CreateLogsToMetrics(context.Background(), CreateSettings{ID: testID}, &defaultCfg, nil)
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

func TestBuilder(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := MakeFactoryMap([]Factory{
		NewFactory("err", nil),
		NewFactory(
			"all",
			func() component.Config { return &defaultCfg },
			WithTracesToTraces(createTracesToTraces, component.StabilityLevelDevelopment),
			WithTracesToMetrics(createTracesToMetrics, component.StabilityLevelDevelopment),
			WithTracesToLogs(createTracesToLogs, component.StabilityLevelDevelopment),
			WithMetricsToTraces(createMetricsToTraces, component.StabilityLevelAlpha),
			WithMetricsToMetrics(createMetricsToMetrics, component.StabilityLevelAlpha),
			WithMetricsToLogs(createMetricsToLogs, component.StabilityLevelAlpha),
			WithLogsToTraces(createLogsToTraces, component.StabilityLevelDeprecated),
			WithLogsToMetrics(createLogsToMetrics, component.StabilityLevelDeprecated),
			WithLogsToLogs(createLogsToLogs, component.StabilityLevelDeprecated),
		),
	}...)
	require.NoError(t, err)

	testCases := []struct {
		name string
		id   component.ID
		err  func(component.DataType, component.DataType) string
	}{
		{
			name: "unknown",
			id:   component.NewID("unknown"),
			err: func(_, _ component.DataType) string {
				return "connector factory not available for: \"unknown\""
			},
		},
		{
			name: "err",
			id:   component.NewID("err"),
			err: func(expType, rcvType component.DataType) string {
				return fmt.Sprintf("connector \"err\" cannot connect from %s to %s: telemetry type is not supported", expType, rcvType)
			},
		},
		{
			name: "all",
			id:   component.NewID("all"),
			err: func(_, _ component.DataType) string {
				return ""
			},
		},
		{
			name: "all/named",
			id:   component.NewIDWithName("all", "named"),
			err: func(_, _ component.DataType) string {
				return ""
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfgs := map[component.ID]component.Config{tt.id: defaultCfg}
			b := NewBuilder(cfgs, factories)

			t2t, err := b.CreateTracesToTraces(context.Background(), createSettings(tt.id), nil)
			if expectedErr := tt.err(component.DataTypeTraces, component.DataTypeTraces); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, t2t)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, t2t)
			}
			t2m, err := b.CreateTracesToMetrics(context.Background(), createSettings(tt.id), nil)
			if expectedErr := tt.err(component.DataTypeTraces, component.DataTypeMetrics); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, t2m)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, t2m)
			}
			t2l, err := b.CreateTracesToLogs(context.Background(), createSettings(tt.id), nil)
			if expectedErr := tt.err(component.DataTypeTraces, component.DataTypeLogs); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, t2l)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, t2l)
			}

			m2t, err := b.CreateMetricsToTraces(context.Background(), createSettings(tt.id), nil)
			if expectedErr := tt.err(component.DataTypeMetrics, component.DataTypeTraces); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, m2t)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, m2t)
			}

			m2m, err := b.CreateMetricsToMetrics(context.Background(), createSettings(tt.id), nil)
			if expectedErr := tt.err(component.DataTypeMetrics, component.DataTypeMetrics); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, m2m)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, m2m)
			}

			m2l, err := b.CreateMetricsToLogs(context.Background(), createSettings(tt.id), nil)
			if expectedErr := tt.err(component.DataTypeMetrics, component.DataTypeLogs); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, m2l)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, m2l)
			}

			l2t, err := b.CreateLogsToTraces(context.Background(), createSettings(tt.id), nil)
			if expectedErr := tt.err(component.DataTypeLogs, component.DataTypeTraces); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, l2t)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, l2t)
			}

			l2m, err := b.CreateLogsToMetrics(context.Background(), createSettings(tt.id), nil)
			if expectedErr := tt.err(component.DataTypeLogs, component.DataTypeMetrics); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, l2m)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, l2m)
			}

			l2l, err := b.CreateLogsToLogs(context.Background(), createSettings(tt.id), nil)
			if expectedErr := tt.err(component.DataTypeLogs, component.DataTypeLogs); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, l2l)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, l2l)
			}
		})
	}
}

func TestBuilderMissingConfig(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := MakeFactoryMap([]Factory{
		NewFactory(
			"all",
			func() component.Config { return &defaultCfg },
			WithTracesToTraces(createTracesToTraces, component.StabilityLevelDevelopment),
			WithTracesToMetrics(createTracesToMetrics, component.StabilityLevelDevelopment),
			WithTracesToLogs(createTracesToLogs, component.StabilityLevelDevelopment),
			WithMetricsToTraces(createMetricsToTraces, component.StabilityLevelAlpha),
			WithMetricsToMetrics(createMetricsToMetrics, component.StabilityLevelAlpha),
			WithMetricsToLogs(createMetricsToLogs, component.StabilityLevelAlpha),
			WithLogsToTraces(createLogsToTraces, component.StabilityLevelDeprecated),
			WithLogsToMetrics(createLogsToMetrics, component.StabilityLevelDeprecated),
			WithLogsToLogs(createLogsToLogs, component.StabilityLevelDeprecated),
		),
	}...)

	require.NoError(t, err)

	bErr := NewBuilder(map[component.ID]component.Config{}, factories)
	missingID := component.NewIDWithName("all", "missing")

	t2t, err := bErr.CreateTracesToTraces(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, t2t)

	t2m, err := bErr.CreateTracesToMetrics(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, t2m)

	t2l, err := bErr.CreateTracesToLogs(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, t2l)

	m2t, err := bErr.CreateMetricsToTraces(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, m2t)

	m2m, err := bErr.CreateMetricsToMetrics(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, m2m)

	m2l, err := bErr.CreateMetricsToLogs(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, m2l)

	l2t, err := bErr.CreateLogsToTraces(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, l2t)

	l2m, err := bErr.CreateLogsToMetrics(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, l2m)

	l2l, err := bErr.CreateLogsToLogs(context.Background(), createSettings(missingID), nil)
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, l2l)
}

func TestBuilderGetters(t *testing.T) {
	factories, err := MakeFactoryMap([]Factory{NewFactory("foo", nil)}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{component.NewID("foo"): struct{}{}}
	b := NewBuilder(cfgs, factories)

	assert.True(t, b.IsConfigured(component.NewID("foo")))
	assert.False(t, b.IsConfigured(component.NewID("bar")))

	assert.NotNil(t, b.Factory(component.NewID("foo").Type()))
	assert.Nil(t, b.Factory(component.NewID("bar").Type()))
}

var nopInstance = &nopConnector{
	Consumer: consumertest.NewNop(),
}

// nopConnector stores consumed traces and metrics for testing purposes.
type nopConnector struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createTracesToTraces(context.Context, CreateSettings, component.Config, consumer.Traces) (Traces, error) {
	return nopInstance, nil
}
func createTracesToMetrics(context.Context, CreateSettings, component.Config, consumer.Metrics) (Traces, error) {
	return nopInstance, nil
}
func createTracesToLogs(context.Context, CreateSettings, component.Config, consumer.Logs) (Traces, error) {
	return nopInstance, nil
}

func createMetricsToTraces(context.Context, CreateSettings, component.Config, consumer.Traces) (Metrics, error) {
	return nopInstance, nil
}
func createMetricsToMetrics(context.Context, CreateSettings, component.Config, consumer.Metrics) (Metrics, error) {
	return nopInstance, nil
}
func createMetricsToLogs(context.Context, CreateSettings, component.Config, consumer.Logs) (Metrics, error) {
	return nopInstance, nil
}

func createLogsToTraces(context.Context, CreateSettings, component.Config, consumer.Traces) (Logs, error) {
	return nopInstance, nil
}
func createLogsToMetrics(context.Context, CreateSettings, component.Config, consumer.Metrics) (Logs, error) {
	return nopInstance, nil
}
func createLogsToLogs(context.Context, CreateSettings, component.Config, consumer.Logs) (Logs, error) {
	return nopInstance, nil
}

func createSettings(id component.ID) CreateSettings {
	return CreateSettings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
