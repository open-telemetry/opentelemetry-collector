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

func TestGetAllConfigs(t *testing.T) {
	cfgs := getAllConfigs()
	require.NotNil(t, cfgs)
}

func TestCreateAllSchemaFiles(t *testing.T) {
	env := testEnv()
	tempDir := t.TempDir()
	env.YamlFilename = func(t reflect.Type, env Env) string {
		return path.Join(tempDir, t.String()+".yaml")
	}
	CreateAllSchemaFiles(env)
	fileInfos, err := ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.NotNil(t, fileInfos)
	file, err := ioutil.ReadFile(path.Join(tempDir, "otlpexporter.Config.yaml"))
	require.NoError(t, err)
	field := field{}
	err = yaml.Unmarshal(file, &field)
	require.NoError(t, err)
	require.Equal(t, "*otlpexporter.Config", field.Type)
	require.NotNil(t, field.Fields)
}
