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

package filtermetric

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset/regexp"
	"github.com/open-telemetry/opentelemetry-collector/testutils/configtestutils"
)

var (
	// regexpNameMatches matches the metrics names specified in testdata/config.yaml
	regexpNameMatches = []string{
		"prefix/.*",
		".*contains.*",
		".*_suffix",
		"full_name_match",
	}

	strictNameMatches = []string{
		"exact_string_match",
	}
)

func createConfigWithRegexpOptions(filters []string, rCfg *regexp.Config) *MatchProperties {
	cfg := createConfig(filters, filterset.Regexp)
	cfg.Config.RegexpConfig = rCfg
	return cfg
}

func TestConfig(t *testing.T) {
	testFile := path.Join(".", "testdata", "config.yaml")
	v, err := configtestutils.CreateViperYamlUnmarshaler(testFile)
	if err != nil {
		t.Errorf("Error creating Viper config loader: %v", err)
	}

	testYamls := map[string]MatchProperties{}
	if err = v.UnmarshalExact(&testYamls); err != nil {
		t.Errorf("Error unmarshaling yaml from test file %v: %v", testFile, err)
	}

	tests := []struct {
		name   string
		expCfg *MatchProperties
	}{
		{
			name:   "config/regexp",
			expCfg: createConfig(regexpNameMatches, filterset.Regexp),
		}, {
			name: "config/regexpoptions",
			expCfg: createConfigWithRegexpOptions(
				regexpNameMatches,
				&regexp.Config{
					CacheEnabled:       true,
					CacheMaxNumEntries: 5,
				},
			),
		}, {
			name:   "config/strict",
			expCfg: createConfig(strictNameMatches, filterset.Strict),
		}, {
			name:   "config/emptyproperties",
			expCfg: createConfig(nil, filterset.Regexp),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := testYamls[test.name]
			assert.Equal(t, *test.expCfg, cfg)

			matcher, err := NewMatcher(&cfg)
			assert.NotNil(t, matcher)
			assert.Nil(t, err)
		})
	}
}
