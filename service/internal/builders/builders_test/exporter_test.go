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
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/service/internal/builders"
)

func TestExporterBuilder(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := otelcol.MakeFactoryMap([]exporter.Factory{
		exporter.NewFactory(component.MustNewType("err"), nil),
		xexporter.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			xexporter.WithTraces(createExporterTraces, component.StabilityLevelDevelopment),
			xexporter.WithMetrics(createExporterMetrics, component.StabilityLevelAlpha),
			xexporter.WithLogs(createExporterLogs, component.StabilityLevelDeprecated),
			xexporter.WithProfiles(createxexporter, component.StabilityLevelDevelopment),
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
			b := builders.NewExporter(cfgs, factories)

			te, err := b.CreateTraces(context.Background(), createExporterSettings(tt.id))
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, te)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopExporterInstance, te)
			}

			me, err := b.CreateMetrics(context.Background(), createExporterSettings(tt.id))
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, me)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopExporterInstance, me)
			}

			le, err := b.CreateLogs(context.Background(), createExporterSettings(tt.id))
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, le)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopExporterInstance, le)
			}

			pe, err := b.CreateProfiles(context.Background(), createExporterSettings(tt.id))
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				assert.Nil(t, pe)
			} else {
				require.NoError(t, err)
				assert.Equal(t, nopExporterInstance, pe)
			}
		})
	}
}

func TestExporterBuilderMissingConfig(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := otelcol.MakeFactoryMap([]exporter.Factory{
		xexporter.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			xexporter.WithTraces(createExporterTraces, component.StabilityLevelDevelopment),
			xexporter.WithMetrics(createExporterMetrics, component.StabilityLevelAlpha),
			xexporter.WithLogs(createExporterLogs, component.StabilityLevelDeprecated),
			xexporter.WithProfiles(createxexporter, component.StabilityLevelDevelopment),
		),
	}...)

	require.NoError(t, err)

	bErr := builders.NewExporter(map[component.ID]component.Config{}, factories)
	missingID := component.MustNewIDWithName("all", "missing")

	te, err := bErr.CreateTraces(context.Background(), createExporterSettings(missingID))
	require.EqualError(t, err, "exporter \"all/missing\" is not configured")
	assert.Nil(t, te)

	me, err := bErr.CreateMetrics(context.Background(), createExporterSettings(missingID))
	require.EqualError(t, err, "exporter \"all/missing\" is not configured")
	assert.Nil(t, me)

	le, err := bErr.CreateLogs(context.Background(), createExporterSettings(missingID))
	require.EqualError(t, err, "exporter \"all/missing\" is not configured")
	assert.Nil(t, le)

	pe, err := bErr.CreateProfiles(context.Background(), createExporterSettings(missingID))
	require.EqualError(t, err, "exporter \"all/missing\" is not configured")
	assert.Nil(t, pe)
}

func TestExporterBuilderFactory(t *testing.T) {
	factories, err := otelcol.MakeFactoryMap([]exporter.Factory{exporter.NewFactory(component.MustNewType("foo"), nil)}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{component.MustNewID("foo"): struct{}{}}
	b := builders.NewExporter(cfgs, factories)

	assert.NotNil(t, b.Factory(component.MustNewID("foo").Type()))
	assert.Nil(t, b.Factory(component.MustNewID("bar").Type()))
}

func TestNewNopExporterConfigsAndFactories(t *testing.T) {
	configs, factories := builders.NewNopExporterConfigsAndFactories()
	builder := builders.NewExporter(configs, factories)
	require.NotNil(t, builder)

	factory := exportertest.NewNopFactory()
	cfg := factory.CreateDefaultConfig()
	set := exportertest.NewNopSettings(factory.Type())
	set.ID = component.NewID(builders.NopType)

	traces, err := factory.CreateTraces(context.Background(), set, cfg)
	require.NoError(t, err)
	bTraces, err := builder.CreateTraces(context.Background(), set)
	require.NoError(t, err)
	assert.IsType(t, traces, bTraces)

	metrics, err := factory.CreateMetrics(context.Background(), set, cfg)
	require.NoError(t, err)
	bMetrics, err := builder.CreateMetrics(context.Background(), set)
	require.NoError(t, err)
	assert.IsType(t, metrics, bMetrics)

	logs, err := factory.CreateLogs(context.Background(), set, cfg)
	require.NoError(t, err)
	bLogs, err := builder.CreateLogs(context.Background(), set)
	require.NoError(t, err)
	assert.IsType(t, logs, bLogs)

	profiles, err := factory.(xexporter.Factory).CreateProfiles(context.Background(), set, cfg)
	require.NoError(t, err)
	bProfiles, err := builder.CreateProfiles(context.Background(), set)
	require.NoError(t, err)
	assert.IsType(t, profiles, bProfiles)
}

var nopExporterInstance = &nopExporter{
	Consumer: consumertest.NewNop(),
}

// nopExporter stores consumed traces, metrics, logs and profiles for testing purposes.
type nopExporter struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createExporterTraces(context.Context, exporter.Settings, component.Config) (exporter.Traces, error) {
	return nopExporterInstance, nil
}

func createExporterMetrics(context.Context, exporter.Settings, component.Config) (exporter.Metrics, error) {
	return nopExporterInstance, nil
}

func createExporterLogs(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
	return nopExporterInstance, nil
}

func createxexporter(context.Context, exporter.Settings, component.Config) (xexporter.Profiles, error) {
	return nopExporterInstance, nil
}

func createExporterSettings(id component.ID) exporter.Settings {
	return exporter.Settings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
