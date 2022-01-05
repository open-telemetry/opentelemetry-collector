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

package configprovider

import (
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

func TestDefaultMapProvider(t *testing.T) {
	mp := NewDefaultMapProvider(path.Join("testdata", "default-config.yaml"), nil)
	retr, err := mp.Retrieve(context.Background(), nil)
	require.NoError(t, err)

	expectedMap := config.NewMapFromStringMap(map[string]interface{}{
		"processors::batch":         nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	m, err := retr.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedMap, m)

	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestDefaultMapProvider_AddNewConfig(t *testing.T) {
	mp := NewDefaultMapProvider(path.Join("testdata", "default-config.yaml"), []string{"processors.batch.timeout=2s"})
	cp, err := mp.Retrieve(context.Background(), nil)
	require.NoError(t, err)

	expectedMap := config.NewMapFromStringMap(map[string]interface{}{
		"processors::batch::timeout": "2s",
		"exporters::otlp::endpoint":  "localhost:4317",
	})
	m, err := cp.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedMap, m)

	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestDefaultMapProvider_OverwriteConfig(t *testing.T) {
	mp := NewDefaultMapProvider(
		path.Join("testdata", "default-config.yaml"),
		[]string{"processors.batch.timeout=2s", "exporters.otlp.endpoint=localhost:1234"})
	cp, err := mp.Retrieve(context.Background(), nil)
	require.NoError(t, err)

	expectedMap := config.NewMapFromStringMap(map[string]interface{}{
		"processors::batch::timeout": "2s",
		"exporters::otlp::endpoint":  "localhost:1234",
	})
	m, err := cp.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedMap, m)

	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestDefaultMapProvider_InexistentFile(t *testing.T) {
	mp := NewDefaultMapProvider(path.Join("testdata", "otelcol-config.yaml"), nil)
	require.NotNil(t, mp)
	_, err := mp.Retrieve(context.Background(), nil)
	require.Error(t, err)

	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestDefaultMapProvider_EmptyFileName(t *testing.T) {
	mp := NewDefaultMapProvider("", nil)
	_, err := mp.Retrieve(context.Background(), nil)
	require.Error(t, err)

	assert.NoError(t, mp.Shutdown(context.Background()))
}
