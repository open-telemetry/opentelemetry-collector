// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmaptest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

func TestLoadConfFileNotFound(t *testing.T) {
	_, err := LoadConf("file/not/found")
	assert.Error(t, err)
}

func TestLoadConfInvalidYAML(t *testing.T) {
	_, err := LoadConf(filepath.Join("testdata", "invalid.yaml"))
	require.Error(t, err)
}

func TestLoadConf(t *testing.T) {
	cfg, err := LoadConf(filepath.Join("testdata", "simple.yaml"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"floating": 3.14}, cfg.ToStringMap())
}

func TestToStringMapSanitizeEmptySlice(t *testing.T) {
	cfg, err := LoadConf(filepath.Join("testdata", "empty-slice.yaml"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"slice": []any{}}, cfg.ToStringMap())
}

func TestValidateProviderScheme(t *testing.T) {
	assert.NoError(t, ValidateProviderScheme(&schemeProvider{scheme: "file"}))
	assert.NoError(t, ValidateProviderScheme(&schemeProvider{scheme: "s3"}))
	assert.NoError(t, ValidateProviderScheme(&schemeProvider{scheme: "a.l-l+"}))
	// Too short.
	require.Error(t, ValidateProviderScheme(&schemeProvider{scheme: "a"}))
	// Invalid first character.
	require.Error(t, ValidateProviderScheme(&schemeProvider{scheme: "3s"}))
	// Invalid underscore character.
	assert.Error(t, ValidateProviderScheme(&schemeProvider{scheme: "all_"}))
}

type schemeProvider struct {
	scheme string
}

func (s schemeProvider) Retrieve(context.Context, string, confmap.WatcherFunc) (*confmap.Retrieved, error) {
	return nil, nil
}

func (s schemeProvider) Scheme() string {
	return s.scheme
}

func (s schemeProvider) Shutdown(context.Context) error {
	return nil
}
