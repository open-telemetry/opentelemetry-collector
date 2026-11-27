// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package dirprovider

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/featuregate"
)

const dirSchemePrefix = schemeName + ":"

func createProvider() confmap.Provider {
	return NewFactory().Create(confmaptest.NewNopProviderSettings())
}

func TestValidateProviderScheme(t *testing.T) {
	assert.NoError(t, confmaptest.ValidateProviderScheme(createProvider()))
}

func TestUnsupportedScheme(t *testing.T) {
	dp := createProvider()
	_, err := dp.Retrieve(context.Background(), "https://", nil)
	require.Error(t, err)
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestEmptyName(t *testing.T) {
	dp := createProvider()
	_, err := dp.Retrieve(context.Background(), "", nil)
	require.Error(t, err)
	require.NoError(t, dp.Shutdown(context.Background()))
}

func TestBareDirectory(t *testing.T) {
	dp := createProvider()
	ret, err := dp.Retrieve(context.Background(), dirSchemePrefix+"testdata/bare", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	expected := confmap.NewFromStringMap(map[string]any{
		"processors::testprocessor":        nil,
		"exporters::otlp_grpc::endpoint":   "localhost:4317",
		"receivers::otlp::protocols::grpc": nil,
	})
	assert.Equal(t, expected, retMap)
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestNoMatchesIsError(t *testing.T) {
	dp := createProvider()
	_, err := dp.Retrieve(context.Background(), dirSchemePrefix+"testdata/empty", nil)
	require.Error(t, err)
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestNonExistentDirIsError(t *testing.T) {
	dp := createProvider()
	_, err := dp.Retrieve(context.Background(), dirSchemePrefix+"testdata/does-not-exist", nil)
	require.Error(t, err)
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestSingleLevelGlob(t *testing.T) {
	dp := createProvider()
	ret, err := dp.Retrieve(context.Background(), dirSchemePrefix+filepath.Join("testdata", "glob", "*.yaml"), nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.True(t, retMap.IsSet("exporters::debug"))
	assert.True(t, retMap.IsSet("service::pipelines::traces::exporters"))
	assert.False(t, retMap.IsSet("must"))
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestRecursiveGlob(t *testing.T) {
	dp := createProvider()
	ret, err := dp.Retrieve(context.Background(), dirSchemePrefix+filepath.Join("testdata", "recursive", "**", "*.yaml"), nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	// top.yaml is at root level and must NOT be loaded ("**" skips root-level files).
	assert.False(t, retMap.IsSet("receivers::nop"), "top.yaml at root must not be loaded")
	assert.True(t, retMap.IsSet("processors::nop"), "sub/nested.yaml should be loaded")
	assert.True(t, retMap.IsSet("exporters::nop"), "sub/deeper/deep.yaml should be loaded")
	assert.False(t, retMap.IsSet("ignore"), "ignore.txt must not be loaded")
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestRecursiveGlobBareDoubleStar(t *testing.T) {
	// dir:/path/** — tail is empty, exercises the tail="" -> "*" branch in recursiveGlob.
	dp := createProvider()
	ret, err := dp.Retrieve(context.Background(), dirSchemePrefix+filepath.Join("testdata", "doublestar", "**"), nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.True(t, retMap.IsSet("service::extensions"), "sub/config.yaml should be loaded")
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestInvalidGlobPatternIsError(t *testing.T) {
	// A malformed glob pattern (unclosed "[") must return an error.
	dp := createProvider()
	_, err := dp.Retrieve(context.Background(), dirSchemePrefix+"testdata/glob/[invalid", nil)
	require.Error(t, err)
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestSingleFileYAML(t *testing.T) {
	dp := createProvider()
	ret, err := dp.Retrieve(context.Background(), dirSchemePrefix+"testdata/bare/01-first.yaml", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.True(t, retMap.IsSet("processors::testprocessor"))
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestSingleFileNonYAMLIsError(t *testing.T) {
	dp := createProvider()
	_, err := dp.Retrieve(context.Background(), dirSchemePrefix+"testdata/bare/notes.txt", nil)
	require.Error(t, err)
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestGlobNoMatchesIsError(t *testing.T) {
	dp := createProvider()
	_, err := dp.Retrieve(context.Background(), dirSchemePrefix+filepath.Join("testdata", "empty", "*.yaml"), nil)
	require.Error(t, err)
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestRecursiveGlobNoMatchesIsError(t *testing.T) {
	dp := createProvider()
	_, err := dp.Retrieve(context.Background(), dirSchemePrefix+filepath.Join("testdata", "empty", "**", "*.yaml"), nil)
	require.Error(t, err)
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestMergeOrderLexicographical(t *testing.T) {
	dp := createProvider()
	ret, err := dp.Retrieve(context.Background(), dirSchemePrefix+"testdata/order", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, "second:4317", retMap.Get("exporters::otlp::endpoint"))
	assert.NoError(t, dp.Shutdown(context.Background()))
}

func TestMergeAppendFeatureGate(t *testing.T) {
	// The gate is defined in confmap/internal/metadata and is not exported.
	// The string ID is the only stable reference available from outside the confmap module.
	const gateID = "confmap.enableMergeAppendOption"

	t.Run("gate off overrides list", func(t *testing.T) {
		dp := createProvider()
		ret, err := dp.Retrieve(context.Background(), dirSchemePrefix+"testdata/lists", nil)
		require.NoError(t, err)
		retMap, err := ret.AsConf()
		require.NoError(t, err)
		assert.Equal(t, []any{"ext1", "ext3"}, retMap.Get("service::extensions"))
		assert.NoError(t, dp.Shutdown(context.Background()))
	})

	t.Run("gate on appends and dedups list", func(t *testing.T) {
		require.NoError(t, featuregate.GlobalRegistry().Set(gateID, true))
		defer func() {
			require.NoError(t, featuregate.GlobalRegistry().Set(gateID, false))
		}()
		dp := createProvider()
		ret, err := dp.Retrieve(context.Background(), dirSchemePrefix+"testdata/lists", nil)
		require.NoError(t, err)
		retMap, err := ret.AsConf()
		require.NoError(t, err)
		assert.Equal(t, []any{"ext1", "ext2", "ext3"}, retMap.Get("service::extensions"))
		assert.NoError(t, dp.Shutdown(context.Background()))
	})
}
