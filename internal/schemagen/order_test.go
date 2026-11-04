// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/confmap"
)

func unmarshalConfigsMetadata(t *testing.T, raw string) *ConfigsMetadata {
	t.Helper()
	var rawMap map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(raw), &rawMap))
	var cm ConfigsMetadata
	require.NoError(t, confmap.NewFromStringMap(rawMap).Unmarshal(&cm, confmap.WithIgnoreUnused()))
	return &cm
}

func TestOrderedPropNames_NoRecordedOrder(t *testing.T) {
	md := &ConfigMetadata{
		Properties: map[string]*ConfigMetadata{
			"zebra": {Type: "string"},
			"alpha": {Type: "string"},
			"mid":   {Type: "string"},
		},
	}
	assert.Equal(t, []string{"alpha", "mid", "zebra"}, OrderedPropNames(md))
}

func TestOrderedPropNames_NilAndEmpty(t *testing.T) {
	assert.Nil(t, OrderedPropNames(nil))
	assert.Empty(t, OrderedPropNames(&ConfigMetadata{}))
}

func TestOrderedPropNames_PartialOrderFallsBackAlphabeticallyForTheRest(t *testing.T) {
	md := &ConfigMetadata{
		Properties: map[string]*ConfigMetadata{
			"zebra": {Type: "string"},
			"alpha": {Type: "string"},
			"mid":   {Type: "string"},
		},
		propOrder: []string{"zebra"},
	}
	// zebra was explicitly ordered first; alpha/mid have no recorded order and
	// are appended afterwards, alphabetically.
	assert.Equal(t, []string{"zebra", "alpha", "mid"}, OrderedPropNames(md))
}

func TestApplyConfigsOrder_PreservesDeclarationOrder(t *testing.T) {
	raw := `
config:
  type: object
  properties:
    zebra:
      type: string
    alpha:
      type: object
      properties:
        delta:
          type: string
        bravo:
          type: string
    mid:
      type: string
`
	cm := unmarshalConfigsMetadata(t, raw)
	require.NoError(t, ApplyConfigsOrder([]byte(raw), cm))

	assert.Equal(t, []string{"zebra", "alpha", "mid"}, OrderedPropNames(cm.Config))
	// nested object properties get their own declaration order too.
	assert.Equal(t, []string{"delta", "bravo"}, OrderedPropNames(cm.Config.Properties["alpha"]))
}

func TestApplyConfigsOrder_ExportedConfigs(t *testing.T) {
	raw := `
exported_configs:
  batch_config:
    type: object
    properties:
      timeout:
        type: string
      size:
        type: string
`
	cm := unmarshalConfigsMetadata(t, raw)
	require.NoError(t, ApplyConfigsOrder([]byte(raw), cm))

	assert.Equal(t, []string{"timeout", "size"}, OrderedPropNames(cm.ExportedConfigs["batch_config"]))
}

func TestApplyConfigsOrder_NilConfigsMetadataIsNoop(t *testing.T) {
	require.NoError(t, ApplyConfigsOrder([]byte("config: {}"), nil))
}

func TestApplyNamedDefsOrder_FlatDocument(t *testing.T) {
	raw := `
metrics_config:
  type: object
  properties:
    enabled:
      type: bool
    aggregation_strategy:
      type: string
events_config:
  type: object
  properties:
    zed:
      type: string
    abe:
      type: string
`
	defs := map[string]*ConfigMetadata{
		"metrics_config": {
			Properties: map[string]*ConfigMetadata{
				"enabled":              {Type: "bool"},
				"aggregation_strategy": {Type: "string"},
			},
		},
		"events_config": {
			Properties: map[string]*ConfigMetadata{
				"zed": {Type: "string"},
				"abe": {Type: "string"},
			},
		},
	}

	require.NoError(t, ApplyNamedDefsOrder([]byte(raw), defs))

	assert.Equal(t, []string{"enabled", "aggregation_strategy"}, OrderedPropNames(defs["metrics_config"]))
	assert.Equal(t, []string{"zed", "abe"}, OrderedPropNames(defs["events_config"]))
}
