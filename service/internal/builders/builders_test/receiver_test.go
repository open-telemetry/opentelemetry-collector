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
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/receiver/xreceiver"
	"go.opentelemetry.io/collector/service/internal/builders"
)

func TestReceiverBuilder(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := otelcol.MakeFactoryMap([]receiver.Factory{
		receiver.NewFactory(component.MustNewType("err"), nil),
		xreceiver.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			xreceiver.WithTraces(createReceiverTraces, component.StabilityLevelDevelopment),
			xreceiver.WithMetrics(createReceiverMetrics, component.StabilityLevelAlpha),
			xreceiver.WithLogs(createReceiverLogs, component.StabilityLevelDeprecated),
			xreceiver.WithProfiles(createReceiverProfiles, component.StabilityLevelAlpha),
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
		nextProfiles xconsumer.Profiles
	}{
		{
			name:         "unknown",
			id:           component.MustNewID("unknown"),
			err:          "receiver factory not available for: \"unknown\"",
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
			b := builders.NewReceiver(cfgs, factories)

			te, err := b.CreateTraces(context.Background(), settings(tt.id), tt.nextTraces)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, te)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopReceiverInstance, te)
			}

			me, err := b.CreateMetrics(context.Background(), settings(tt.id), tt.nextMetrics)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, me)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopReceiverInstance, me)
			}

			le, err := b.CreateLogs(context.Background(), settings(tt.id), tt.nextLogs)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, le)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopReceiverInstance, le)
			}

			pe, err := b.CreateProfiles(context.Background(), settings(tt.id), tt.nextProfiles)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, pe)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopReceiverInstance, pe)
			}
		})
	}
}

func TestReceiverBuilderMissingConfig(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := otelcol.MakeFactoryMap([]receiver.Factory{
		xreceiver.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			xreceiver.WithTraces(createReceiverTraces, component.StabilityLevelDevelopment),
			xreceiver.WithMetrics(createReceiverMetrics, component.StabilityLevelAlpha),
			xreceiver.WithLogs(createReceiverLogs, component.StabilityLevelDeprecated),
			xreceiver.WithProfiles(createReceiverProfiles, component.StabilityLevelAlpha),
		),
	}...)

	require.NoError(t, err)

	bErr := builders.NewReceiver(map[component.ID]component.Config{}, factories)
	missingID := component.MustNewIDWithName("all", "missing")

	te, err := bErr.CreateTraces(context.Background(), settings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, te)

	me, err := bErr.CreateMetrics(context.Background(), settings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, me)

	le, err := bErr.CreateLogs(context.Background(), settings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, le)

	pe, err := bErr.CreateProfiles(context.Background(), settings(missingID), consumertest.NewNop())
	require.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, pe)
}

func TestReceiverBuilderFactory(t *testing.T) {
	factories, err := otelcol.MakeFactoryMap([]receiver.Factory{receiver.NewFactory(component.MustNewType("foo"), nil)}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{component.MustNewID("foo"): struct{}{}}
	b := builders.NewReceiver(cfgs, factories)

	assert.NotNil(t, b.Factory(component.MustNewID("foo").Type()))
	assert.Nil(t, b.Factory(component.MustNewID("bar").Type()))
}

func TestNewNopReceiverConfigsAndFactories(t *testing.T) {
	configs, factories := builders.NewNopReceiverConfigsAndFactories()
	builder := builders.NewReceiver(configs, factories)
	require.NotNil(t, builder)

	factory := receivertest.NewNopFactory()
	cfg := factory.CreateDefaultConfig()
	set := receivertest.NewNopSettings(factory.Type())
	set.ID = component.NewID(builders.NopType)

	traces, err := factory.CreateTraces(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bTraces, err := builder.CreateTraces(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, traces, bTraces)

	metrics, err := factory.CreateMetrics(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bMetrics, err := builder.CreateMetrics(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, metrics, bMetrics)

	logs, err := factory.CreateLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bLogs, err := builder.CreateLogs(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, logs, bLogs)

	profiles, err := factory.(xreceiver.Factory).CreateProfiles(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bProfiles, err := builder.CreateProfiles(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, profiles, bProfiles)
}

func settings(id component.ID) receiver.Settings {
	return receiver.Settings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

var nopReceiverInstance = &nopReceiver{
	Consumer: consumertest.NewNop(),
}

// nopReceiver stores consumed traces and metrics for testing purposes.
type nopReceiver struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createReceiverTraces(context.Context, receiver.Settings, component.Config, consumer.Traces) (receiver.Traces, error) {
	return nopReceiverInstance, nil
}

func createReceiverMetrics(context.Context, receiver.Settings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
	return nopReceiverInstance, nil
}

func createReceiverLogs(context.Context, receiver.Settings, component.Config, consumer.Logs) (receiver.Logs, error) {
	return nopReceiverInstance, nil
}

func createReceiverProfiles(context.Context, receiver.Settings, component.Config, xconsumer.Profiles) (xreceiver.Profiles, error) {
	return nopReceiverInstance, nil
}
