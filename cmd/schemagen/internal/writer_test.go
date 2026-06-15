// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteSchemaToFile_YAML(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Mode:         Component,
		OutputFolder: dir,
		FileType:     "yaml",
	}
	schema := testSchema(t)

	path, err := WriteSchemaToFile(schema, cfg)
	require.NoError(t, err)

	expectedPath := filepath.Join(dir, "config.schema.yaml")
	require.Equal(t, expectedPath, path)
	require.FileExists(t, path)

	expected, err := schema.ToYAML()
	require.NoError(t, err)

	actual, err := os.ReadFile(path) // #nosec G304
	require.NoError(t, err)
	require.Equal(t, string(expected), string(actual))
}

func TestWriteSchemaToFile_JSON(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Mode:         Component,
		OutputFolder: dir,
		FileType:     "json",
	}
	schema := testSchema(t)

	path, err := WriteSchemaToFile(schema, cfg)
	require.NoError(t, err)

	expectedPath := filepath.Join(dir, "config.schema.json")
	require.Equal(t, expectedPath, path)
	require.FileExists(t, path)

	expected, err := schema.ToJSON()
	require.NoError(t, err)

	actual, err := os.ReadFile(path) // #nosec G304
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(actual))
}

func TestWriteSchemaToFile_Package(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Mode:         Package,
		DirPath:      dir,
		OutputFolder: dir,
		FileType:     "json",
	}
	schema := testSchema(t)

	path, err := WriteSchemaToFile(schema, cfg)
	require.NoError(t, err)

	expectedPath := filepath.Join(dir, "config.schema.json")
	require.Equal(t, expectedPath, path)
	require.FileExists(t, path)

	expected, err := schema.ToJSON()
	require.NoError(t, err)

	actual, err := os.ReadFile(path) // #nosec G304
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(actual))
}

func TestWriteSchemaToFile_WriteError(t *testing.T) {
	dir := t.TempDir()
	unwritable := filepath.Join(dir, "missing") // dir absent so os.WriteFile fails
	cfg := &Config{
		Mode:         Component,
		OutputFolder: unwritable,
		FileType:     "yaml",
	}
	schema := testSchema(t)

	path, err := WriteSchemaToFile(schema, cfg)
	require.Error(t, err)

	expectedPath := filepath.Join(unwritable, "config.schema.yaml")
	require.Empty(t, path)
	require.NoFileExists(t, expectedPath)
}

func testSchema(t *testing.T) *Schema {
	t.Helper()
	schema := CreateSchema()
	schema.AddProperty("name", CreateSimpleField(SchemaTypeString, "field"))
	return schema
}
