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

package envmapprovider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

const envSchemePrefix = schemeName + ":"

func TestEmptyName(t *testing.T) {
	env := New()
	_, err := env.Retrieve(context.Background(), "", nil)
	require.Error(t, err)
	assert.NoError(t, env.Shutdown(context.Background()))
}

func TestUnsupportedScheme(t *testing.T) {
	env := New()
	_, err := env.Retrieve(context.Background(), "http://", nil)
	assert.Error(t, err)
	assert.NoError(t, env.Shutdown(context.Background()))
}

func TestInvalidYAML(t *testing.T) {
	bytes, err := os.ReadFile(filepath.Join("testdata", "invalid-yaml.yaml"))
	require.NoError(t, err)
	const envName = "invalid-yaml"
	t.Setenv(envName, string(bytes))
	env := New()
	_, err = env.Retrieve(context.Background(), envSchemePrefix+envName, nil)
	assert.Error(t, err)
	assert.NoError(t, env.Shutdown(context.Background()))
}

func TestEnv(t *testing.T) {
	bytes, err := os.ReadFile(filepath.Join("testdata", "default-config.yaml"))
	require.NoError(t, err)
	const envName = "default-config"
	t.Setenv(envName, string(bytes))

	env := New()
	ret, err := env.Retrieve(context.Background(), envSchemePrefix+envName, nil)
	require.NoError(t, err)
	expectedMap := config.NewMapFromStringMap(map[string]interface{}{
		"processors::batch":         nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	assert.Equal(t, expectedMap.ToStringMap(), ret.Map.ToStringMap())

	assert.NoError(t, env.Shutdown(context.Background()))
}
