// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor/internal"
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
		WithTraces(createTraces, component.StabilityLevelAlpha),
		WithMetrics(createMetrics, component.StabilityLevelBeta),
		WithLogs(createLogs, component.StabilityLevelUnmaintained))
	assert.Equal(t, testType, f.Type())
	assert.EqualValues(t, &defaultCfg, f.CreateDefaultConfig())

	wrongID := component.MustNewID("wrong")
	wrongIDErrStr := internal.ErrIDMismatch(wrongID, testType).Error()

	assert.Equal(t, component.StabilityLevelAlpha, f.TracesStability())
	_, err := f.CreateTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	_, err = f.CreateTraces(context.Background(), Settings{ID: wrongID}, &defaultCfg, consumertest.NewNop())
	require.EqualError(t, err, wrongIDErrStr)

	assert.Equal(t, component.StabilityLevelBeta, f.MetricsStability())
	_, err = f.CreateMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	_, err = f.CreateMetrics(context.Background(), Settings{ID: wrongID}, &defaultCfg, consumertest.NewNop())
	require.EqualError(t, err, wrongIDErrStr)

	assert.Equal(t, component.StabilityLevelUnmaintained, f.LogsStability())
	_, err = f.CreateLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	_, err = f.CreateLogs(context.Background(), Settings{ID: wrongID}, &defaultCfg, consumertest.NewNop())
	require.EqualError(t, err, wrongIDErrStr)
}

var nopInstance = &nopProcessor{
	Consumer: consumertest.NewNop(),
}

// nopProcessor stores consumed traces and metrics for testing purposes.
type nopProcessor struct {
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
