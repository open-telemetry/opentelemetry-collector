// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ocbplugin

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
			m := &mockPlugin{minVersion: "0.150.0"}
			input := "action: " + tt.action + "\nocb_version: v0.151.0\nconfig:\n  key: foo\n"
			inputFile := filepath.Join(t.TempDir(), "input.yaml")
			require.NoError(t, os.WriteFile(inputFile, []byte(input), 0600))
			err := runPlugin(m, inputFile)
			require.NoError(t, err)
			tt.validate(t, m)
		})
	}
}

func TestRunPlugin_UnsupportedVersion(t *testing.T) {
	m := &mockPlugin{minVersion: "0.151.0"}
	input := "action: pre-build\nocb_version: v0.150.0\nconfig:\n  key: foo\n"
	inputFile := filepath.Join(t.TempDir(), "input.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(input), 0600))
	err := runPlugin(m, inputFile)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedOCBVersion)
}

func TestRunPlugin_ActionError(t *testing.T) {
	expectedErr := errors.New("custom pre-build error")
	m := &mockPlugin{preBuildErr: expectedErr}
	input := "action: pre-build\nconfig:\n  key: foo\n"
	inputFile := filepath.Join(t.TempDir(), "input.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(input), 0600))
	err := runPlugin(m, inputFile)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestRunPlugin_UnknownAction(t *testing.T) {
	m := &mockPlugin{}
	input := "action: invalid-action\n"
	inputFile := filepath.Join(t.TempDir(), "input.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(input), 0600))
	err := runPlugin(m, inputFile)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownAction)
	assert.Contains(t, err.Error(), "invalid-action")
}

func TestRunPlugin_InvalidYAML(t *testing.T) {
	m := &mockPlugin{}
	input := ": invalid: yaml: ["
	inputFile := filepath.Join(t.TempDir(), "input.yaml")
	require.NoError(t, os.WriteFile(inputFile, []byte(input), 0600))
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

func buildDummyPlugin(t *testing.T) string {
	t.Helper()
	binName := "dummyplugin"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(t.TempDir(), binName)
	cmd := exec.Command("go", "build", "-o", binPath, "../testdata/dummyplugin")
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "failed to build dummyplugin: %s", string(out))
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(binPath))
	})
	return binPath
}

func runDummyPluginSubprocess(binPath string, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestRunPlugin_Subprocess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping subprocess integration test in short mode")
		return
	}

	binPath := buildDummyPlugin(t)

	tests := []struct {
		name       string
		input      string
		wantStdout string
		wantStderr string
	}{
		{
			name:       "pre-generate",
			input:      "action: pre-generate\nocb_version: v0.151.0\nconfig:\n  message: foo\n",
			wantStdout: "pre-generate:foo\n",
		},
		{
			name:       "post-generate",
			input:      "action: post-generate\nocb_version: v0.151.0\nconfig:\n  message: foo\n",
			wantStdout: "post-generate:foo\n",
		},
		{
			name:       "pre-build",
			input:      "action: pre-build\nocb_version: v0.151.0\nconfig:\n  message: foo\n",
			wantStdout: "pre-build:foo\n",
		},
		{
			name:       "post-build",
			input:      "action: post-build\nocb_version: v0.151.0\nconfig:\n  message: foo\n",
			wantStdout: "post-build:foo\n",
		},
		{
			name:       "unsupported ocb version exits non-zero",
			input:      "action: pre-generate\nocb_version: v0.150.0\nconfig:\n  message: foo\n",
			wantStderr: ErrUnsupportedOCBVersion.Error(),
		},
		{
			name:       "unsupported hook action exits non-zero",
			input:      "action: pre-generate\nocb_version: v0.151.0\nconfig:\n  unsupported: true\n",
			wantStderr: ErrUnsupportedActionPreGenerate.Error(),
		},
		{
			name:       "action error exits non-zero",
			input:      "action: pre-build\nocb_version: v0.151.0\nconfig:\n  error: custom failure\n",
			wantStderr: "error running 'pre-build' plugin action: custom failure",
		},
		{
			name:       "unknown action exits non-zero",
			input:      "action: invalid-action\nocb_version: v0.151.0\n",
			wantStderr: ErrUnknownAction.Error(),
		},
		{
			name:       "missing input file exits non-zero",
			wantStderr: "error reading plugin input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputFile := filepath.Join(t.TempDir(), "input.yaml")
			if tt.input != "" {
				require.NoError(t, os.WriteFile(inputFile, []byte(tt.input), 0600))
			}

			stdout, stderr, err := runDummyPluginSubprocess(binPath, inputFile)
			if tt.wantStderr != "" {
				require.Error(t, err)
				assert.Empty(t, stdout)
				assert.Contains(t, stderr, tt.wantStderr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantStdout, stdout)
			assert.Empty(t, stderr)
		})
	}
}
