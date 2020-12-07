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
)

func TestGetAllConfigs(t *testing.T) {
	cfgs := getAllConfigs(testComponents())
	require.NotNil(t, cfgs)
}

func TestCreateAllSchemaFiles(t *testing.T) {
	e := testEnv()
	tempDir := t.TempDir()
	e.yamlFilename = func(t reflect.Type, e env) string {
		return path.Join(tempDir, t.String()+".yaml")
	}
	createAllSchemaFiles(testComponents(), e)
	fileInfos, err := ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.NotNil(t, fileInfos)
	file, err := ioutil.ReadFile(path.Join(tempDir, "otlpexporter.Config.yaml"))
	require.NoError(t, err)
	fld := field{}
	err = yaml.Unmarshal(file, &fld)
	require.NoError(t, err)
	require.Equal(t, "*otlpexporter.Config", fld.Type)
	require.NotNil(t, fld.Fields)
}
