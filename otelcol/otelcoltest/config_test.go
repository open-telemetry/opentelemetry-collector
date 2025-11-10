// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcoltest

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/pipelines"
)

func TestLoadConfig(t *testing.T) {
	factories, err := NopFactories()
	require.NoError(t, err)

	cfg, err := LoadConfig(filepath.Join("testdata", "config.yaml"), factories)
	require.NoError(t, err)

	// Verify extensions.
	require.Len(t, cfg.Extensions, 2)
	assert.Contains(t, cfg.Extensions, component.MustNewID("nop"))
	assert.Contains(t, cfg.Extensions, component.MustNewIDWithName("nop", "myextension"))

	// Verify receivers
	require.Len(t, cfg.Receivers, 2)
	assert.Contains(t, cfg.Receivers, component.MustNewID("nop"))
	assert.Contains(t, cfg.Receivers, component.MustNewIDWithName("nop", "myreceiver"))

	// Verify exporters
	assert.Len(t, cfg.Exporters, 2)
	assert.Contains(t, cfg.Exporters, component.MustNewID("nop"))
	assert.Contains(t, cfg.Exporters, component.MustNewIDWithName("nop", "myexporter"))

	// Verify procs
	assert.Len(t, cfg.Processors, 2)
	assert.Contains(t, cfg.Processors, component.MustNewID("nop"))
	assert.Contains(t, cfg.Processors, component.MustNewIDWithName("nop", "myprocessor"))

	// Verify connectors
	assert.Len(t, cfg.Connectors, 1)
	assert.Contains(t, cfg.Connectors, component.MustNewIDWithName("nop", "myconnector"))

	// Verify service.
	require.Len(t, cfg.Service.Extensions, 1)
	assert.Contains(t, cfg.Service.Extensions, component.MustNewID("nop"))
	require.Len(t, cfg.Service.Pipelines, 1)
	assert.Equal(t,
		&pipelines.PipelineConfig{
			Receivers:  []component.ID{component.MustNewID("nop")},
			Processors: []component.ID{component.MustNewID("nop")},
			Exporters:  []component.ID{component.MustNewID("nop")},
		},
		cfg.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)],
		"Did not load pipeline config correctly")

	// Verify telemetry
	assert.Equal(t, struct{}{}, cfg.Service.Telemetry)
}

func TestLoadConfig_DefaultNopTelemetry(t *testing.T) {
	factories, err := NopFactories()
	require.NoError(t, err)
	factories.Telemetry = nil

	cfg, err := LoadConfig(filepath.Join("testdata", "config.yaml"), factories)
	require.NoError(t, err)

	assert.Equal(t, struct{}{}, cfg.Service.Telemetry)
}

func TestLoadConfigAndValidate(t *testing.T) {
	factories, err := NopFactories()
	require.NoError(t, err)

	cfgValidate, errValidate := LoadConfigAndValidate(filepath.Join("testdata", "config.yaml"), factories)
	require.NoError(t, errValidate)

	cfg, errLoad := LoadConfig(filepath.Join("testdata", "config.yaml"), factories)
	require.NoError(t, errLoad)

	assert.Equal(t, cfg, cfgValidate)
}

func TestLoadConfigEnv(t *testing.T) {
	factories, err := NopFactories()
	require.NoError(t, err)

	for _, tt := range []struct {
		file string
	}{
		{file: filepath.Join("testdata", "config_env.yaml")},
		{file: filepath.Join("testdata", "config_default_scheme.yaml")},
	} {
		t.Run(tt.file, func(t *testing.T) {
			t.Setenv("RECEIVERS", "[nop]")
			cfg, err := LoadConfigAndValidate(tt.file, factories)
			require.NoError(t, err)

			assert.Equal(t, []component.ID{component.MustNewID("nop")}, cfg.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)].Receivers)
		})
	}
}
