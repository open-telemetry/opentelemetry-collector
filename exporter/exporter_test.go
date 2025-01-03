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
	"go.opentelemetry.io/collector/pipeline"
)

func TestNewFactory(t *testing.T) {
	testType := component.MustNewType("test")
	defaultCfg := struct{}{}
	f := NewFactory(
		testType,
		func() component.Config { return &defaultCfg })
	assert.EqualValues(t, testType, f.Type())
	assert.EqualValues(t, &defaultCfg, f.CreateDefaultConfig())
	_, err := f.CreateTraces(context.Background(), Settings{}, &defaultCfg)
	require.ErrorIs(t, err, pipeline.ErrSignalNotSupported)
	_, err = f.CreateMetrics(context.Background(), Settings{}, &defaultCfg)
	require.ErrorIs(t, err, pipeline.ErrSignalNotSupported)
	_, err = f.CreateLogs(context.Background(), Settings{}, &defaultCfg)
	require.ErrorIs(t, err, pipeline.ErrSignalNotSupported)
}

func TestNewFactoryWithOptions(t *testing.T) {
	testType := component.MustNewType("test")
	defaultCfg := struct{}{}
	f := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithTraces(createTraces, component.StabilityLevelDevelopment),
		WithMetrics(createMetrics, component.StabilityLevelAlpha),
		WithLogs(createLogs, component.StabilityLevelDeprecated))
	assert.EqualValues(t, testType, f.Type())
	assert.EqualValues(t, &defaultCfg, f.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDevelopment, f.TracesStability())
	_, err := f.CreateTraces(context.Background(), Settings{}, &defaultCfg)
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, f.MetricsStability())
	_, err = f.CreateMetrics(context.Background(), Settings{}, &defaultCfg)
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, f.LogsStability())
	_, err = f.CreateLogs(context.Background(), Settings{}, &defaultCfg)
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
