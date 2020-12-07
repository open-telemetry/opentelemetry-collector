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

package schemagen

import (
	"io/ioutil"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/defaultcomponents"
)

func TestCreateReceiverConfig(t *testing.T) {
	cfg, err := getConfig(testComponents(), "receiver", "otlp")
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestCreateProcesorConfig(t *testing.T) {
	cfg, err := getConfig(testComponents(), "processor", "filter")
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
			cfg, err := getConfig(testComponents(), test.componentType, test.name)
			require.NoError(t, err)
			require.NotNil(t, cfg)
		})
	}
}

func TestCreateSingleSchemaFile(t *testing.T) {
	e := testEnv()
	tempDir := t.TempDir()
	e.yamlFilename = func(reflect.Type, env) string {
		return path.Join(tempDir, schemaFilename)
	}
	createSingleSchemaFile(testComponents(), "exporter", "otlp", e)
	file, err := ioutil.ReadFile(path.Join(tempDir, schemaFilename))
	require.NoError(t, err)
	fld := field{}
	err = yaml.Unmarshal(file, &fld)
	require.NoError(t, err)
	require.Equal(t, "*otlpexporter.Config", fld.Type)
	require.NotNil(t, fld.Fields)
}

func testComponents() component.Factories {
	components, err := defaultcomponents.Components()
	if err != nil {
		panic(err)
	}
	return components
}
