// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package yamlprovider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestValidateProviderScheme(t *testing.T) {
	assert.NoError(t, confmaptest.ValidateProviderScheme(New()))
}

func TestEmpty(t *testing.T) {
	sp := New()
	_, err := sp.Retrieve(context.Background(), "", nil)
	assert.Error(t, err)
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestInvalidYAML(t *testing.T) {
	sp := New()
	_, err := sp.Retrieve(context.Background(), "yaml:[invalid,", nil)
	assert.Error(t, err)
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestOneValue(t *testing.T) {
	sp := New()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors::batch::timeout: 2s", nil)
	assert.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"processors": map[string]any{
			"batch": map[string]any{
				"timeout": "2s",
			},
		},
	}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestNamedComponent(t *testing.T) {
	sp := New()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors::batch/foo::timeout: 3s", nil)
	assert.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"processors": map[string]any{
			"batch/foo": map[string]any{
				"timeout": "3s",
			},
		},
	}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestMapEntry(t *testing.T) {
	sp := New()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors: {batch/foo::timeout: 3s, batch::timeout: 2s}", nil)
	assert.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
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
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestArrayEntry(t *testing.T) {
	sp := New()
	ret, err := sp.Retrieve(context.Background(), "yaml:service::extensions: [zpages, zpages/foo]", nil)
	assert.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
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
	sp := New()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors::batch/foo::timeout: 3s\nprocessors::batch::timeout: 2s", nil)
	assert.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
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
	assert.NoError(t, sp.Shutdown(context.Background()))
}

func TestDotSeparator(t *testing.T) {
	sp := New()
	ret, err := sp.Retrieve(context.Background(), "yaml:processors.batch.timeout: 4s", nil)
	assert.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"processors.batch.timeout": "4s"}, retMap.ToStringMap())
	assert.NoError(t, sp.Shutdown(context.Background()))
}
