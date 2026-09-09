// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name     string
		dst      map[string]any
		src      map[string]any
		expected map[string]any
	}{
		{
			name:     "scalar src wins",
			dst:      map[string]any{"a": "old"},
			src:      map[string]any{"a": "new"},
			expected: map[string]any{"a": "new"},
		},
		{
			name:     "nil dst",
			dst:      nil,
			src:      map[string]any{"a": 1},
			expected: map[string]any{"a": 1},
		},
		{
			name:     "missing key in dst is introduced",
			dst:      map[string]any{"a": 1},
			src:      map[string]any{"b": 2},
			expected: map[string]any{"a": 1, "b": 2},
		},
		{
			name:     "nested maps are recursed",
			dst:      map[string]any{"nested": map[string]any{"x": 1, "y": 2}},
			src:      map[string]any{"nested": map[string]any{"x": 10}},
			expected: map[string]any{"nested": map[string]any{"x": 10, "y": 2}},
		},
		{
			name:     "list overlay replaces dst list",
			dst:      map[string]any{"items": []any{1, 2}},
			src:      map[string]any{"items": []any{3}},
			expected: map[string]any{"items": []any{3}},
		},
		{
			name:     "src map vs dst scalar: src wins",
			dst:      map[string]any{"key": "scalar"},
			src:      map[string]any{"key": map[string]any{"nested": true}},
			expected: map[string]any{"key": map[string]any{"nested": true}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := deepMerge(tc.dst, tc.src)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestApplyOverlayToYAML(t *testing.T) {
	base := []byte(`
description: original description
type: object
properties:
  foo:
    type: string
`)
	overlayContent := `
description: overridden description
`
	tmpDir := t.TempDir()
	overlayPath := filepath.Join(tmpDir, "overlay.yaml")
	require.NoError(t, os.WriteFile(overlayPath, []byte(overlayContent), 0o600))

	cfg := &Config{
		ComponentOverride: &ComponentOverride{
			OverlayFile: overlayPath,
		},
	}

	result, err := ApplyOverlayToYAML(base, cfg)
	require.NoError(t, err)

	expected := `
description: overridden description
type: object
properties:
  foo:
    type: string
`
	require.YAMLEq(t, expected, string(result))
}

func TestApplyOverlayToYAML_NoOverlay(t *testing.T) {
	base := []byte(`type: object`)
	cfg := &Config{}

	result, err := ApplyOverlayToYAML(base, cfg)
	require.NoError(t, err)
	assert.Equal(t, base, result)
}

func TestApplyOverlayToYAML_MissingFile(t *testing.T) {
	base := []byte(`type: object`)
	cfg := &Config{
		ComponentOverride: &ComponentOverride{
			OverlayFile: "/nonexistent/path/overlay.yaml",
		},
	}

	_, err := ApplyOverlayToYAML(base, cfg)
	require.Error(t, err)
}

func TestApplyOverlayToYAML_NestedDefs(t *testing.T) {
	base := []byte(`
$defs:
  my_type:
    x-customType: some.type
type: object
properties:
  field:
    $ref: my_type
`)
	overlayContent := `
$defs:
  my_type:
    description: added via overlay
`
	tmpDir := t.TempDir()
	overlayPath := filepath.Join(tmpDir, "overlay.yaml")
	require.NoError(t, os.WriteFile(overlayPath, []byte(overlayContent), 0o600))

	cfg := &Config{
		ComponentOverride: &ComponentOverride{
			OverlayFile: overlayPath,
		},
	}

	result, err := ApplyOverlayToYAML(base, cfg)
	require.NoError(t, err)

	expected := `
$defs:
  my_type:
    description: added via overlay
    x-customType: some.type
type: object
properties:
  field:
    $ref: my_type
`
	require.YAMLEq(t, expected, string(result))
}
