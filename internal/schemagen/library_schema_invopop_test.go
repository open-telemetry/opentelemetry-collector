// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToInvopopSchema_MapsCoreFields(t *testing.T) {
	src := &Metadata{
		Config: &ConfigMetadata{
			Schema:      schemaVersion,
			Type:        "object",
			Title:       "TestComponent",
			Description: "A test schema",
			Properties: map[string]*ConfigMetadata{
				"endpoint": {Type: "string", Description: "The endpoint"},
			},
		},
		ExportedConfigs: map[string]*ConfigMetadata{
			"sub_config": {Type: "object"},
		},
	}

	got := ToInvopopSchema(src)
	require.NotNil(t, got)
	require.Equal(t, "object", got.Type)
	require.Equal(t, "TestComponent", got.Title)
	require.NotNil(t, got.Properties)
	endpoint, ok := got.Properties.Get("endpoint")
	require.True(t, ok)
	require.Equal(t, "The endpoint", endpoint.Description)
	require.Contains(t, got.Definitions, "sub_config")

	data, err := json.Marshal(got)
	require.NoError(t, err)
	require.Contains(t, string(data), `"$defs"`)
	require.Contains(t, string(data), `"sub_config"`)
	require.Contains(t, string(data), `"endpoint"`)
}

func TestToInvopopSchema_CustomExtensionsRoutedThroughExtras(t *testing.T) {
	src := &Metadata{
		Config: &ConfigMetadata{
			Type: "object",
			Properties: map[string]*ConfigMetadata{
				"timeout": {
					Type:       "string",
					GoType:     "time.Duration",
					IsPointer:  true,
					IsOptional: true,
				},
			},
		},
	}

	got := ToInvopopSchema(src)
	timeout, ok := got.Properties.Get("timeout")
	require.True(t, ok)
	require.Equal(t, "time.Duration", timeout.Extras["x-customType"])
	require.Equal(t, true, timeout.Extras["x-pointer"])
	require.Equal(t, true, timeout.Extras["x-optional"])

	data, err := json.Marshal(got)
	require.NoError(t, err)
	// The library hoists Extras into the schema body at marshal time.
	require.Contains(t, string(data), `"x-customType":"time.Duration"`)
	require.Contains(t, string(data), `"x-pointer":true`)
	require.Contains(t, string(data), `"x-optional":true`)
}
