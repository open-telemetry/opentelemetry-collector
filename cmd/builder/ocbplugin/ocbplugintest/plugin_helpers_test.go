// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ocbplugintest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/cmd/builder/ocbplugin"
)

func TestIsValidOCBPlugin(t *testing.T) {
	testCases := []struct {
		name   string
		plugin ocbplugin.OCBPlugin
		valid  bool
	}{
		{
			name:   "valid plugin",
			plugin: &mockPlugin{minVersion: "v0.150.0"},
			valid:  true,
		},
		{
			name:   "invalid plugin bad OCBMinVersion",
			plugin: &mockPlugin{minVersion: "nonsense"},
			valid:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.valid, IsValidOCBPlugin(tc.plugin))
		})
	}
}

type mockPlugin struct {
	minVersion string
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
	return errors.New("unimplemented")
}

func (m *mockPlugin) PostBuild(_ map[string]any) error {
	return errors.New("unimplemented")
}
