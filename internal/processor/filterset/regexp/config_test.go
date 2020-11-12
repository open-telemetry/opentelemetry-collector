// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package regexp

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configtest"
)

func TestConfig(t *testing.T) {
	testFile := path.Join(".", "testdata", "config.yaml")
	v := configtest.NewViperFromYamlFile(t, testFile)

	actualConfigs := map[string]*Config{}
	require.NoErrorf(t, v.UnmarshalExact(&actualConfigs), "unable to unmarshal yaml from file %v", testFile)

	expectedConfigs := map[string]*Config{
		"regexp/default": {},
		"regexp/cachedisabledwithsize": {
			CacheEnabled:       false,
			CacheMaxNumEntries: 10,
		},
		"regexp/cacheenablednosize": {
			CacheEnabled: true,
		},
	}

	for testName, actualCfg := range actualConfigs {
		t.Run(testName, func(t *testing.T) {
			expCfg, ok := expectedConfigs[testName]
			assert.True(t, ok)
			assert.Equal(t, expCfg, actualCfg)

			fs, err := NewFilterSet([]string{}, actualCfg)
			assert.NoError(t, err)
			assert.NotNil(t, fs)
		})
	}
}
