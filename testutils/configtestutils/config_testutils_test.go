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

package configtestutils

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	Value    string       `mapstructure:"topvalue"`
	Nested   NestedStruct `mapstructure:"nested"`
	Squashed NestedStruct `mapstructure:",squash"`
}

type NestedStruct struct {
	Value string `mapstructure:"nestedvalue"`
}

func TestCreateViperYamlUnmarshaler(t *testing.T) {
	testFile := path.Join(".", "testdata", "config.yaml")
	v, err := CreateViperYamlUnmarshaler(testFile)
	if err != nil {
		t.Errorf("Error creating viper: %v", err)
	}

	actualConfigs := map[string]TestConfig{}
	if err = v.UnmarshalExact(&actualConfigs); err != nil {
		t.Errorf("Error unmarshaling yaml from test file %v: %v", testFile, err)
	}

	topLevelValue := "toplevelvalue"
	nestedValue := "nestedvalue"
	squashedvalue := "squashedvalue"

	expectedConfigs := map[string]TestConfig{
		"test/fullyaml": {
			Value: topLevelValue,
			Nested: NestedStruct{
				Value: nestedValue,
			},
			Squashed: NestedStruct{
				Value: squashedvalue,
			},
		},
		"test/partialyaml": {
			Value: topLevelValue,
			Squashed: NestedStruct{
				Value: squashedvalue,
			},
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
