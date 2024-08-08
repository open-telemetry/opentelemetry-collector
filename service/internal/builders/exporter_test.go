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
)

func TestBuilder(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := exporter.MakeFactoryMap([]exporter.Factory{
		exporter.NewFactory(component.MustNewType("err"), nil),
		exporter.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			exporter.WithTraces(createExporterTraces, component.StabilityLevelDevelopment),
			exporter.WithMetrics(createExporterMetrics, component.StabilityLevelAlpha),
			exporter.WithLogs(createExporterLogs, component.StabilityLevelDeprecated),
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
			b := NewExporter(cfgs, factories)

			te, err := b.CreateTraces(context.Background(), createExporterSettings(tt.id))
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, te)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopExporterInstance, te)
			}

			me, err := b.CreateMetrics(context.Background(), createExporterSettings(tt.id))
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, me)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopExporterInstance, me)
			}

			le, err := b.CreateLogs(context.Background(), createExporterSettings(tt.id))
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
				assert.Nil(t, le)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, nopExporterInstance, le)
			}
		})
	}
}

func TestBuilderMissingConfig(t *testing.T) {
	defaultCfg := struct{}{}
	factories, err := exporter.MakeFactoryMap([]exporter.Factory{
		exporter.NewFactory(
			component.MustNewType("all"),
			func() component.Config { return &defaultCfg },
			exporter.WithTraces(createExporterTraces, component.StabilityLevelDevelopment),
			exporter.WithMetrics(createExporterMetrics, component.StabilityLevelAlpha),
			exporter.WithLogs(createExporterLogs, component.StabilityLevelDeprecated),
		),
	}...)

	require.NoError(t, err)

	bErr := NewExporter(map[component.ID]component.Config{}, factories)
	missingID := component.MustNewIDWithName("all", "missing")

	te, err := bErr.CreateTraces(context.Background(), createExporterSettings(missingID))
	assert.EqualError(t, err, "exporter \"all/missing\" is not configured")
	assert.Nil(t, te)

	me, err := bErr.CreateMetrics(context.Background(), createExporterSettings(missingID))
	assert.EqualError(t, err, "exporter \"all/missing\" is not configured")
	assert.Nil(t, me)

	le, err := bErr.CreateLogs(context.Background(), createExporterSettings(missingID))
	assert.EqualError(t, err, "exporter \"all/missing\" is not configured")
	assert.Nil(t, le)
}

func TestBuilderFactory(t *testing.T) {
	factories, err := exporter.MakeFactoryMap([]exporter.Factory{exporter.NewFactory(component.MustNewType("foo"), nil)}...)
	require.NoError(t, err)

	cfgs := map[component.ID]component.Config{component.MustNewID("foo"): struct{}{}}
	b := NewExporter(cfgs, factories)

	assert.NotNil(t, b.Factory(component.MustNewID("foo").Type()))
	assert.Nil(t, b.Factory(component.MustNewID("bar").Type()))
}

func TestNewNopExporterConfigsAndFactories(t *testing.T) {
	configs, factories := NewNopExporterConfigsAndFactories()
	builder := NewExporter(configs, factories)
	require.NotNil(t, builder)

	factory := exportertest.NewNopFactory()
	cfg := factory.CreateDefaultConfig()
	set := exportertest.NewNopSettings()
	set.ID = component.NewID(nopType)

	traces, err := factory.CreateTracesExporter(context.Background(), set, cfg)
	require.NoError(t, err)
	bTraces, err := builder.CreateTraces(context.Background(), set)
	require.NoError(t, err)
	assert.IsType(t, traces, bTraces)

	metrics, err := factory.CreateMetricsExporter(context.Background(), set, cfg)
	require.NoError(t, err)
	bMetrics, err := builder.CreateMetrics(context.Background(), set)
	require.NoError(t, err)
	assert.IsType(t, metrics, bMetrics)

	logs, err := factory.CreateLogsExporter(context.Background(), set, cfg)
	require.NoError(t, err)
	bLogs, err := builder.CreateLogs(context.Background(), set)
	require.NoError(t, err)
	assert.IsType(t, logs, bLogs)
}

var nopExporterInstance = &nopExporter{
	Consumer: consumertest.NewNop(),
}

// nopExporter stores consumed traces and metrics for testing purposes.
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

func createExporterSettings(id component.ID) exporter.Settings {
	return exporter.Settings{
		ID:                id,
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
