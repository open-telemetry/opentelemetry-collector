// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fileprovider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

const fileSchemePrefix = schemeName + ":"

func TestValidateProviderScheme(t *testing.T) {
	assert.NoError(t, confmaptest.ValidateProviderScheme(createProvider()))
}

func TestEmptyName(t *testing.T) {
	fp := createProvider()
	_, err := fp.Retrieve(context.Background(), "", nil)
	require.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestUnsupportedScheme(t *testing.T) {
	fp := createProvider()
	_, err := fp.Retrieve(context.Background(), "https://", nil)
	require.Error(t, err)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestNonExistent(t *testing.T) {
	fp := createProvider()
	_, err := fp.Retrieve(context.Background(), fileSchemePrefix+filepath.Join("testdata", "non-existent.yaml"), nil)
	require.Error(t, err)
	_, err = fp.Retrieve(context.Background(), fileSchemePrefix+absolutePath(t, filepath.Join("testdata", "non-existent.yaml")), nil)
	require.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestInvalidYAML(t *testing.T) {
	fp := createProvider()
	ret, err := fp.Retrieve(context.Background(), fileSchemePrefix+filepath.Join("testdata", "invalid-yaml.yaml"), nil)
	require.NoError(t, err)
	raw, err := ret.AsRaw()
	require.NoError(t, err)
	assert.IsType(t, "", raw)

	ret, err = fp.Retrieve(context.Background(), fileSchemePrefix+absolutePath(t, filepath.Join("testdata", "invalid-yaml.yaml")), nil)
	require.NoError(t, err)
	raw, err = ret.AsRaw()
	require.NoError(t, err)
	assert.IsType(t, "", raw)

	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestRelativePath(t *testing.T) {
	fp := createProvider()
	ret, err := fp.Retrieve(context.Background(), fileSchemePrefix+filepath.Join("testdata", "default-config.yaml"), nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	expectedMap := confmap.NewFromStringMap(map[string]any{
		"processors::testprocessor": nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	assert.Equal(t, expectedMap, retMap)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestAbsolutePath(t *testing.T) {
	fp := createProvider()
	ret, err := fp.Retrieve(context.Background(), fileSchemePrefix+absolutePath(t, filepath.Join("testdata", "default-config.yaml")), nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	expectedMap := confmap.NewFromStringMap(map[string]any{
		"processors::testprocessor": nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	assert.Equal(t, expectedMap, retMap)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func absolutePath(t *testing.T, relativePath string) string {
	dir, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(dir, relativePath)
}

func createProvider() confmap.Provider {
	return NewFactory().Create(confmaptest.NewNopProviderSettings())
}
