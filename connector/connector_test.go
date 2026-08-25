// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	testType = component.MustNewType("test")
	testID   = component.MustNewIDWithName("test", "name")
)

func TestNewFactoryNoOptions(t *testing.T) {
	defaultCfg := struct{}{}
	factory := NewFactory(testType, func() component.Config { return &defaultCfg })
	assert.Equal(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalTraces, pipeline.SignalTraces))
	_, err = factory.CreateTracesToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalTraces, pipeline.SignalMetrics))
	_, err = factory.CreateTracesToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalTraces, pipeline.SignalLogs))

	_, err = factory.CreateMetricsToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalMetrics, pipeline.SignalTraces))
	_, err = factory.CreateMetricsToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalMetrics, pipeline.SignalMetrics))
	_, err = factory.CreateMetricsToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalMetrics, pipeline.SignalLogs))

	_, err = factory.CreateLogsToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalLogs, pipeline.SignalTraces))
	_, err = factory.CreateLogsToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalLogs, pipeline.SignalMetrics))
	_, err = factory.CreateLogsToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalLogs, pipeline.SignalLogs))
}

func TestNewFactoryWithSameTypes(t *testing.T) {
	defaultCfg := struct{}{}
	factory := NewFactory(testType, func() component.Config { return &defaultCfg },
		WithTracesToTraces(createTracesToTraces, component.StabilityLevelAlpha),
		WithMetricsToMetrics(createMetricsToMetrics, component.StabilityLevelBeta),
		WithLogsToLogs(createLogsToLogs, component.StabilityLevelUnmaintained))
	assert.Equal(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	wrongID := component.MustNewID("wrong")
	wrongIDErrStr := internal.ErrIDMismatch(wrongID, testType).Error()

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToTracesStability())
	_, err := factory.CreateTracesToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	_, err = factory.CreateTracesToTraces(context.Background(), Settings{ID: wrongID}, &defaultCfg, consumertest.NewNop())
	require.ErrorContains(t, err, wrongIDErrStr)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToMetricsStability())
	_, err = factory.CreateMetricsToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	_, err = factory.CreateMetricsToMetrics(context.Background(), Settings{ID: wrongID}, &defaultCfg, consumertest.NewNop())
	require.ErrorContains(t, err, wrongIDErrStr)

	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToLogsStability())
	_, err = factory.CreateLogsToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	_, err = factory.CreateLogsToLogs(context.Background(), Settings{ID: wrongID}, &defaultCfg, consumertest.NewNop())
	require.ErrorContains(t, err, wrongIDErrStr)

	_, err = factory.CreateTracesToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalTraces, pipeline.SignalMetrics))
	_, err = factory.CreateTracesToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalTraces, pipeline.SignalLogs))

	_, err = factory.CreateMetricsToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalMetrics, pipeline.SignalTraces))
	_, err = factory.CreateMetricsToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalMetrics, pipeline.SignalLogs))

	_, err = factory.CreateLogsToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalLogs, pipeline.SignalTraces))
	_, err = factory.CreateLogsToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalLogs, pipeline.SignalMetrics))
}

func TestNewFactoryWithTranslateTypes(t *testing.T) {
	defaultCfg := struct{}{}
	factory := NewFactory(testType, func() component.Config { return &defaultCfg },
		WithTracesToMetrics(createTracesToMetrics, component.StabilityLevelDevelopment),
		WithTracesToLogs(createTracesToLogs, component.StabilityLevelAlpha),
		WithMetricsToTraces(createMetricsToTraces, component.StabilityLevelBeta),
		WithMetricsToLogs(createMetricsToLogs, component.StabilityLevelStable),
		WithLogsToTraces(createLogsToTraces, component.StabilityLevelDeprecated),
		WithLogsToMetrics(createLogsToMetrics, component.StabilityLevelUnmaintained))
	assert.Equal(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalTraces, pipeline.SignalTraces))
	_, err = factory.CreateMetricsToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalMetrics, pipeline.SignalMetrics))
	_, err = factory.CreateLogsToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, pipeline.SignalLogs, pipeline.SignalLogs))

	assert.Equal(t, component.StabilityLevelDevelopment, factory.TracesToMetricsStability())
	_, err = factory.CreateTracesToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToLogsStability())
	_, err = factory.CreateTracesToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToTracesStability())
	_, err = factory.CreateMetricsToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelStable, factory.MetricsToLogsStability())
	_, err = factory.CreateMetricsToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.LogsToTracesStability())
	_, err = factory.CreateLogsToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToMetricsStability())
	_, err = factory.CreateLogsToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.NoError(t, err)
}

func TestNewFactoryWithAllTypes(t *testing.T) {
	defaultCfg := struct{}{}
	factory := NewFactory(testType, func() component.Config { return &defaultCfg },
		WithTracesToTraces(createTracesToTraces, component.StabilityLevelAlpha),
		WithTracesToMetrics(createTracesToMetrics, component.StabilityLevelDevelopment),
		WithTracesToLogs(createTracesToLogs, component.StabilityLevelAlpha),
		WithMetricsToTraces(createMetricsToTraces, component.StabilityLevelBeta),
		WithMetricsToMetrics(createMetricsToMetrics, component.StabilityLevelBeta),
		WithMetricsToLogs(createMetricsToLogs, component.StabilityLevelStable),
		WithLogsToTraces(createLogsToTraces, component.StabilityLevelDeprecated),
		WithLogsToMetrics(createLogsToMetrics, component.StabilityLevelUnmaintained),
		WithLogsToLogs(createLogsToLogs, component.StabilityLevelUnmaintained))
	assert.Equal(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToTracesStability())
	_, err := factory.CreateTracesToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.Equal(t, component.StabilityLevelDevelopment, factory.TracesToMetricsStability())
	_, err = factory.CreateTracesToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToLogsStability())
	_, err = factory.CreateTracesToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToTracesStability())
	_, err = factory.CreateMetricsToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToMetricsStability())
	_, err = factory.CreateMetricsToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.Equal(t, component.StabilityLevelStable, factory.MetricsToLogsStability())
	_, err = factory.CreateMetricsToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.LogsToTracesStability())
	_, err = factory.CreateLogsToTraces(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToMetricsStability())
	_, err = factory.CreateLogsToMetrics(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToLogsStability())
	_, err = factory.CreateLogsToLogs(context.Background(), Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.NoError(t, err)
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

func createTracesToTraces(context.Context, Settings, component.Config, consumer.Traces) (Traces, error) {
	return nopInstance, nil
}

func createTracesToMetrics(context.Context, Settings, component.Config, consumer.Metrics) (Traces, error) {
	return nopInstance, nil
}

func createTracesToLogs(context.Context, Settings, component.Config, consumer.Logs) (Traces, error) {
	return nopInstance, nil
}

func createMetricsToTraces(context.Context, Settings, component.Config, consumer.Traces) (Metrics, error) {
	return nopInstance, nil
}

func createMetricsToMetrics(context.Context, Settings, component.Config, consumer.Metrics) (Metrics, error) {
	return nopInstance, nil
}

func createMetricsToLogs(context.Context, Settings, component.Config, consumer.Logs) (Metrics, error) {
	return nopInstance, nil
}

func createLogsToTraces(context.Context, Settings, component.Config, consumer.Traces) (Logs, error) {
	return nopInstance, nil
}

func createLogsToMetrics(context.Context, Settings, component.Config, consumer.Metrics) (Logs, error) {
	return nopInstance, nil
}

func createLogsToLogs(context.Context, Settings, component.Config, consumer.Logs) (Logs, error) {
	return nopInstance, nil
}
