// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package envprovider

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/internal/envvar"
)

const envSchemePrefix = schemeName + ":"

const validYAML = `
processors:
  batch:
exporters:
  otlp:
    endpoint: "localhost:4317"
`

func TestValidateProviderScheme(t *testing.T) {
	assert.NoError(t, confmaptest.ValidateProviderScheme(createProvider()))
}

func TestEmptyName(t *testing.T) {
	env := createProvider()
	_, err := env.Retrieve(context.Background(), "", nil)
	require.Error(t, err)
	assert.NoError(t, env.Shutdown(context.Background()))
}

func TestUnsupportedScheme(t *testing.T) {
	env := createProvider()
	_, err := env.Retrieve(context.Background(), "https://", nil)
	assert.Error(t, err)
	assert.NoError(t, env.Shutdown(context.Background()))
}

func TestInvalidYAML(t *testing.T) {
	const envName = "invalid_yaml"
	t.Setenv(envName, "[invalid,")
	env := createProvider()
	ret, err := env.Retrieve(context.Background(), envSchemePrefix+envName, nil)
	require.NoError(t, err)
	raw, err := ret.AsRaw()
	require.NoError(t, err)
	assert.IsType(t, "", raw)
	assert.NoError(t, env.Shutdown(context.Background()))
}

func TestEnv(t *testing.T) {
	const envName = "default_config"
	t.Setenv(envName, validYAML)

	env := createProvider()
	ret, err := env.Retrieve(context.Background(), envSchemePrefix+envName, nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
	expectedMap := confmap.NewFromStringMap(map[string]any{
		"processors::batch":         nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	assert.Equal(t, expectedMap.ToStringMap(), retMap.ToStringMap())

	assert.NoError(t, env.Shutdown(context.Background()))
}

func TestEnvWithLogger(t *testing.T) {
	const envName = "default_config"
	t.Setenv(envName, validYAML)
	core, ol := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	env := NewFactory().Create(confmap.ProviderSettings{Logger: logger})
	ret, err := env.Retrieve(context.Background(), envSchemePrefix+envName, nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
	expectedMap := confmap.NewFromStringMap(map[string]any{
		"processors::batch":         nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	assert.Equal(t, expectedMap.ToStringMap(), retMap.ToStringMap())

	assert.NoError(t, env.Shutdown(context.Background()))
	assert.Equal(t, 0, ol.Len())
}

func TestUnsetEnvWithLoggerWarn(t *testing.T) {
	const envName = "default_config"
	core, ol := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	env := NewFactory().Create(confmap.ProviderSettings{Logger: logger})
	ret, err := env.Retrieve(context.Background(), envSchemePrefix+envName, nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
	expectedMap := confmap.NewFromStringMap(map[string]any{})
	assert.Equal(t, expectedMap.ToStringMap(), retMap.ToStringMap())

	assert.NoError(t, env.Shutdown(context.Background()))

	assert.Equal(t, 1, ol.Len())
	logLine := ol.All()[0]
	assert.Equal(t, "Configuration references unset environment variable", logLine.Message)
	assert.Equal(t, zap.WarnLevel, logLine.Level)
	assert.Equal(t, envName, logLine.Context[0].String)
}

func TestEnvVarNameRestriction(t *testing.T) {
	const envName = "default%config"
	t.Setenv(envName, validYAML)

	env := createProvider()
	ret, err := env.Retrieve(context.Background(), envSchemePrefix+envName, nil)
	assert.Equal(t, err, fmt.Errorf("environment variable \"default%%config\" has invalid name: must match regex %s", envvar.ValidationRegexp))
	assert.NoError(t, env.Shutdown(context.Background()))
	assert.Nil(t, ret)
}

func TestEmptyEnvWithLoggerWarn(t *testing.T) {
	const envName = "default_config"
	t.Setenv(envName, "")

	core, ol := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	env := NewFactory().Create(confmap.ProviderSettings{Logger: logger})
	ret, err := env.Retrieve(context.Background(), envSchemePrefix+envName, nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
	expectedMap := confmap.NewFromStringMap(map[string]any{})
	assert.Equal(t, expectedMap.ToStringMap(), retMap.ToStringMap())

	assert.NoError(t, env.Shutdown(context.Background()))

	assert.Equal(t, 1, ol.Len())
	logLine := ol.All()[0]
	assert.Equal(t, "Configuration references empty environment variable", logLine.Message)
	assert.Equal(t, zap.InfoLevel, logLine.Level)
	assert.Equal(t, envName, logLine.Context[0].String)
}

func TestEnvWithDefaultValue(t *testing.T) {
	env := createProvider()
	tests := []struct {
		name        string
		unset       bool
		value       string
		uri         string
		expectedVal string
		expectedErr string
	}{
		{name: "unset", unset: true, uri: "env:MY_VAR:-default % value", expectedVal: "default % value"},
		{name: "unset2", unset: true, uri: "env:MY_VAR:-", expectedVal: ""}, // empty default still applies
		{name: "empty", value: "", uri: "env:MY_VAR:-foo", expectedVal: ""},
		{name: "not empty", value: "value", uri: "env:MY_VAR:-", expectedVal: "value"},
		{name: "syntax1", unset: true, uri: "env:-MY_VAR", expectedErr: "invalid name"},
		{name: "syntax2", unset: true, uri: "env:MY_VAR:-test:-test", expectedVal: "test:-test"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !test.unset {
				t.Setenv("MY_VAR", test.value)
			}
			ret, err := env.Retrieve(context.Background(), test.uri, nil)
			if test.expectedErr != "" {
				require.ErrorContains(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err)
			str, err := ret.AsString()
			require.NoError(t, err)
			assert.Equal(t, test.expectedVal, str)
		})
	}
	assert.NoError(t, env.Shutdown(context.Background()))
}

func createProvider() confmap.Provider {
	return NewFactory().Create(confmaptest.NewNopProviderSettings())
}
