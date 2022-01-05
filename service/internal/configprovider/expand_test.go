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

package configprovider

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/config/configtest"
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
	const valueExtraListElement = "some list value"
	assert.NoError(t, os.Setenv("EXTRA", valueExtra))
	assert.NoError(t, os.Setenv("EXTRA_MAP_VALUE_1", valueExtraMapValue+"_1"))
	assert.NoError(t, os.Setenv("EXTRA_MAP_VALUE_2", valueExtraMapValue+"_2"))
	assert.NoError(t, os.Setenv("EXTRA_LIST_VALUE_1", valueExtraListElement+"_1"))
	assert.NoError(t, os.Setenv("EXTRA_LIST_VALUE_2", valueExtraListElement+"_2"))

	defer func() {
		assert.NoError(t, os.Unsetenv("EXTRA"))
		assert.NoError(t, os.Unsetenv("EXTRA_MAP_VALUE"))
		assert.NoError(t, os.Unsetenv("EXTRA_LIST_VALUE_1"))
	}()

	// Cannot use configtest.LoadConfigMap because of circular deps.
	ret, errRet := configmapprovider.NewFile(path.Join("testdata", "expand-with-no-env.yaml")).Retrieve(context.Background(), nil)
	require.NoError(t, errRet, "Unable to get expected config")
	expectedCfgMap, errExpected := ret.Get(context.Background())
	require.NoError(t, errExpected, "Unable to get expected config")

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cfgMap, err := configtest.LoadConfigMap(path.Join("testdata", test.name))
			require.NoError(t, err, "Unable to get config")

			// Test that expanded configs are the same with the simple config with no env vars.
			require.NoError(t, NewExpandConverter()(cfgMap))
			assert.Equal(t, expectedCfgMap.ToStringMap(), cfgMap.ToStringMap())
		})
	}
}

func TestNewExpandConverter_EscapedEnvVars(t *testing.T) {
	const receiverExtraMapValue = "some map value"
	assert.NoError(t, os.Setenv("MAP_VALUE_2", receiverExtraMapValue))
	defer func() {
		assert.NoError(t, os.Unsetenv("MAP_VALUE_2"))
	}()

	// Retrieve the config
	cfgMap, err := configtest.LoadConfigMap(path.Join("testdata", "expand-escaped-env.yaml"))
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
	require.NoError(t, NewExpandConverter()(cfgMap))
	assert.Equal(t, expectedMap, cfgMap.ToStringMap())
}
