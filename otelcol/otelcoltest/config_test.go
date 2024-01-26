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
	assert.Contains(t, cfg.Extensions, component.NewID(component.MustType("nop")))
	assert.Contains(t, cfg.Extensions, component.NewIDWithName(component.MustType("nop"), "myextension"))

	// Verify receivers
	require.Len(t, cfg.Receivers, 2)
	assert.Contains(t, cfg.Receivers, component.NewID(component.MustType("nop")))
	assert.Contains(t, cfg.Receivers, component.NewIDWithName(component.MustType("nop"), "myreceiver"))

	// Verify exporters
	assert.Len(t, cfg.Exporters, 2)
	assert.Contains(t, cfg.Exporters, component.NewID(component.MustType("nop")))
	assert.Contains(t, cfg.Exporters, component.NewIDWithName(component.MustType("nop"), "myexporter"))

	// Verify procs
	assert.Len(t, cfg.Processors, 2)
	assert.Contains(t, cfg.Processors, component.NewID(component.MustType("nop")))
	assert.Contains(t, cfg.Processors, component.NewIDWithName(component.MustType("nop"), "myprocessor"))

	// Verify connectors
	assert.Len(t, cfg.Connectors, 1)
	assert.Contains(t, cfg.Connectors, component.NewIDWithName(component.MustType("nop"), "myconnector"))

	// Verify service.
	require.Len(t, cfg.Service.Extensions, 1)
	assert.Contains(t, cfg.Service.Extensions, component.NewID(component.MustType("nop")))
	require.Len(t, cfg.Service.Pipelines, 1)
	assert.Equal(t,
		&pipelines.PipelineConfig{
			Receivers:  []component.ID{component.NewID(component.MustType("nop"))},
			Processors: []component.ID{component.NewID(component.MustType("nop"))},
			Exporters:  []component.ID{component.NewID(component.MustType("nop"))},
		},
		cfg.Service.Pipelines[component.NewID(component.MustType("traces"))],
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
