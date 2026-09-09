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
	md := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string"},
		},
	}}

	err := WriteJSONSchema(dir, "test", "test", md)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, fileName)) // #nosec G304
	require.NoError(t, err)
	require.Contains(t, string(content), `"$schema"`)
	require.Contains(t, string(content), `"endpoint"`)
}

func TestWriteJSONSchema_OmitsInternalFields(t *testing.T) {
	dir := t.TempDir()
	md := &ConfigsMetadata{Config: &ConfigMetadata{
		Type: "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {
				Type:   "string",
				GoType: "string",
			},
		},
	}}

	err := WriteJSONSchema(dir, "test", "test", md)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, fileName)) // #nosec G304
	require.NoError(t, err)
	require.NotContains(t, string(content), "x-customType")
}

func TestWriteJSONSchema_InvalidDir(t *testing.T) {
	md := &ConfigsMetadata{Config: &ConfigMetadata{Type: "object"}}
	err := WriteJSONSchema("/nonexistent/path/that/does/not/exist", "test", "test", md)
	require.Error(t, err)
}
