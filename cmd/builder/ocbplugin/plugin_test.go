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
	"go.opentelemetry.io/collector/cmd/builder/ocbplugin/internal/helper"
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

func TestRunPlugin_Invalid(t *testing.T) {
	errCustom := errors.New("custom pre-build error")

	tests := []struct {
		name        string
		plugin      *mockPlugin
		input       string
		noInputFile bool
		wantErrIs   error
		wantErrMsg  string
	}{
		{
			name:      "unsupported version",
			plugin:    &mockPlugin{minVersion: "v0.151.0"},
			input:     "action: pre-build\nocb_version: v0.150.0\nconfig:\n  key: foo\n",
			wantErrIs: ErrUnsupportedOCBVersion,
		},
		{
			name:      "invalid ocb version",
			plugin:    &mockPlugin{minVersion: "v0.151.0"},
			input:     "action: pre-build\nocb_version: latest\nconfig:\n  key: foo\n",
			wantErrIs: helper.ErrInvalidSemverString,
		},
		{
			name:      "invalid plugin min version",
			plugin:    &mockPlugin{minVersion: "latest"},
			input:     "action: pre-build\nocb_version: v0.150.0\nconfig:\n  key: foo\n",
			wantErrIs: helper.ErrInvalidSemverString,
		},
		{
			name:      "action error",
			plugin:    &mockPlugin{minVersion: "v0.150.0", preBuildErr: errCustom},
			input:     "action: pre-build\nocb_version: v0.151.0\nconfig:\n  key: foo\n",
			wantErrIs: errCustom,
		},
		{
			name:       "unknown action",
			plugin:     &mockPlugin{minVersion: "v0.150.0"},
			input:      "action: invalid-action\nocb_version: v0.151.0\n",
			wantErrIs:  ErrUnknownAction,
			wantErrMsg: "invalid-action",
		},
		{
			name:       "invalid yaml",
			plugin:     &mockPlugin{},
			input:      ": invalid: yaml: [",
			wantErrMsg: "error decoding plugin input",
		},
		{
			name:        "missing file",
			plugin:      &mockPlugin{},
			noInputFile: true,
			wantErrIs:   os.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputFile := filepath.Join(t.TempDir(), "input.yaml")
			if !tt.noInputFile {
				require.NoError(t, os.WriteFile(inputFile, []byte(tt.input), 0o600))
			}
			err := runPlugin(tt.plugin, inputFile)
			require.Error(t, err)
			if tt.wantErrIs != nil {
				assert.ErrorIs(t, err, tt.wantErrIs)
			}
			if tt.wantErrMsg != "" {
				assert.ErrorContains(t, err, tt.wantErrMsg)
			}
		})
	}
}
