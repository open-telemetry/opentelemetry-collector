// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ocbplugin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPlugin struct {
	preGenerateErr  error
	postGenerateErr error
	preBuildErr     error
	postBuildErr    error
	minVersion      string

	preGenerateCalled  bool
	postGenerateCalled bool
	preBuildCalled     bool
	postBuildCalled    bool
	lastConfig         map[string]any
}

func (m *mockPlugin) PreGenerate(config map[string]any) error {
	m.preGenerateCalled = true
	m.lastConfig = config
	return m.preGenerateErr
}

func (m *mockPlugin) PostGenerate(config map[string]any) error {
	m.postGenerateCalled = true
	m.lastConfig = config
	return m.postGenerateErr
}

func (m *mockPlugin) PreBuild(config map[string]any) error {
	m.preBuildCalled = true
	m.lastConfig = config
	return m.preBuildErr
}

func (m *mockPlugin) PostBuild(config map[string]any) error {
	m.postBuildCalled = true
	m.lastConfig = config
	return m.postBuildErr
}

func (m *mockPlugin) MinOCBVersion() string {
	return m.minVersion
}

func TestRunPlugin_LifecycleActions(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		validate func(t *testing.T, m *mockPlugin)
	}{
		{
			name:   "pre-generate",
			action: "pre-generate",
			validate: func(t *testing.T, m *mockPlugin) {
				assert.True(t, m.preGenerateCalled)
				assert.Equal(t, "foo", m.lastConfig["key"])
			},
		},
		{
			name:   "post-generate",
			action: "post-generate",
			validate: func(t *testing.T, m *mockPlugin) {
				assert.True(t, m.postGenerateCalled)
				assert.Equal(t, "foo", m.lastConfig["key"])
			},
		},
		{
			name:   "pre-build",
			action: "pre-build",
			validate: func(t *testing.T, m *mockPlugin) {
				assert.True(t, m.preBuildCalled)
				assert.Equal(t, "foo", m.lastConfig["key"])
			},
		},
		{
			name:   "post-build",
			action: "post-build",
			validate: func(t *testing.T, m *mockPlugin) {
				assert.True(t, m.postBuildCalled)
				assert.Equal(t, "foo", m.lastConfig["key"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockPlugin{minVersion: "v0.150.0"}
			input := "action: " + tt.action + "\nocb_version: v0.151.0\nconfig:\n  key: foo\n"
			inputFile := filepath.Join(t.TempDir(), "input.yaml")
			require.NoError(t, os.WriteFile(inputFile, []byte(input), 0o600))
			err := runPlugin(m, inputFile)
			require.NoError(t, err)
			tt.validate(t, m)
		})
	}
}

func TestRunPlugin_UnsupportedVersion(t *testing.T) {
	m := &mockPlugin{minVersion: "v0.151.0"}
	input := "action: pre-build\nocb_version: v0.150.0\nconfig:\n  key: foo\n"
	inputFile := filepath.Join(t.TempDir(), "input.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(input), 0o600))
	err := runPlugin(m, inputFile)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedOCBVersion)
}

func TestRunPlugin_InvalidOCBVersion(t *testing.T) {
	m := &mockPlugin{minVersion: "v0.151.0"}
	input := "action: pre-build\nocb_version: latest\nconfig:\n  key: foo\n"
	inputFile := filepath.Join(t.TempDir(), "input.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(input), 0o600))
	err := runPlugin(m, inputFile)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSemverString)
}

func TestRunPlugin_InvalidPluginMinVersion(t *testing.T) {
	m := &mockPlugin{minVersion: "latest"}
	input := "action: pre-build\nocb_version: v0.150.0\nconfig:\n  key: foo\n"
	inputFile := filepath.Join(t.TempDir(), "input.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(input), 0o600))
	err := runPlugin(m, inputFile)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSemverString)
}

func TestRunPlugin_ActionError(t *testing.T) {
	expectedErr := errors.New("custom pre-build error")
	m := &mockPlugin{minVersion: "v0.150.0", preBuildErr: expectedErr}
	input := "action: pre-build\nocb_version: v0.151.0\nconfig:\n  key: foo\n"
	inputFile := filepath.Join(t.TempDir(), "input.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(input), 0o600))
	err := runPlugin(m, inputFile)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestRunPlugin_UnknownAction(t *testing.T) {
	m := &mockPlugin{minVersion: "v0.150.0"}
	input := "action: invalid-action\nocb_version: v0.151.0\n"
	inputFile := filepath.Join(t.TempDir(), "input.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(input), 0o600))
	err := runPlugin(m, inputFile)
	require.ErrorIs(t, err, ErrUnknownAction)
	assert.Contains(t, err.Error(), "invalid-action")
}

func TestRunPlugin_InvalidYAML(t *testing.T) {
	m := &mockPlugin{}
	input := ": invalid: yaml: ["
	inputFile := filepath.Join(t.TempDir(), "input.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(input), 0o600))
	err := runPlugin(m, inputFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error decoding plugin input")
}

func TestRunPlugin_MissingFile(t *testing.T) {
	m := &mockPlugin{}
	err := runPlugin(m, filepath.Join(t.TempDir(), "nonexistent.yaml"))
	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
