// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteJSONSchema(t *testing.T) {
	dir := t.TempDir()
	md := &ConfigMetadata{
		Schema: schemaVersion,
		Type:   "object",
		Properties: map[string]*ConfigMetadata{
			"endpoint": {Type: "string"},
		},
	}

	err := WriteJSONSchema(dir, md)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, fileName)) // #nosec G304
	require.NoError(t, err)
	require.Contains(t, string(content), `"$schema"`)
	require.Contains(t, string(content), `"endpoint"`)
}

func TestWriteJSONSchema_InvalidDir(t *testing.T) {
	md := &ConfigMetadata{Type: "object"}
	err := WriteJSONSchema("/nonexistent/path/that/does/not/exist", md)
	require.Error(t, err)
}
