// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package yamlprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestValidateProviderScheme(t *testing.T) {
	assert.NoError(t, confmaptest.ValidateProviderScheme(createProvider()))
}

func TestEmpty(t *testing.T) {
	sp := createProvider()
	_, err := sp.Retrieve(t.Context(), "", nil)
	require.Error(t, err)
	assert.NoError(t, sp.Shutdown(t.Context()))
}

func TestInvalidYAML(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(t.Context(), "yaml:[invalid,", nil)
	require.NoError(t, err)
	raw, err := ret.AsRaw()
	require.NoError(t, err)
	assert.IsType(t, "", raw)

	assert.NoError(t, sp.Shutdown(t.Context()))
}

func TestOneValue(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(t.Context(), "yaml:processors::batch::timeout: 2s", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"processors": map[string]any{
			"batch": map[string]any{
				"timeout": "2s",
			},
		},
	}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(t.Context()))
}

func TestNamedComponent(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(t.Context(), "yaml:processors::batch/foo::timeout: 3s", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"processors": map[string]any{
			"batch/foo": map[string]any{
				"timeout": "3s",
			},
		},
	}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(t.Context()))
}

func TestMapEntry(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(t.Context(), "yaml:processors: {batch/foo::timeout: 3s, batch::timeout: 2s}", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"processors": map[string]any{
			"batch/foo": map[string]any{
				"timeout": "3s",
			},
			"batch": map[string]any{
				"timeout": "2s",
			},
		},
	}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(t.Context()))
}

func TestArrayEntry(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(t.Context(), "yaml:service::extensions: [zpages, zpages/foo]", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"service": map[string]any{
			"extensions": []any{
				"zpages",
				"zpages/foo",
			},
		},
	}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(t.Context()))
}

func TestNewLine(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(t.Context(), "yaml:processors::batch/foo::timeout: 3s\nprocessors::batch::timeout: 2s", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"processors": map[string]any{
			"batch/foo": map[string]any{
				"timeout": "3s",
			},
			"batch": map[string]any{
				"timeout": "2s",
			},
		},
	}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(t.Context()))
}

func TestDotSeparator(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(t.Context(), "yaml:processors.batch.timeout: 4s", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"processors.batch.timeout": "4s"}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(t.Context()))
}

func createProvider() confmap.Provider {
	return NewFactory().Create(confmaptest.NewNopProviderSettings())
}
