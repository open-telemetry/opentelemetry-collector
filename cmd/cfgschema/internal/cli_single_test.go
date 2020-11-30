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

package internal

import (
	"io/ioutil"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestCreateReceiverConfig(t *testing.T) {
	cfg, err := getConfig("receiver", "otlp")
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestCreateProcesorConfig(t *testing.T) {
	cfg, err := getConfig("processor", "filter")
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name          string
		componentType string
	}{
		{
			name:          "otlp",
			componentType: "receiver",
		},
		{
			name:          "filter",
			componentType: "processor",
		},
		{
			name:          "otlp",
			componentType: "exporter",
		},
		{
			name:          "zpages",
			componentType: "extension",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := getConfig(test.componentType, test.name)
			require.NoError(t, err)
			require.NotNil(t, cfg)
		})
	}
}

func TestCreateSingleConfigSchema(t *testing.T) {
	env := testEnv()
	tempDir := t.TempDir()
	env.GetTargetYamlDir = func(reflect.Type, Env) string {
		return tempDir
	}
	CreateSingleCfgSchemaFile("exporter", "otlp", env)
	file, err := ioutil.ReadFile(path.Join(tempDir, schemaFile))
	require.NoError(t, err)
	field := field{}
	err = yaml.Unmarshal(file, &field)
	require.NoError(t, err)
	require.Equal(t, "*otlpexporter.Config", field.Type)
	require.NotNil(t, field.Fields)
}
