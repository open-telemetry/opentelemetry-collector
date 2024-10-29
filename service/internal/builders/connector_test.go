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
	"go.opentelemetry.io/collector/connector/connectorprofiles"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/pipelineprofiles"
)

func TestConnectorBuilder(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := connector.MakeFactoryMap([]connector.Factory{
		connector.NewFactory(component.MustNewType("err"), nil),
		connectorprofiles.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			connectorprofiles.WithTracesToTraces(createConnectorTracesToTraces, component.StabilityLevelDevelopment),
			connectorprofiles.WithTracesToMetrics(createConnectorTracesToMetrics, component.StabilityLevelDevelopment),
			connectorprofiles.WithTracesToLogs(createConnectorTracesToLogs, component.StabilityLevelDevelopment),
			connectorprofiles.WithTracesToProfiles(createConnectorTracesToProfiles, component.StabilityLevelDevelopment),
			connectorprofiles.WithMetricsToTraces(createConnectorMetricsToTraces, component.StabilityLevelAlpha),
			connectorprofiles.WithMetricsToMetrics(createConnectorMetricsToMetrics, component.StabilityLevelAlpha),
			connectorprofiles.WithMetricsToLogs(createConnectorMetricsToLogs, component.StabilityLevelAlpha),
			connectorprofiles.WithMetricsToProfiles(createConnectorMetricsToProfiles, component.StabilityLevelAlpha),
			connectorprofiles.WithLogsToTraces(createConnectorLogsToTraces, component.StabilityLevelDeprecated),
			connectorprofiles.WithLogsToMetrics(createConnectorLogsToMetrics, component.StabilityLevelDeprecated),
			connectorprofiles.WithLogsToLogs(createConnectorLogsToLogs, component.StabilityLevelDeprecated),
			connectorprofiles.WithLogsToProfiles(createConnectorLogsToProfiles, component.StabilityLevelDeprecated),
			connectorprofiles.WithProfilesToTraces(createConnectorProfilesToTraces, component.StabilityLevelDevelopment),
			connectorprofiles.WithProfilesToMetrics(createConnectorProfilesToMetrics, component.StabilityLevelDevelopment),
			connectorprofiles.WithProfilesToLogs(createConnectorProfilesToLogs, component.StabilityLevelDevelopment),
			connectorprofiles.WithProfilesToProfiles(createConnectorProfilesToProfiles, component.StabilityLevelDevelopment),
		),
	}...)
	require.NoError(t, err)

	testCases := []struct {
		name         string
		id           component.ID
		err          func(pipeline.Signal, pipeline.Signal) string
		nextTraces   consumer.Traces
		nextLogs     consumer.Logs
		nextMetrics  consumer.Metrics
		nextProfiles consumerprofiles.Profiles
	}{
		{
			name: "unknown",
			id:   component.MustNewID("unknown"),
			err: func(pipeline.Signal, pipeline.Signal) string {
				return "connector factory not available for: \"unknown\""
			},
			nextTraces:   consumertest.NewNop(),
			nextLogs:     consumertest.NewNop(),
			nextMetrics:  consumertest.NewNop(),
			nextProfiles: consumertest.NewNop(),
		},
		{
			name: "err",
			id:   component.MustNewID("err"),
			err: func(expType, rcvType pipeline.Signal) string {
				return fmt.Sprintf("connector \"err\" cannot connect from %s to %s: telemetry type is not supported", expType, rcvType)
			},
			nextTraces:   consumertest.NewNop(),
			nextLogs:     consumertest.NewNop(),
			nextMetrics:  consumertest.NewNop(),
			nextProfiles: consumertest.NewNop(),
		},
		{
			name: "all",
			id:   component.MustNewID("all"),
			err: func(pipeline.Signal, pipeline.Signal) string {
				return ""
			},
			nextTraces:   consumertest.NewNop(),
			nextLogs:     consumertest.NewNop(),
			nextMetrics:  consumertest.NewNop(),
			nextProfiles: consumertest.NewNop(),
		},
		{
			name: "all/named",
			id:   component.MustNewIDWithName("all", "named"),
			err: func(pipeline.Signal, pipeline.Signal) string {
				return ""
			},
			nextTraces:   consumertest.NewNop(),
			nextLogs:     consumertest.NewNop(),
			nextMetrics:  consumertest.NewNop(),
			nextProfiles: consumertest.NewNop(),
		},
		{
			name: "no next consumer",
			id:   component.MustNewID("unknown"),
			err: func(_, _ pipeline.Signal) string {
				return "nil next Consumer"
			},
			nextTraces:   nil,
			nextLogs:     nil,
			nextMetrics:  nil,
			nextProfiles: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfgs := map[component.ID]component.Config{tt.id: defaultCfg}
			b := NewConnector(cfgs, factories)

			t2t, err := b.CreateTracesToTraces(context.Background(), createConnectorSettings(tt.id), tt.nextTraces)
			if expectedErr := tt.err(pipeline.SignalTraces, pipeline.SignalTraces); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, t2t)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, t2t)
			}
			t2m, err := b.CreateTracesToMetrics(context.Background(), createConnectorSettings(tt.id), tt.nextMetrics)
			if expectedErr := tt.err(pipeline.SignalTraces, pipeline.SignalMetrics); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, t2m)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, t2m)
			}
			t2l, err := b.CreateTracesToLogs(context.Background(), createConnectorSettings(tt.id), tt.nextLogs)
			if expectedErr := tt.err(pipeline.SignalTraces, pipeline.SignalLogs); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, t2l)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, t2l)
			}
			t2p, err := b.CreateTracesToProfiles(context.Background(), createConnectorSettings(tt.id), tt.nextProfiles)
			if expectedErr := tt.err(pipeline.SignalTraces, pipelineprofiles.SignalProfiles); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, t2p)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, t2p)
			}

			m2t, err := b.CreateMetricsToTraces(context.Background(), createConnectorSettings(tt.id), tt.nextTraces)
			if expectedErr := tt.err(pipeline.SignalMetrics, pipeline.SignalTraces); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, m2t)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, m2t)
			}

			m2m, err := b.CreateMetricsToMetrics(context.Background(), createConnectorSettings(tt.id), tt.nextMetrics)
			if expectedErr := tt.err(pipeline.SignalMetrics, pipeline.SignalMetrics); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, m2m)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, m2m)
			}

			m2l, err := b.CreateMetricsToLogs(context.Background(), createConnectorSettings(tt.id), tt.nextLogs)
			if expectedErr := tt.err(pipeline.SignalMetrics, pipeline.SignalLogs); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, m2l)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, m2l)
			}
			m2p, err := b.CreateMetricsToProfiles(context.Background(), createConnectorSettings(tt.id), tt.nextProfiles)
			if expectedErr := tt.err(pipeline.SignalMetrics, pipelineprofiles.SignalProfiles); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, m2p)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, m2p)
			}

			l2t, err := b.CreateLogsToTraces(context.Background(), createConnectorSettings(tt.id), tt.nextTraces)
			if expectedErr := tt.err(pipeline.SignalLogs, pipeline.SignalTraces); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, l2t)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, l2t)
			}

			l2m, err := b.CreateLogsToMetrics(context.Background(), createConnectorSettings(tt.id), tt.nextMetrics)
			if expectedErr := tt.err(pipeline.SignalLogs, pipeline.SignalMetrics); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, l2m)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, l2m)
			}

			l2l, err := b.CreateLogsToLogs(context.Background(), createConnectorSettings(tt.id), tt.nextLogs)
			if expectedErr := tt.err(pipeline.SignalLogs, pipeline.SignalLogs); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, l2l)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, l2l)
			}
			l2p, err := b.CreateLogsToProfiles(context.Background(), createConnectorSettings(tt.id), tt.nextProfiles)
			if expectedErr := tt.err(pipeline.SignalLogs, pipelineprofiles.SignalProfiles); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, l2p)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, l2p)
			}

			p2t, err := b.CreateProfilesToTraces(context.Background(), createConnectorSettings(tt.id), tt.nextTraces)
			if expectedErr := tt.err(pipelineprofiles.SignalProfiles, pipeline.SignalTraces); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, p2t)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, p2t)
			}
			p2m, err := b.CreateProfilesToMetrics(context.Background(), createConnectorSettings(tt.id), tt.nextMetrics)
			if expectedErr := tt.err(pipelineprofiles.SignalProfiles, pipeline.SignalMetrics); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, p2m)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, p2m)
			}
			p2l, err := b.CreateProfilesToLogs(context.Background(), createConnectorSettings(tt.id), tt.nextLogs)
			if expectedErr := tt.err(pipelineprofiles.SignalProfiles, pipeline.SignalLogs); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, p2l)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, p2l)
			}
			p2p, err := b.CreateProfilesToProfiles(context.Background(), createConnectorSettings(tt.id), tt.nextProfiles)
			if expectedErr := tt.err(pipelineprofiles.SignalProfiles, pipelineprofiles.SignalProfiles); expectedErr != "" {
				assert.EqualError(t, err, expectedErr)
				assert.Nil(t, p2p)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopConnectorInstance, p2p)
			}
		})
	}
}

func TestConnectorBuilderMissingConfig(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := connector.MakeFactoryMap([]connector.Factory{
		connectorprofiles.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			connectorprofiles.WithTracesToTraces(createConnectorTracesToTraces, component.StabilityLevelDevelopment),
			connectorprofiles.WithTracesToMetrics(createConnectorTracesToMetrics, component.StabilityLevelDevelopment),
			connectorprofiles.WithTracesToLogs(createConnectorTracesToLogs, component.StabilityLevelDevelopment),
			connectorprofiles.WithTracesToProfiles(createConnectorTracesToProfiles, component.StabilityLevelDevelopment),
			connectorprofiles.WithMetricsToTraces(createConnectorMetricsToTraces, component.StabilityLevelAlpha),
			connectorprofiles.WithMetricsToMetrics(createConnectorMetricsToMetrics, component.StabilityLevelAlpha),
			connectorprofiles.WithMetricsToLogs(createConnectorMetricsToLogs, component.StabilityLevelAlpha),
			connectorprofiles.WithMetricsToProfiles(createConnectorMetricsToProfiles, component.StabilityLevelAlpha),
			connectorprofiles.WithLogsToTraces(createConnectorLogsToTraces, component.StabilityLevelDeprecated),
			connectorprofiles.WithLogsToMetrics(createConnectorLogsToMetrics, component.StabilityLevelDeprecated),
			connectorprofiles.WithLogsToLogs(createConnectorLogsToLogs, component.StabilityLevelDeprecated),
			connectorprofiles.WithLogsToProfiles(createConnectorLogsToProfiles, component.StabilityLevelDeprecated),
			connectorprofiles.WithProfilesToTraces(createConnectorProfilesToTraces, component.StabilityLevelDevelopment),
			connectorprofiles.WithProfilesToMetrics(createConnectorProfilesToMetrics, component.StabilityLevelDevelopment),
			connectorprofiles.WithProfilesToLogs(createConnectorProfilesToLogs, component.StabilityLevelDevelopment),
			connectorprofiles.WithProfilesToProfiles(createConnectorProfilesToProfiles, component.StabilityLevelDevelopment),
		),
	}...)

	require.NoError(t, err)

	bErr := NewConnector(map[component.ID]component.Config{}, factories)
	missingID := component.MustNewIDWithName("all", "missing")

	t2t, err := bErr.CreateTracesToTraces(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, t2t)

	t2m, err := bErr.CreateTracesToMetrics(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, t2m)

	t2l, err := bErr.CreateTracesToLogs(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, t2l)

	t2p, err := bErr.CreateTracesToProfiles(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, t2p)

	m2t, err := bErr.CreateMetricsToTraces(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, m2t)

	m2m, err := bErr.CreateMetricsToMetrics(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, m2m)

	m2l, err := bErr.CreateMetricsToLogs(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, m2l)

	m2p, err := bErr.CreateMetricsToProfiles(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, m2p)

	l2t, err := bErr.CreateLogsToTraces(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, l2t)

	l2m, err := bErr.CreateLogsToMetrics(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, l2m)

	l2l, err := bErr.CreateLogsToLogs(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, l2l)

	l2p, err := bErr.CreateLogsToProfiles(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, l2p)

	p2t, err := bErr.CreateProfilesToTraces(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, p2t)

	p2m, err := bErr.CreateProfilesToMetrics(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, p2m)

	p2l, err := bErr.CreateProfilesToLogs(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, p2l)

	p2p, err := bErr.CreateProfilesToProfiles(context.Background(), createConnectorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "connector \"all/missing\" is not configured")
	assert.Nil(t, p2p)
}

func TestConnectorBuilderGetters(t *testing.T) {
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

	tracesToProfiles, err := factory.(connectorprofiles.Factory).CreateTracesToProfiles(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bTracesToProfiles, err := builder.CreateTracesToProfiles(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, tracesToProfiles, bTracesToProfiles)

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

	metricsToProfiles, err := factory.(connectorprofiles.Factory).CreateMetricsToProfiles(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bMetricsToProfiles, err := builder.CreateMetricsToProfiles(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, metricsToProfiles, bMetricsToProfiles)

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

	logsToProfiles, err := factory.(connectorprofiles.Factory).CreateLogsToProfiles(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bLogsToProfiles, err := builder.CreateLogsToProfiles(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, logsToProfiles, bLogsToProfiles)

	profilesToTraces, err := factory.(connectorprofiles.Factory).CreateProfilesToTraces(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bProfilesToTraces, err := builder.CreateProfilesToTraces(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, profilesToTraces, bProfilesToTraces)

	profilesToMetrics, err := factory.(connectorprofiles.Factory).CreateProfilesToMetrics(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bProfilesToMetrics, err := builder.CreateProfilesToMetrics(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, profilesToMetrics, bProfilesToMetrics)

	profilesToLogs, err := factory.(connectorprofiles.Factory).CreateProfilesToLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bProfilesToLogs, err := builder.CreateProfilesToLogs(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, profilesToLogs, bProfilesToLogs)

	profilesToProfiles, err := factory.(connectorprofiles.Factory).CreateProfilesToProfiles(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bProfilesToProfiles, err := builder.CreateProfilesToProfiles(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, profilesToProfiles, bProfilesToProfiles)
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
func createConnectorTracesToProfiles(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Traces, error) {
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
func createConnectorMetricsToProfiles(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Metrics, error) {
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
func createConnectorLogsToProfiles(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Logs, error) {
	return nopConnectorInstance, nil
}

func createConnectorProfilesToTraces(context.Context, connector.Settings, component.Config, consumer.Traces) (connectorprofiles.Profiles, error) {
	return nopConnectorInstance, nil
}
func createConnectorProfilesToMetrics(context.Context, connector.Settings, component.Config, consumer.Metrics) (connectorprofiles.Profiles, error) {
	return nopConnectorInstance, nil
}
func createConnectorProfilesToLogs(context.Context, connector.Settings, component.Config, consumer.Logs) (connectorprofiles.Profiles, error) {
	return nopConnectorInstance, nil
}
func createConnectorProfilesToProfiles(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connectorprofiles.Profiles, error) {
	return nopConnectorInstance, nil
}

func createConnectorSettings(id component.ID) connector.Settings {
	return connector.Settings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
