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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

func TestLocalMapProvider(t *testing.T) {
	mp := NewLocal("testdata/default-config.yaml", nil)
	retr, err := mp.Retrieve(context.Background(), nil)
	require.NoError(t, err)

	expectedMap, err := config.NewMapFromBuffer(strings.NewReader(`
processors:
  batch:
exporters:
  otlp:
    endpoint: "localhost:4317"`))
	require.NoError(t, err)
	m, err := retr.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedMap, m)

	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestLocalMapProvider_AddNewConfig(t *testing.T) {
	mp := NewLocal("testdata/default-config.yaml", []string{"processors.batch.timeout=2s"})
	cp, err := mp.Retrieve(context.Background(), nil)
	require.NoError(t, err)

	expectedMap, err := config.NewMapFromBuffer(strings.NewReader(`
processors:
  batch:
    timeout: 2s
exporters:
  otlp:
    endpoint: "localhost:4317"`))
	require.NoError(t, err)
	m, err := cp.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedMap, m)

	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestLocalMapProvider_OverwriteConfig(t *testing.T) {
	mp := NewLocal(
		"testdata/default-config.yaml",
		[]string{"processors.batch.timeout=2s", "exporters.otlp.endpoint=localhost:1234"})
	cp, err := mp.Retrieve(context.Background(), nil)
	require.NoError(t, err)

	expectedMap, err := config.NewMapFromBuffer(strings.NewReader(`
processors:
  batch:
    timeout: 2s
exporters:
  otlp:
    endpoint: "localhost:1234"`))
	require.NoError(t, err)
	m, err := cp.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedMap, m)

	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestLocalMapProvider_InexistentFile(t *testing.T) {
	mp := NewLocal("testdata/otelcol-config.yaml", nil)
	require.NotNil(t, mp)
	_, err := mp.Retrieve(context.Background(), nil)
	require.Error(t, err)

	assert.NoError(t, mp.Shutdown(context.Background()))
}

func TestLocalMapProvider_EmptyFileName(t *testing.T) {
	mp := NewLocal("", nil)
	_, err := mp.Retrieve(context.Background(), nil)
	require.Error(t, err)

	assert.NoError(t, mp.Shutdown(context.Background()))
}
