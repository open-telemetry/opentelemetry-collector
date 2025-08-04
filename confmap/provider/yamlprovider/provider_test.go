// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package yamlprovider

import (
	"context"
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
	_, err := sp.Retrieve(context.Background(), "", nil)
	require.Error(t, err)
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestInvalidYAML(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(context.Background(), "yaml:[invalid,", nil)
	require.NoError(t, err)
	raw, err := ret.AsRaw()
	require.NoError(t, err)
	assert.IsType(t, "", raw)

	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestOneValue(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors::test::timeout: 2s", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"processors": map[string]any{
			"test": map[string]any{
				"timeout": "2s",
			},
		},
	}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestNamedComponent(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors::test/foo::timeout: 3s", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"processors": map[string]any{
			"test/foo": map[string]any{
				"timeout": "3s",
			},
		},
	}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestMapEntry(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors: {test/foo::timeout: 3s, test::timeout: 2s}", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"processors": map[string]any{
			"test/foo": map[string]any{
				"timeout": "3s",
			},
			"test": map[string]any{
				"timeout": "2s",
			},
		},
	}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestArrayEntry(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(context.Background(), "yaml:service::extensions: [zpages, zpages/foo]", nil)
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
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestNewLine(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors::test/foo::timeout: 3s\nprocessors::test::timeout: 2s", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"processors": map[string]any{
			"test/foo": map[string]any{
				"timeout": "3s",
			},
			"test": map[string]any{
				"timeout": "2s",
			},
		},
	}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestDotSeparator(t *testing.T) {
	sp := createProvider()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors.test.timeout: 4s", nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"processors.test.timeout": "4s"}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func createProvider() confmap.Provider {
	return NewFactory().Create(confmaptest.NewNopProviderSettings())
}
