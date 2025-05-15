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
	"go.opentelemetry.io/collector/exporter/internal/experr"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	testType = component.MustNewType("test")
	testID   = component.NewID(testType)
)

func TestNewFactory(t *testing.T) {
	defaultCfg := struct{}{}
	f := NewFactory(
		testType,
		func() component.Config { return &defaultCfg })
	assert.Equal(t, testType, f.Type())
	assert.EqualValues(t, &defaultCfg, f.CreateDefaultConfig())
	_, err := f.CreateTraces(context.Background(), Settings{ID: testID}, &defaultCfg)
	require.ErrorIs(t, err, pipeline.ErrSignalNotSupported)
	_, err = f.CreateMetrics(context.Background(), Settings{ID: testID}, &defaultCfg)
	require.ErrorIs(t, err, pipeline.ErrSignalNotSupported)
	_, err = f.CreateLogs(context.Background(), Settings{ID: testID}, &defaultCfg)
	require.ErrorIs(t, err, pipeline.ErrSignalNotSupported)
}

func TestNewFactoryWithOptions(t *testing.T) {
	defaultCfg := struct{}{}
	f := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithTraces(createTraces, component.StabilityLevelDevelopment),
		WithMetrics(createMetrics, component.StabilityLevelAlpha),
		WithLogs(createLogs, component.StabilityLevelDeprecated))
	assert.Equal(t, testType, f.Type())
	assert.EqualValues(t, &defaultCfg, f.CreateDefaultConfig())

	wrongID := component.MustNewID("wrong")
	wrongIDErrStr := experr.ErrIDMismatch(wrongID, testType).Error()

	assert.Equal(t, component.StabilityLevelDevelopment, f.TracesStability())
	_, err := f.CreateTraces(context.Background(), Settings{ID: testID}, &defaultCfg)
	require.NoError(t, err)
	_, err = f.CreateTraces(context.Background(), Settings{ID: wrongID}, &defaultCfg)
	require.EqualError(t, err, wrongIDErrStr)

	assert.Equal(t, component.StabilityLevelAlpha, f.MetricsStability())
	_, err = f.CreateMetrics(context.Background(), Settings{ID: testID}, &defaultCfg)
	require.NoError(t, err)
	_, err = f.CreateMetrics(context.Background(), Settings{ID: wrongID}, &defaultCfg)
	require.EqualError(t, err, wrongIDErrStr)

	assert.Equal(t, component.StabilityLevelDeprecated, f.LogsStability())
	_, err = f.CreateLogs(context.Background(), Settings{ID: testID}, &defaultCfg)
	require.NoError(t, err)
	_, err = f.CreateLogs(context.Background(), Settings{ID: wrongID}, &defaultCfg)
	require.EqualError(t, err, wrongIDErrStr)
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
