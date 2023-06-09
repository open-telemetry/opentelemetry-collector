// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcoltest

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/pipelines"
)

func TestLoadConfig(t *testing.T) {
	factories, err := NopFactories()
	assert.NoError(t, err)

	cfg, err := LoadConfig(filepath.Join("testdata", "config.yaml"), factories)
	require.NoError(t, err)

	// Verify extensions.
	require.Len(t, cfg.Extensions, 2)
	assert.Contains(t, cfg.Extensions, component.NewID("nop"))
	assert.Contains(t, cfg.Extensions, component.NewIDWithName("nop", "myextension"))

	// Verify receivers
	require.Len(t, cfg.Receivers, 2)
	assert.Contains(t, cfg.Receivers, component.NewID("nop"))
	assert.Contains(t, cfg.Receivers, component.NewIDWithName("nop", "myreceiver"))

	// Verify exporters
	assert.Len(t, cfg.Exporters, 2)
	assert.Contains(t, cfg.Exporters, component.NewID("nop"))
	assert.Contains(t, cfg.Exporters, component.NewIDWithName("nop", "myexporter"))

	// Verify procs
	assert.Len(t, cfg.Processors, 2)
	assert.Contains(t, cfg.Processors, component.NewID("nop"))
	assert.Contains(t, cfg.Processors, component.NewIDWithName("nop", "myprocessor"))

	// Verify connectors
	assert.Len(t, cfg.Connectors, 1)
	assert.Contains(t, cfg.Connectors, component.NewIDWithName("nop", "myconnector"))

	// Verify service.
	require.Len(t, cfg.Service.Extensions, 1)
	assert.Contains(t, cfg.Service.Extensions, component.NewID("nop"))
	require.Len(t, cfg.Service.Pipelines, 1)
	assert.Equal(t,
		&pipelines.PipelineConfig{
			Receivers:  []component.ID{component.NewID("nop")},
			Processors: []component.ID{component.NewID("nop")},
			Exporters:  []component.ID{component.NewID("nop")},
		},
		cfg.Service.Pipelines[component.NewID("traces")],
		"Did not load pipeline config correctly")
}

func TestLoadConfigAndValidate(t *testing.T) {
	factories, err := NopFactories()
	assert.NoError(t, err)

	cfgValidate, errValidate := LoadConfigAndValidate(filepath.Join("testdata", "config.yaml"), factories)
	require.NoError(t, errValidate)

	cfg, errLoad := LoadConfig(filepath.Join("testdata", "config.yaml"), factories)
	require.NoError(t, errLoad)

	assert.Equal(t, cfg, cfgValidate)
}
