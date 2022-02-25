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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

const fileSchemePrefix = fileSchemeName + ":"

func TestFile_EmptyName(t *testing.T) {
	fp := NewFile()
	_, err := fp.Retrieve(context.Background(), "", nil)
	require.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestFile_UnsupportedScheme(t *testing.T) {
	fp := NewFile()
	_, err := fp.Retrieve(context.Background(), "http://", nil)
	assert.Error(t, err)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestFile_NonExistent(t *testing.T) {
	fp := NewFile()
	_, err := fp.Retrieve(context.Background(), fileSchemePrefix+filepath.Join("testdata", "non-existent.yaml"), nil)
	assert.Error(t, err)
	_, err = fp.Retrieve(context.Background(), fileSchemePrefix+absolutePath(t, filepath.Join("testdata", "non-existent.yaml")), nil)
	assert.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestFile_InvalidYaml(t *testing.T) {
	fp := NewFile()
	_, err := fp.Retrieve(context.Background(), fileSchemePrefix+filepath.Join("testdata", "invalid-yaml.yaml"), nil)
	assert.Error(t, err)
	_, err = fp.Retrieve(context.Background(), fileSchemePrefix+absolutePath(t, filepath.Join("testdata", "invalid-yaml.yaml")), nil)
	assert.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestFile_RelativePath(t *testing.T) {
	fp := NewFile()
	ret, err := fp.Retrieve(context.Background(), fileSchemePrefix+filepath.Join("testdata", "default-config.yaml"), nil)
	require.NoError(t, err)
	expectedMap := config.NewMapFromStringMap(map[string]interface{}{
		"processors::batch":         nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	assert.Equal(t, expectedMap, ret.Map)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestFile_AbsolutePath(t *testing.T) {
	fp := NewFile()
	ret, err := fp.Retrieve(context.Background(), fileSchemePrefix+absolutePath(t, filepath.Join("testdata", "default-config.yaml")), nil)
	require.NoError(t, err)
	expectedMap := config.NewMapFromStringMap(map[string]interface{}{
		"processors::batch":         nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	assert.Equal(t, expectedMap, ret.Map)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func absolutePath(t *testing.T, relativePath string) string {
	dir, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(dir, relativePath)
}
