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

package configmapprovider

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

func TestEnv_EmptyName(t *testing.T) {
	exp := NewEnv("")
	_, err := exp.Retrieve(context.Background(), nil)
	require.Error(t, err)
}

func TestEnv_InvalidYaml(t *testing.T) {
	bytes, err := os.ReadFile(path.Join("testdata", "invalid-yaml.yaml"))
	require.NoError(t, err)
	const envName = "invalid-yaml"
	require.NoError(t, os.Setenv(envName, string(bytes)))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv(envName))
	})
	env := NewEnv(envName)
	_, err = env.Retrieve(context.Background(), nil)
	assert.Error(t, err)
}

func TestEnv(t *testing.T) {
	bytes, err := os.ReadFile(path.Join("testdata", "default-config.yaml"))
	require.NoError(t, err)
	const envName = "default-config"
	require.NoError(t, os.Setenv(envName, string(bytes)))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv(envName))
	})
	env := NewEnv(envName)
	ret, err := env.Retrieve(context.Background(), nil)
	assert.NoError(t, err)
	cfg, err := ret.Get(context.Background())
	assert.NoError(t, err)
	expectedMap := config.NewMapFromStringMap(map[string]interface{}{
		"processors::batch":         nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	assert.Equal(t, expectedMap, cfg)
	assert.NoError(t, ret.Close(context.Background()))
	assert.NoError(t, env.Shutdown(context.Background()))
}
