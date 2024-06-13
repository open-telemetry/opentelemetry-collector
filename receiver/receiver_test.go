// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumerexperimental"
	"go.opentelemetry.io/collector/consumerexperimental/consumerexperimentaltest"
)

func TestNewFactory(t *testing.T) {
	var testType = component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg })
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesReceiver(context.Background(), Settings{}, &defaultCfg, consumertest.NewNop())
	assert.Error(t, err)
	_, err = factory.CreateMetricsReceiver(context.Background(), Settings{}, &defaultCfg, consumertest.NewNop())
	assert.Error(t, err)
	_, err = factory.CreateLogsReceiver(context.Background(), Settings{}, &defaultCfg, consumertest.NewNop())
	assert.Error(t, err)

	_, err = factory.CreateProfilesReceiver(context.Background(), Settings{}, &defaultCfg, consumerexperimentaltest.NewNop())
	assert.Error(t, err)
}

func TestNewFactoryWithOptions(t *testing.T) {
	var testType = component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithTraces(createTraces, component.StabilityLevelDeprecated),
		WithMetrics(createMetrics, component.StabilityLevelAlpha),
		WithLogs(createLogs, component.StabilityLevelStable),

		WithProfiles(createProfiles, component.StabilityLevelDeprecated),
	)
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDeprecated, factory.TracesReceiverStability())
	_, err := factory.CreateTracesReceiver(context.Background(), Settings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.MetricsReceiverStability())
	_, err = factory.CreateMetricsReceiver(context.Background(), Settings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelStable, factory.LogsReceiverStability())
	_, err = factory.CreateLogsReceiver(context.Background(), Settings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.ProfilesReceiverStability())
	_, err = factory.CreateProfilesReceiver(context.Background(), Settings{}, &defaultCfg, nil)
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

	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestBuilder(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := MakeFactoryMap([]Factory{
		NewFactory(component.MustNewType("err"), nil),
		NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			WithTraces(createTraces, component.StabilityLevelDevelopment),
			WithMetrics(createMetrics, component.StabilityLevelAlpha),
			WithLogs(createLogs, component.StabilityLevelDeprecated),

			WithProfiles(createProfiles, component.StabilityLevelDevelopment),
		),
	}...)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		id          component.ID
		err         string
		nextTraces  consumer.Traces
		nextLogs    consumer.Logs
		nextMetrics consumer.Metrics

		nextProfiles consumerexperimental.Profiles
	}{
		{
			name:        "unknown",
			id:          component.MustNewID("unknown"),
			err:         "receiver factory not available for: \"unknown\"",
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),

			nextProfiles: consumerexperimentaltest.NewNop(),
		},
		{
			name:        "err",
			id:          component.MustNewID("err"),
			err:         "telemetry type is not supported",
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),

			nextProfiles: consumerexperimentaltest.NewNop(),
		},
		{
			name:        "all",
			id:          component.MustNewID("all"),
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),

			nextProfiles: consumerexperimentaltest.NewNop(),
		},
		{
			name:        "all/named",
			id:          component.MustNewIDWithName("all", "named"),
			nextTraces:  consumertest.NewNop(),
			nextLogs:    consumertest.NewNop(),
			nextMetrics: consumertest.NewNop(),

			nextProfiles: consumerexperimentaltest.NewNop(),
		},
		{
			name:        "no next consumer",
			id:          component.MustNewID("unknown"),
			err:         "nil next Consumer",
			nextTraces:  nil,
			nextLogs:    nil,
			nextMetrics: nil,

			nextProfiles: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfgs := map[component.ID]component.Config{tt.id: defaultCfg}
			b := NewBuilder(cfgs, factories)

			te, err := b.CreateTraces(context.Background(), settings(tt.id), tt.nextTraces)
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, te)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, te)
			}

			me, err := b.CreateMetrics(context.Background(), settings(tt.id), tt.nextMetrics)
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, me)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, me)
			}

			le, err := b.CreateLogs(context.Background(), settings(tt.id), tt.nextLogs)
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, le)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, le)
			}

			pe, err := b.CreateProfiles(context.Background(), settings(tt.id), tt.nextProfiles)
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, pe)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, pe)
			}
		})
	}
}

func TestBuilderMissingConfig(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := MakeFactoryMap([]Factory{
		NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			WithTraces(createTraces, component.StabilityLevelDevelopment),
			WithMetrics(createMetrics, component.StabilityLevelAlpha),
			WithLogs(createLogs, component.StabilityLevelDeprecated),

			WithProfiles(createProfiles, component.StabilityLevelDevelopment),
		),
	}...)

	require.NoError(t, err)

	bErr := NewBuilder(map[component.ID]component.Config{}, factories)
	missingID := component.MustNewIDWithName("all", "missing")

	te, err := bErr.CreateTraces(context.Background(), settings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, te)

	me, err := bErr.CreateMetrics(context.Background(), settings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, me)

	le, err := bErr.CreateLogs(context.Background(), settings(missingID), consumertest.NewNop())
	assert.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, le)

	pe, err := bErr.CreateProfiles(context.Background(), settings(missingID), consumerexperimentaltest.NewNop())
	assert.EqualError(t, err, "receiver \"all/missing\" is not configured")
	assert.Nil(t, pe)
}

func TestBuilderFactory(t *testing.T) {
	factories, err := MakeFactoryMap([]Factory{NewFactory(component.MustNewType("foo"), nil)}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{component.MustNewID("foo"): struct{}{}}
	b := NewBuilder(cfgs, factories)

	assert.NotNil(t, b.Factory(component.MustNewID("foo").Type()))
	assert.Nil(t, b.Factory(component.MustNewID("bar").Type()))
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

func createProfiles(context.Context, Settings, component.Config, consumerexperimental.Profiles) (Profiles, error) {
	return nopInstance, nil
}

func settings(id component.ID) Settings {
	return Settings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
