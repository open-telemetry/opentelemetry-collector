// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builders

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

func TestBuilder(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := connector.MakeFactoryMap([]connector.Factory{
		connector.NewFactory(component.MustNewType("err"), nil),
		connector.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			connector.WithTracesToTraces(createConnectorTracesToTraces, component.StabilityLevelDevelopment),
			connector.WithTracesToMetrics(createConnectorTracesToMetrics, component.StabilityLevelDevelopment),
			connector.WithTracesToLogs(createConnectorTracesToLogs, component.StabilityLevelDevelopment),
			connector.WithMetricsToTraces(createConnectorMetricsToTraces, component.StabilityLevelAlpha),
			connector.WithMetricsToMetrics(createConnectorMetricsToMetrics, component.StabilityLevelAlpha),
			connector.WithMetricsToLogs(createConnectorMetricsToLogs, component.StabilityLevelAlpha),
			connector.WithLogsToTraces(createConnectorLogsToTraces, component.StabilityLevelDeprecated),
			connector.WithLogsToMetrics(createConnectorLogsToMetrics, component.StabilityLevelDeprecated),
			connector.WithLogsToLogs(createConnectorLogsToLogs, component.StabilityLevelDeprecated),
		),
	}...)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		id          component.ID
		err         func(component.DataType, component.DataType) string
		nextTraces  consumer.Traces
		nextLogs    consumer.Logs
		nextMetrics consumer.Metrics
	}{
		{
			name: "unknown",
			id:   component.MustNewID("unknown"),
			err: func(component.DataType, component.DataType) string {
				return "connector factory not available for: \"unknown\""
			},
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),
		},
		{
			name: "err",
			id:   component.MustNewID("err"),
			err: func(expType, rcvType component.DataType) string {
				return fmt.Sprintf("connector \"err\" cannot connect from %s to %s: telemetry type is not supported", expType, rcvType)
			},
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),
		},
		{
			name: "all",
			id:   component.MustNewID("all"),
			err: func(component.DataType, component.DataType) string {
				return ""
			},
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),
		},
		{
			name: "all/named",
			id:   component.MustNewIDWithName("all", "named"),
			err: func(component.DataType, component.DataType) string {
				return ""
			},
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),
		},
		{
			name: "no next consumer",
			id:   component.MustNewID("unknown"),
			err: func(_, _ component.DataType) string {
				return "nil next Consumer"
			},
			nextTraces:  nil,
			nextLogs:    nil,
			nextMetrics: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfgs := map[component.ID]component.Config{tt.id: defaultCfg}
			b := NewConnector(cfgs, factories)

			t2t, err := b.CreateTracesToTraces(context.Background(), createConnectorSettings(tt.id), tt.nextTraces)
			if expectedErr := tt.err(component.DataTypeTraces, component.DataTypeTraces); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, t2t)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, t2t)
			}
			t2m, err := b.CreateTracesToMetrics(context.Background(), createConnectorSettings(tt.id), tt.nextMetrics)
			if expectedErr := tt.err(component.DataTypeTraces, component.DataTypeMetrics); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, t2m)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, t2m)
			}
			t2l, err := b.CreateTracesToLogs(context.Background(), createConnectorSettings(tt.id), tt.nextLogs)
			if expectedErr := tt.err(component.DataTypeTraces, component.DataTypeLogs); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, t2l)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, t2l)
			}

			m2t, err := b.CreateMetricsToTraces(context.Background(), createConnectorSettings(tt.id), tt.nextTraces)
			if expectedErr := tt.err(component.DataTypeMetrics, component.DataTypeTraces); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, m2t)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, m2t)
			}

			m2m, err := b.CreateMetricsToMetrics(context.Background(), createConnectorSettings(tt.id), tt.nextMetrics)
			if expectedErr := tt.err(component.DataTypeMetrics, component.DataTypeMetrics); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, m2m)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, m2m)
			}

			m2l, err := b.CreateMetricsToLogs(context.Background(), createConnectorSettings(tt.id), tt.nextLogs)
			if expectedErr := tt.err(component.DataTypeMetrics, component.DataTypeLogs); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, m2l)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, m2l)
			}

			l2t, err := b.CreateLogsToTraces(context.Background(), createConnectorSettings(tt.id), tt.nextTraces)
			if expectedErr := tt.err(component.DataTypeLogs, component.DataTypeTraces); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, l2t)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, l2t)
			}

			l2m, err := b.CreateLogsToMetrics(context.Background(), createConnectorSettings(tt.id), tt.nextMetrics)
			if expectedErr := tt.err(component.DataTypeLogs, component.DataTypeMetrics); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, l2m)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, l2m)
			}

			l2l, err := b.CreateLogsToLogs(context.Background(), createConnectorSettings(tt.id), tt.nextLogs)
			if expectedErr := tt.err(component.DataTypeLogs, component.DataTypeLogs); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, l2l)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, l2l)
			}
		})
	}
}

func TestBuilderMissingConfig(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := connector.MakeFactoryMap([]connector.Factory{
		connector.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			connector.WithTracesToTraces(createConnectorTracesToTraces, component.StabilityLevelDevelopment),
			connector.WithTracesToMetrics(createConnectorTracesToMetrics, component.StabilityLevelDevelopment),
			connector.WithTracesToLogs(createConnectorTracesToLogs, component.StabilityLevelDevelopment),
			connector.WithMetricsToTraces(createConnectorMetricsToTraces, component.StabilityLevelAlpha),
			connector.WithMetricsToMetrics(createConnectorMetricsToMetrics, component.StabilityLevelAlpha),
			connector.WithMetricsToLogs(createConnectorMetricsToLogs, component.StabilityLevelAlpha),
			connector.WithLogsToTraces(createConnectorLogsToTraces, component.StabilityLevelDeprecated),
			connector.WithLogsToMetrics(createConnectorLogsToMetrics, component.StabilityLevelDeprecated),
			connector.WithLogsToLogs(createConnectorLogsToLogs, component.StabilityLevelDeprecated),
		),
	}...)

	require.NoError(t, err)

	bErr := NewConnector(map[component.ID]component.Config{}, factories)
	missingID := component.MustNewIDWithName("all", "missing")

	t2t, err := bErr.CreateTracesToTraces(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, t2t)

	t2m, err := bErr.CreateTracesToMetrics(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, t2m)

	t2l, err := bErr.CreateTracesToLogs(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, t2l)

	m2t, err := bErr.CreateMetricsToTraces(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, m2t)

	m2m, err := bErr.CreateMetricsToMetrics(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, m2m)

	m2l, err := bErr.CreateMetricsToLogs(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, m2l)

	l2t, err := bErr.CreateLogsToTraces(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, l2t)

	l2m, err := bErr.CreateLogsToMetrics(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, l2m)

	l2l, err := bErr.CreateLogsToLogs(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, l2l)
}

func TestBuilderGetters(t *testing.T) {
	factories, err := connector.MakeFactoryMap([]connector.Factory{connector.NewFactory(component.MustNewType("foo"), nil)}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{component.MustNewID("foo"): struct{}{}}
	b := NewConnector(cfgs, factories)

	assert.True(t, b.IsConfigured(component.MustNewID("foo")))
	assert.False(t, b.IsConfigured(component.MustNewID("bar")))

	assert.NotNil(t, b.Factory(component.MustNewID("foo").Type()))
	assert.Nil(t, b.Factory(component.MustNewID("bar").Type()))
}

func TestNewNopConnectorConfigsAndFactories(t *testing.T) {
	configs, factories := NewNopConnectorConfigsAndFactories()
	builder := NewConnector(configs, factories)
	require.NotNil(t, builder)

	factory := connectortest.NewNopFactory()
	cfg := factory.CreateDefaultConfig()
	set := connectortest.NewNopSettings()
	set.ID = component.NewIDWithName(nopType, "conn")

	tracesToTraces, err := factory.CreateTracesToTraces(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bTracesToTraces, err := builder.CreateTracesToTraces(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, tracesToTraces, bTracesToTraces)

	tracesToMetrics, err := factory.CreateTracesToMetrics(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bTracesToMetrics, err := builder.CreateTracesToMetrics(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, tracesToMetrics, bTracesToMetrics)

	tracesToLogs, err := factory.CreateTracesToLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bTracesToLogs, err := builder.CreateTracesToLogs(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, tracesToLogs, bTracesToLogs)

	metricsToTraces, err := factory.CreateMetricsToTraces(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bMetricsToTraces, err := builder.CreateMetricsToTraces(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, metricsToTraces, bMetricsToTraces)

	metricsToMetrics, err := factory.CreateMetricsToMetrics(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bMetricsToMetrics, err := builder.CreateMetricsToMetrics(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, metricsToMetrics, bMetricsToMetrics)

	metricsToLogs, err := factory.CreateMetricsToLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bMetricsToLogs, err := builder.CreateMetricsToLogs(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, metricsToLogs, bMetricsToLogs)

	logsToTraces, err := factory.CreateLogsToTraces(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bLogsToTraces, err := builder.CreateLogsToTraces(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, logsToTraces, bLogsToTraces)

	logsToMetrics, err := factory.CreateLogsToMetrics(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bLogsToMetrics, err := builder.CreateLogsToMetrics(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, logsToMetrics, bLogsToMetrics)

	logsToLogs, err := factory.CreateLogsToLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bLogsToLogs, err := builder.CreateLogsToLogs(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, logsToLogs, bLogsToLogs)
}

var nopConnectorInstance = &nopConnector{
	Consumer: consumertest.NewNop(),
}

// nopConnector stores consumed traces and metrics for testing purposes.
type nopConnector struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createConnectorTracesToTraces(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Traces, error) {
	return nopConnectorInstance, nil
}
func createConnectorTracesToMetrics(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Traces, error) {
	return nopConnectorInstance, nil
}
func createConnectorTracesToLogs(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Traces, error) {
	return nopConnectorInstance, nil
}

func createConnectorMetricsToTraces(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Metrics, error) {
	return nopConnectorInstance, nil
}
func createConnectorMetricsToMetrics(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Metrics, error) {
	return nopConnectorInstance, nil
}
func createConnectorMetricsToLogs(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Metrics, error) {
	return nopConnectorInstance, nil
}

func createConnectorLogsToTraces(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Logs, error) {
	return nopConnectorInstance, nil
}
func createConnectorLogsToMetrics(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Logs, error) {
	return nopConnectorInstance, nil
}
func createConnectorLogsToLogs(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Logs, error) {
	return nopConnectorInstance, nil
}

func createConnectorSettings(id component.ID) connector.Settings {
	return connector.Settings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
