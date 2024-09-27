// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

func TestNewFactory(t *testing.T) {
	var testType = component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg })
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTraces(context.Background(), Settings{}, &defaultCfg)
	require.Error(t, err)
	_, err = factory.CreateMetrics(context.Background(), Settings{}, &defaultCfg)
	require.Error(t, err)
	_, err = factory.CreateLogsExporter(context.Background(), Settings{}, &defaultCfg)
	require.Error(t, err)
	_, err = factory.CreateEntities(context.Background(), Settings{}, &defaultCfg)
	require.Error(t, err)
}

func TestNewFactoryWithOptions(t *testing.T) {
	var testType = component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithTraces(createTraces, component.StabilityLevelDevelopment),
		WithMetrics(createMetrics, component.StabilityLevelAlpha),
		WithLogs(createLogs, component.StabilityLevelDeprecated))
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDevelopment, factory.TracesStability())
	_, err := factory.CreateTraces(context.Background(), Settings{}, &defaultCfg)
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.MetricsStability())
	_, err = factory.CreateMetrics(context.Background(), Settings{}, &defaultCfg)
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.LogsStability())
	_, err = factory.CreateLogs(context.Background(), Settings{}, &defaultCfg)
	assert.NoError(t, err)
}

func TestMakeFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []Factory
		out  map[component.Type]Factory
	}

	p1 := NewFactory(component.MustNewType("p1"), nil)
	p2 := NewFactory(component.MustNewType("p2"), nil)
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
			in:   []Factory{p1, p2, NewFactory(component.MustNewType("p1"), nil)},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}

var nopInstance = &nop{
	Consumer: consumertest.NewNop(),
}

// nop stores consumed traces and metrics for testing purposes.
type nop struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createTraces(context.Context, Settings, component.Config) (Traces, error) {
	return nopInstance, nil
}

func createMetrics(context.Context, Settings, component.Config) (Metrics, error) {
	return nopInstance, nil
}

func createLogs(context.Context, Settings, component.Config) (Logs, error) {
	return nopInstance, nil
}
