// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteJSONSchema(t *testing.T) {
	dir := t.TempDir()
	doc := &JSONSchemaDoc{ConfigMetadata: &ConfigMetadata{
		Schema: schemaVersion,
		Type:   "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string"},
		},
	}}

	err := WriteJSONSchema(dir, doc)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, fileName)) // #nosec G304
	require.NoError(t, err)
	require.Contains(t, string(content), `"$schema"`)
	require.Contains(t, string(content), `"endpoint"`)
}

func TestWriteJSONSchema_OmitsInternalResolvedFrom(t *testing.T) {
	dir := t.TempDir()
	doc := &JSONSchemaDoc{ConfigMetadata: &ConfigMetadata{
		Schema:       schemaVersion,
		Type:         "object",
		ResolvedFrom: "go.opentelemetry.io/collector/config/confighttp.ClientConfig",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {
				Type:         "string",
				ResolvedFrom: "string",
			},
		},
	}}

	err := WriteJSONSchema(dir, doc)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, fileName)) // #nosec G304
	require.NoError(t, err)
	require.NotContains(t, string(content), "ResolvedFrom")
}

func TestWriteJSONSchema_InvalidDir(t *testing.T) {
	doc := &JSONSchemaDoc{ConfigMetadata: &ConfigMetadata{Type: "object"}}
	err := WriteJSONSchema("/nonexistent/path/that/does/not/exist", doc)
	require.Error(t, err)
}

func TestJSONSchemaDoc_EmitsDefs(t *testing.T) {
	dir := t.TempDir()
	doc := (&Metadata{
		Config: &ConfigMetadata{Type: "object", Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string"},
		}},
		ExportedConfigs: map[string]*ConfigMetadata{
			"sub_config": {Type: "object", Properties: map[string]*ConfigMetadata{
				"port": {Type: "integer"},
			}},
		},
	}).AsJSONSchema()

	err := WriteJSONSchema(dir, doc)
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(dir, fileName)) // #nosec G304
	require.NoError(t, err)
	require.Contains(t, string(content), `"$defs"`)
	require.Contains(t, string(content), `"sub_config"`)
	require.Contains(t, string(content), `"port"`)
}
