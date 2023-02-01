// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package expandconverter

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			require.NoError(t, New().Convert(context.Background(), conf))
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
	require.NoError(t, New().Convert(context.Background(), conf))

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
	require.NoError(t, New().Convert(context.Background(), conf))
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
			require.NoError(t, New().Convert(context.Background(), conf))
			assert.Equal(t, tt.expected, conf.ToStringMap())
		})
	}
}
