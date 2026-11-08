// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ocbplugintest

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/cmd/builder/ocbplugin"
	"go.opentelemetry.io/collector/cmd/builder/ocbplugin/internal/helper"
)

func TestIsValidOCBPlugin(t *testing.T) {
	testCases := []struct {
		name        string
		plugin      ocbplugin.OCBPlugin
		expectedErr error
	}{
		{
			name:   "valid plugin",
			plugin: &mockPlugin{minVersion: "v0.150.0"},
		},
		{
			name:        "invalid plugin bad OCBMinVersion",
			plugin:      &mockPlugin{minVersion: "nonsense"},
			expectedErr: helper.ErrInvalidSemverString,
		},
		{
			name:   "non-normalized input version",
			plugin: &mockPlugin{minVersion: "0.150.0"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.expectedErr != nil {
				assert.ErrorIs(t, IsValidOCBPlugin(tc.plugin), helper.ErrInvalidSemverString)
			} else {
				assert.NoError(t, IsValidOCBPlugin(tc.plugin))
			}
		})
	}
}

func TestIsValidOCBPlugin_RunPlugin(t *testing.T) {
	plugin := &mockPlugin{minVersion: "0.150.0"}
	require.NoError(t, IsValidOCBPlugin(plugin))

	input := "action: pre-build\nocb_version: 0.151.0\nconfig:\n  key: foo\n"
	inputFile := filepath.Join(t.TempDir(), "input.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(input), 0o600))

	origArgs := os.Args
	origUsage := flag.Usage
	t.Cleanup(func() {
		os.Args = origArgs
		flag.Usage = origUsage
	})
	os.Args = []string{os.Args[0], inputFile}

	ocbplugin.RunPlugin(plugin)
	assert.True(t, plugin.preBuildCalled)
}

type mockPlugin struct {
	minVersion     string
	preBuildCalled bool
}

func (m *mockPlugin) MinOCBVersion() string {
	return m.minVersion
}

func (m *mockPlugin) PreGenerate(_ map[string]any) error {
	return errors.New("unimplemented")
}

func (m *mockPlugin) PostGenerate(_ map[string]any) error {
	return errors.New("unimplemented")
}

func (m *mockPlugin) PreBuild(_ map[string]any) error {
	m.preBuildCalled = true
	return nil
}

func (m *mockPlugin) PostBuild(_ map[string]any) error {
	return errors.New("unimplemented")
}
