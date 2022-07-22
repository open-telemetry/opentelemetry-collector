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
	"go.opentelemetry.io/collector/internal/nonfatalerror"
	"go.opentelemetry.io/collector/service/featuregate"
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
		map[string]interface{}{
			"test_string_map": map[string]interface{}{
				"recv": "$MAP_VALUE",
			},
			"test_interface_map": map[interface{}]interface{}{
				"recv": "$MAP_VALUE",
			}},
	)
	require.NoError(t, New().Convert(context.Background(), conf))

	expectedMap := map[string]interface{}{
		"test_string_map": map[string]interface{}{
			"recv": receiverExtraMapValue,
		},
		"test_interface_map": map[string]interface{}{
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

	expectedMap := map[string]interface{}{
		"test_map": map[string]interface{}{
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

func TestMissingEnvVar(t *testing.T) {
	const receiverExtraMapValue = "some map value"
	t.Setenv("MAP_VALUE", receiverExtraMapValue)
	sourceMap := map[string]interface{}{
		"test_string_map": map[string]interface{}{
			"recv":    []interface{}{map[string]interface{}{"field": "$MAP_VALUE", "unknown": "$UNKNOWN_1"}},
			"unknown": "$UNKNOWN_1-$UNKNOWN_2-$UNKNOWN_3",
		},
	}

	tests := []struct {
		name            string
		registryBuilder func() *featuregate.Registry
		expectedMap     map[string]interface{}
		expectedErr     string
		isNonFatal      bool
	}{
		{
			name: "no fail on unknown environment variable",
			registryBuilder: func() *featuregate.Registry {
				registry := featuregate.NewRegistry()
				registry.MustRegister(raiseErrorOnUnknownEnvVarFeatureGate)
				registry.MustApply(map[string]bool{
					raiseErrorOnUnknownEnvVarFeatureGateID: false,
				})
				return registry
			},
			expectedMap: map[string]interface{}{
				"test_string_map": map[string]interface{}{
					"recv":    []interface{}{map[string]interface{}{"field": receiverExtraMapValue, "unknown": ""}},
					"unknown": "--",
				},
			},
			expectedErr: "Non fatal error: failed to expand unknown environment variable(s): [UNKNOWN_1 UNKNOWN_2 UNKNOWN_3]. " +
				"Use \"confmap.expandconverter.RaiseErrorOnUnknownEnvVar\" to turn this into a fatal error",
			isNonFatal: true,
		},
		{
			name: "fail on environment variable",
			registryBuilder: func() *featuregate.Registry {
				registry := featuregate.NewRegistry()
				registry.MustRegister(raiseErrorOnUnknownEnvVarFeatureGate)
				registry.MustApply(map[string]bool{
					raiseErrorOnUnknownEnvVarFeatureGateID: true,
				})
				return registry
			},
			expectedErr: "failed to expand unknown environment variable(s): [UNKNOWN_1 UNKNOWN_2 UNKNOWN_3]",
			isNonFatal:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := tt.registryBuilder()
			converter := newWithRegistry(registry)
			conf := confmap.NewFromStringMap(sourceMap)

			err := converter.Convert(context.Background(), conf)
			if err != nil || tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Equal(t, tt.isNonFatal, nonfatalerror.IsNonFatal(err))
			} else {
				assert.Equal(t, tt.expectedMap, conf.ToStringMap())
			}
		})
	}
}
