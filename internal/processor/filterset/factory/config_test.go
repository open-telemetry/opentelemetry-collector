// Copyright 2020 OpenTelemetry Authors
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

package factory

import (
	"path"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector/testutils/configtestutils"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	testFile := path.Join(".", "testdata", "config.yaml")
	v, err := configtestutils.CreateViperYamlUnmarshaler(testFile)
	if err != nil {
		t.Errorf("Error configuring viper: %v", err)
	}

	actualConfigs := map[string]MatchConfig{}
	if err = v.UnmarshalExact(&actualConfigs); err != nil {
		t.Errorf("Error unmarshaling yaml from test file %v: %v", testFile, err)
	}

	expectedConfigs := map[string]MatchConfig{
		"regexp/default": MatchConfig{
			MatchType: REGEXP,
		},
		"regexp/emptyoptions": MatchConfig{
			MatchType: REGEXP,
		},
		"regexp/cachedisabledwithsize": MatchConfig{
			MatchType: REGEXP,
			Regexp: &RegexpConfig{
				CacheEnabled:       false,
				CacheMaxNumEntries: 10,
			},
		},
		"regexp/cacheenablednosize": MatchConfig{
			MatchType: REGEXP,
			Regexp: &RegexpConfig{
				CacheEnabled: true,
			},
		},
		"strict/default": MatchConfig{
			MatchType: STRICT,
		},
		"strict/emptyoptions": MatchConfig{
			MatchType: STRICT,
		},
	}

	for testName, actualCfg := range actualConfigs {
		t.Run(testName, func(t *testing.T) {
			expCfg, ok := expectedConfigs[testName]
			assert.True(t, ok)
			assert.Equal(t, expCfg, actualCfg)
		})
	}
}
