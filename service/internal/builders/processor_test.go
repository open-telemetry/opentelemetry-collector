// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builders

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorprofiles"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestProcessorBuilder(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := processor.MakeFactoryMap([]processor.Factory{
		processor.NewFactory(component.MustNewType("err"), nil),
		processor.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			processor.WithTraces(createProcessorTraces, component.StabilityLevelDevelopment),
			processor.WithMetrics(createProcessorMetrics, component.StabilityLevelAlpha),
			processor.WithLogs(createProcessorLogs, component.StabilityLevelDeprecated),
			processorprofiles.WithProfiles(createProcessorProfiles, component.StabilityLevelDevelopment),
		),
	}...)
	require.NoError(t, err)

	testCases := []struct {
		name         string
		id           component.ID
		err          string
		nextTraces   consumer.Traces
		nextLogs     consumer.Logs
		nextMetrics  consumer.Metrics
		nextProfiles consumerprofiles.Profiles
	}{
		{
			name:         "unknown",
			id:           component.MustNewID("unknown"),
			err:          "processor factory not available for: \"unknown\"",
			nextTraces:   consumertest.NewNop(),
			nextLogs:     consumertest.NewNop(),
			nextMetrics:  consumertest.NewNop(),
			nextProfiles: consumertest.NewNop(),
		},
		{
			name:         "err",
			id:           component.MustNewID("err"),
			err:          "telemetry type is not supported",
			nextTraces:   consumertest.NewNop(),
			nextLogs:     consumertest.NewNop(),
			nextMetrics:  consumertest.NewNop(),
			nextProfiles: consumertest.NewNop(),
		},
		{
			name:         "all",
			id:           component.MustNewID("all"),
			nextTraces:   consumertest.NewNop(),
			nextLogs:     consumertest.NewNop(),
			nextMetrics:  consumertest.NewNop(),
			nextProfiles: consumertest.NewNop(),
		},
		{
			name:         "all/named",
			id:           component.MustNewIDWithName("all", "named"),
			nextTraces:   consumertest.NewNop(),
			nextLogs:     consumertest.NewNop(),
			nextMetrics:  consumertest.NewNop(),
			nextProfiles: consumertest.NewNop(),
		},
		{
			name:         "no next consumer",
			id:           component.MustNewID("unknown"),
			err:          "nil next Consumer",
			nextTraces:   nil,
			nextLogs:     nil,
			nextMetrics:  nil,
			nextProfiles: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfgs := map[component.ID]component.Config{tt.id: defaultCfg}
			b := NewProcessor(cfgs, factories)

			te, err := b.CreateTraces(context.Background(), createProcessorSettings(tt.id), tt.nextTraces)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, te)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopProcessorInstance, te)
			}

			me, err := b.CreateMetrics(context.Background(), createProcessorSettings(tt.id), tt.nextMetrics)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, me)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopProcessorInstance, me)
			}

			le, err := b.CreateLogs(context.Background(), createProcessorSettings(tt.id), tt.nextLogs)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, le)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopProcessorInstance, le)
			}

			pe, err := b.CreateProfiles(context.Background(), createProcessorSettings(tt.id), tt.nextProfiles)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, pe)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopProcessorInstance, pe)
			}
		})
	}
}

func TestProcessorBuilderMissingConfig(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := processor.MakeFactoryMap([]processor.Factory{
		processor.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			processor.WithTraces(createProcessorTraces, component.StabilityLevelDevelopment),
			processor.WithMetrics(createProcessorMetrics, component.StabilityLevelAlpha),
			processor.WithLogs(createProcessorLogs, component.StabilityLevelDeprecated),
			processorprofiles.WithProfiles(createProcessorProfiles, component.StabilityLevelDevelopment),
		),
	}...)

	require.NoError(t, err)

	bErr := NewProcessor(map[component.ID]component.Config{}, factories)
	missingID := component.MustNewIDWithName("all", "missing")

	te, err := bErr.CreateTraces(context.Background(), createProcessorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "processor \"all/missing\" is not configured")
	assert.Nil(t, te)

	me, err := bErr.CreateMetrics(context.Background(), createProcessorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "processor \"all/missing\" is not configured")
	assert.Nil(t, me)

	le, err := bErr.CreateLogs(context.Background(), createProcessorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "processor \"all/missing\" is not configured")
	assert.Nil(t, le)

	pe, err := bErr.CreateProfiles(context.Background(), createProcessorSettings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "processor \"all/missing\" is not configured")
	assert.Nil(t, pe)
}

func TestProcessorBuilderFactory(t *testing.T) {
	factories, err := processor.MakeFactoryMap([]processor.Factory{processor.NewFactory(component.MustNewType("foo"), nil)}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{component.MustNewID("foo"): struct{}{}}
	b := NewProcessor(cfgs, factories)

	assert.NotNil(t, b.Factory(component.MustNewID("foo").Type()))
	assert.Nil(t, b.Factory(component.MustNewID("bar").Type()))
}

func TestNewNopProcessorBuilder(t *testing.T) {
	configs, factories := NewNopProcessorConfigsAndFactories()
	builder := NewProcessor(configs, factories)
	require.NotNil(t, builder)

	factory := processortest.NewNopFactory()
	cfg := factory.CreateDefaultConfig()
	set := processortest.NewNopSettings()
	set.ID = component.NewID(nopType)

	traces, err := factory.CreateTracesProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bTraces, err := builder.CreateTraces(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, traces, bTraces)

	metrics, err := factory.CreateMetricsProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bMetrics, err := builder.CreateMetrics(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, metrics, bMetrics)

	logs, err := factory.CreateLogsProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bLogs, err := builder.CreateLogs(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, logs, bLogs)

	profiles, err := factory.CreateProfilesProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bProfiles, err := builder.CreateProfiles(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, profiles, bProfiles)
}

var nopProcessorInstance = &nopProcessor{
	Consumer: consumertest.NewNop(),
}

// nopProcessor stores consumed traces, metrics, logs and profiles for testing purposes.
type nopProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createProcessorTraces(context.Context, processor.Settings, component.Config, consumer.Traces) (processor.Traces, error) {
	return nopProcessorInstance, nil
}

func createProcessorMetrics(context.Context, processor.Settings, component.Config, consumer.Metrics) (processor.Metrics, error) {
	return nopProcessorInstance, nil
}

func createProcessorLogs(context.Context, processor.Settings, component.Config, consumer.Logs) (processor.Logs, error) {
	return nopProcessorInstance, nil
}

func createProcessorProfiles(context.Context, processor.Settings, component.Config, consumerprofiles.Profiles) (processorprofiles.Profiles, error) {
	return nopProcessorInstance, nil
}

func createProcessorSettings(id component.ID) processor.Settings {
	return processor.Settings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
