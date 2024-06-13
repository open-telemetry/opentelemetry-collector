// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
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
	_, err := factory.CreateTracesExporter(context.Background(), Settings{}, &defaultCfg)
	assert.Error(t, err)
	_, err = factory.CreateMetricsExporter(context.Background(), Settings{}, &defaultCfg)
	assert.Error(t, err)
	_, err = factory.CreateLogsExporter(context.Background(), Settings{}, &defaultCfg)
	assert.Error(t, err)

	_, err = factory.CreateProfilesExporter(context.Background(), Settings{}, &defaultCfg)
	assert.Error(t, err)
}

func TestNewFactoryWithOptions(t *testing.T) {
	var testType = component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithTraces(createTraces, component.StabilityLevelDevelopment),
		WithMetrics(createMetrics, component.StabilityLevelAlpha),
		WithLogs(createLogs, component.StabilityLevelDeprecated),

		WithProfiles(createProfiles, component.StabilityLevelDevelopment),
	)
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDevelopment, factory.TracesExporterStability())
	_, err := factory.CreateTracesExporter(context.Background(), Settings{}, &defaultCfg)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.MetricsExporterStability())
	_, err = factory.CreateMetricsExporter(context.Background(), Settings{}, &defaultCfg)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.LogsExporterStability())
	_, err = factory.CreateLogsExporter(context.Background(), Settings{}, &defaultCfg)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDevelopment, factory.ProfilesExporterStability())
	_, err = factory.CreateProfilesExporter(context.Background(), Settings{}, &defaultCfg)
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
		name string
		id   component.ID
		err  string
	}{
		{
			name: "unknown",
			id:   component.MustNewID("unknown"),
			err:  "exporter factory not available for: \"unknown\"",
		},
		{
			name: "err",
			id:   component.MustNewID("err"),
			err:  "telemetry type is not supported",
		},
		{
			name: "all",
			id:   component.MustNewID("all"),
		},
		{
			name: "all/named",
			id:   component.MustNewIDWithName("all", "named"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfgs := map[component.ID]component.Config{tt.id: defaultCfg}
			b := NewBuilder(cfgs, factories)

			te, err := b.CreateTraces(context.Background(), createSettings(tt.id))
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, te)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, te)
			}

			me, err := b.CreateMetrics(context.Background(), createSettings(tt.id))
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, me)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, me)
			}

			le, err := b.CreateLogs(context.Background(), createSettings(tt.id))
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, le)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopInstance, le)
			}

			pe, err := b.CreateProfiles(context.Background(), createSettings(tt.id))
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, pe)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopExperimentalInstance, pe)
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

	te, err := bErr.CreateTraces(context.Background(), createSettings(missingID))
	assert.EqualError(t, err, "exporter \"all/missing\" is not configured")
	assert.Nil(t, te)

	me, err := bErr.CreateMetrics(context.Background(), createSettings(missingID))
	assert.EqualError(t, err, "exporter \"all/missing\" is not configured")
	assert.Nil(t, me)

	le, err := bErr.CreateLogs(context.Background(), createSettings(missingID))
	assert.EqualError(t, err, "exporter \"all/missing\" is not configured")
	assert.Nil(t, le)

	pe, err := bErr.CreateProfiles(context.Background(), createSettings(missingID))
	assert.EqualError(t, err, "exporter \"all/missing\" is not configured")
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

var nopInstance = &nopExporter{
	Consumer: consumertest.NewNop(),
}

var nopExperimentalInstance = &nopExperimentalExporter{
	Consumer: consumerexperimentaltest.NewNop(),
}

// nopExporter stores consumed traces and metrics for testing purposes.
type nopExporter struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

// nopExporter stores consumed profiles for testing purposes.
type nopExperimentalExporter struct {
	component.StartFunc
	component.ShutdownFunc
	consumerexperimentaltest.Consumer
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

func createProfiles(context.Context, Settings, component.Config) (Profiles, error) {
	return nopExperimentalInstance, nil
}

func createSettings(id component.ID) Settings {
	return Settings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
