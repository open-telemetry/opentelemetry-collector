// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package expandconverter

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestNewExpandConverter(t *testing.T) {
	var testCases = []struct {
		name string // test case name (also file name containing config yaml)
	}{
		{name: "expand-with-no-env.yaml"},
		{name: "expand-with-partial-env.yaml"},
		{name: "expand-with-all-env.yaml"},
	}

	const valueExtra = "some string"
	const valueExtraMapValue = "some map value"
	const valueExtraListMapValue = "some list map value"
	const valueExtraListElement = "some list value"
	t.Setenv("EXTRA", valueExtra)
	t.Setenv("EXTRA_MAP_VALUE_1", valueExtraMapValue+"_1")
	t.Setenv("EXTRA_MAP_VALUE_2", valueExtraMapValue+"_2")
	t.Setenv("EXTRA_LIST_MAP_VALUE_1", valueExtraListMapValue+"_1")
	t.Setenv("EXTRA_LIST_MAP_VALUE_2", valueExtraListMapValue+"_2")
	t.Setenv("EXTRA_LIST_VALUE_1", valueExtraListElement+"_1")
	t.Setenv("EXTRA_LIST_VALUE_2", valueExtraListElement+"_2")

	expectedCfgMap, errExpected := confmaptest.LoadConf(filepath.Join("testdata", "expand-with-no-env.yaml"))
	require.NoError(t, errExpected, "Unable to get expected config")

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			conf, err := confmaptest.LoadConf(filepath.Join("testdata", test.name))
			require.NoError(t, err, "Unable to get config")

			// Test that expanded configs are the same with the simple config with no env vars.
			require.NoError(t, createConverter().Convert(context.Background(), conf))
			assert.Equal(t, expectedCfgMap.ToStringMap(), conf.ToStringMap())
		})
	}
}

func TestNewExpandConverter_EscapedMaps(t *testing.T) {
	const receiverExtraMapValue = "some map value"
	t.Setenv("MAP_VALUE", receiverExtraMapValue)

	conf := confmap.NewFromStringMap(
		map[string]any{
			"test_string_map": map[string]any{
				"recv": "$MAP_VALUE",
			},
			"test_interface_map": map[any]any{
				"recv": "$MAP_VALUE",
			}},
	)
	require.NoError(t, createConverter().Convert(context.Background(), conf))

	expectedMap := map[string]any{
		"test_string_map": map[string]any{
			"recv": receiverExtraMapValue,
		},
		"test_interface_map": map[string]any{
			"recv": receiverExtraMapValue,
		}}
	assert.Equal(t, expectedMap, conf.ToStringMap())
}

func TestNewExpandConverter_EscapedEnvVars(t *testing.T) {
	const receiverExtraMapValue = "some map value"
	t.Setenv("MAP_VALUE_2", receiverExtraMapValue)

	// Retrieve the config
	conf, err := confmaptest.LoadConf(filepath.Join("testdata", "expand-escaped-env.yaml"))
	require.NoError(t, err, "Unable to get config")

	expectedMap := map[string]any{
		"test_map": map[string]any{
			// $$ -> escaped $
			"recv.1": "$MAP_VALUE_1",
			// $$$ -> escaped $ + substituted env var
			"recv.2": "$" + receiverExtraMapValue,
			// $$$$ -> two escaped $
			"recv.3": "$$MAP_VALUE_3",
			// escaped $ in the middle
			"recv.4": "some${MAP_VALUE_4}text",
			// $$$$ -> two escaped $
			"recv.5": "${ONE}${TWO}",
			// trailing escaped $
			"recv.6": "text$",
			// escaped $ alone
			"recv.7": "$",
		}}
	require.NoError(t, createConverter().Convert(context.Background(), conf))
	assert.Equal(t, expectedMap, conf.ToStringMap())
}

func TestNewExpandConverterHostPort(t *testing.T) {
	t.Setenv("HOST", "127.0.0.1")
	t.Setenv("PORT", "4317")

	var testCases = []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "brackets",
			input: map[string]any{
				"test": "${HOST}:${PORT}",
			},
			expected: map[string]any{
				"test": "127.0.0.1:4317",
			},
		},
		{
			name: "no brackets",
			input: map[string]any{
				"test": "$HOST:$PORT",
			},
			expected: map[string]any{
				"test": "127.0.0.1:4317",
			},
		},
		{
			name: "mix",
			input: map[string]any{
				"test": "${HOST}:$PORT",
			},
			expected: map[string]any{
				"test": "127.0.0.1:4317",
			},
		},
		{
			name: "reverse mix",
			input: map[string]any{
				"test": "$HOST:${PORT}",
			},
			expected: map[string]any{
				"test": "127.0.0.1:4317",
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			conf := confmap.NewFromStringMap(tt.input)
			require.NoError(t, createConverter().Convert(context.Background(), conf))
			assert.Equal(t, tt.expected, conf.ToStringMap())
		})
	}
}

func NewTestConverter() (confmap.Converter, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.InfoLevel)
	conv := converter{loggedDeprecations: make(map[string]struct{}), logger: zap.New(core)}
	return conv, logs
}

func TestDeprecatedWarning(t *testing.T) {
	msgTemplate := `Variable substitution using $VAR will be deprecated in favor of ${VAR} and ${env:VAR}, please update $%s`
	t.Setenv("HOST", "127.0.0.1")
	t.Setenv("PORT", "4317")

	t.Setenv("HOST_NAME", "127.0.0.2")
	t.Setenv("HOST.NAME", "127.0.0.3")

	var testCases = []struct {
		name             string
		input            map[string]any
		expectedOutput   map[string]any
		expectedWarnings []string
	}{
		{
			name: "no warning",
			input: map[string]any{
				"test": "${HOST}:${PORT}",
			},
			expectedOutput: map[string]any{
				"test": "127.0.0.1:4317",
			},
			expectedWarnings: []string{},
		},
		{
			name: "one deprecated var",
			input: map[string]any{
				"test": "${HOST}:$PORT",
			},
			expectedOutput: map[string]any{
				"test": "127.0.0.1:4317",
			},
			expectedWarnings: []string{"PORT"},
		},
		{
			name: "two deprecated vars",
			input: map[string]any{
				"test": "$HOST:$PORT",
			},
			expectedOutput: map[string]any{
				"test": "127.0.0.1:4317",
			},
			expectedWarnings: []string{"HOST", "PORT"},
		},
		{
			name: "one depracated serveral times",
			input: map[string]any{
				"test":  "$HOST,$HOST",
				"test2": "$HOST",
			},
			expectedOutput: map[string]any{
				"test":  "127.0.0.1,127.0.0.1",
				"test2": "127.0.0.1",
			},
			expectedWarnings: []string{"HOST"},
		},
		{
			name: "one warning",
			input: map[string]any{
				"test": "$HOST_NAME,${HOST.NAME}",
			},
			expectedOutput: map[string]any{
				"test": "127.0.0.2,127.0.0.3",
			},
			expectedWarnings: []string{"HOST_NAME"},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			conf := confmap.NewFromStringMap(tt.input)
			conv, logs := NewTestConverter()
			require.NoError(t, conv.Convert(context.Background(), conf))

			assert.Equal(t, tt.expectedOutput, conf.ToStringMap())
			assert.Equal(t, len(tt.expectedWarnings), len(logs.All()))
			for i, variable := range tt.expectedWarnings {
				errorMsg := fmt.Sprintf(msgTemplate, variable)
				assert.Equal(t, errorMsg, logs.All()[i].Message)
			}
		})
	}
}

func createConverter() confmap.Converter {
	return NewFactory().Create(confmap.ConverterSettings{})
}
