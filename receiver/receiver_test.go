// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver/internal"
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
	_, err := f.CreateTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.ErrorIs(t, err, pipeline.ErrSignalNotSupported)
	_, err = f.CreateMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.ErrorIs(t, err, pipeline.ErrSignalNotSupported)
	_, err = f.CreateLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.ErrorIs(t, err, pipeline.ErrSignalNotSupported)
}

func TestNewFactoryWithOptions(t *testing.T) {
	defaultCfg := struct{}{}
	f := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithTraces(createTraces, component.StabilityLevelDeprecated),
		WithMetrics(createMetrics, component.StabilityLevelAlpha),
		WithLogs(createLogs, component.StabilityLevelStable))
	assert.Equal(t, testType, f.Type())
	assert.EqualValues(t, &defaultCfg, f.CreateDefaultConfig())

	wrongID := component.MustNewID("wrong")
	wrongIDErrStr := internal.ErrIDMismatch(wrongID, testType).Error()

	assert.Equal(t, component.StabilityLevelDeprecated, f.TracesStability())
	_, err := f.CreateTraces(context.Background(), Settings{ID: testID}, &defaultCfg, nil)
	require.NoError(t, err)
	_, err = f.CreateTraces(context.Background(), Settings{ID: wrongID}, &defaultCfg, nil)
	require.EqualError(t, err, wrongIDErrStr)

	assert.Equal(t, component.StabilityLevelAlpha, f.MetricsStability())
	_, err = f.CreateMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, nil)
	require.NoError(t, err)
	_, err = f.CreateMetrics(context.Background(), Settings{ID: wrongID}, &defaultCfg, nil)
	require.EqualError(t, err, wrongIDErrStr)

	assert.Equal(t, component.StabilityLevelStable, f.LogsStability())
	_, err = f.CreateLogs(context.Background(), Settings{ID: testID}, &defaultCfg, nil)
	require.NoError(t, err)
	_, err = f.CreateLogs(context.Background(), Settings{ID: wrongID}, &defaultCfg, nil)
	require.EqualError(t, err, wrongIDErrStr)
}

var nopInstance = &nopReceiver{
	Consumer: consumertest.NewNop(),
}

// nopReceiver stores consumed traces and metrics for testing purposes.
type nopReceiver struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createTraces(context.Context, Settings, component.Config, consumer.Traces) (Traces, error) {
	return nopInstance, nil
}

func createMetrics(context.Context, Settings, component.Config, consumer.Metrics) (Metrics, error) {
	return nopInstance, nil
}

func createLogs(context.Context, Settings, component.Config, consumer.Logs) (Logs, error) {
	return nopInstance, nil
}
