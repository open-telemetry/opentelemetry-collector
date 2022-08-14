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

package fileprovider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

const fileSchemePrefix = schemeName + ":"

func TestValidateProviderScheme(t *testing.T) {
	assert.NoError(t, confmaptest.ValidateProviderScheme(New()))
}

func TestEmptyName(t *testing.T) {
	fp := New()
	_, err := fp.Retrieve(context.Background(), "", nil)
	require.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestUnsupportedScheme(t *testing.T) {
	fp := New()
	_, err := fp.Retrieve(context.Background(), "https://", nil)
	assert.Error(t, err)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestNonExistent(t *testing.T) {
	fp := New()
	_, err := fp.Retrieve(context.Background(), fileSchemePrefix+filepath.Join("testdata", "non-existent.yaml"), nil)
	assert.Error(t, err)
	_, err = fp.Retrieve(context.Background(), fileSchemePrefix+absolutePath(t, filepath.Join("testdata", "non-existent.yaml")), nil)
	assert.Error(t, err)
	watchFunc := func(_ *confmap.ChangeEvent) {}
	_, err = fp.Retrieve(context.Background(), fileSchemePrefix+filepath.Join("testdata", "non-existent.yaml"), watchFunc)
	assert.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestInvalidYAML(t *testing.T) {
	fp := New()
	_, err := fp.Retrieve(context.Background(), fileSchemePrefix+filepath.Join("testdata", "invalid-yaml.yaml"), nil)
	assert.Error(t, err)
	_, err = fp.Retrieve(context.Background(), fileSchemePrefix+absolutePath(t, filepath.Join("testdata", "invalid-yaml.yaml")), nil)
	assert.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestRelativePath(t *testing.T) {
	fp := New()
	ret, err := fp.Retrieve(context.Background(), fileSchemePrefix+filepath.Join("testdata", "default-config.yaml"), nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
	expectedMap := confmap.NewFromStringMap(map[string]interface{}{
		"processors::batch":         nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	assert.Equal(t, expectedMap, retMap)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestAbsolutePath(t *testing.T) {
	fp := New()
	ret, err := fp.Retrieve(context.Background(), fileSchemePrefix+absolutePath(t, filepath.Join("testdata", "default-config.yaml")), nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
	expectedMap := confmap.NewFromStringMap(map[string]interface{}{
		"processors::batch":         nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	assert.Equal(t, expectedMap, retMap)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestWatcherStartAndShutdown(t *testing.T) {
	fp := New()
	watchFunc := func(_ *confmap.ChangeEvent) {}
	ret, err := fp.Retrieve(context.Background(), fileSchemePrefix+absolutePath(t, filepath.Join("testdata", "default-config.yaml")), watchFunc)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	assert.NoError(t, err)
	expectedMap := confmap.NewFromStringMap(map[string]interface{}{
		"processors::batch":         nil,
		"exporters::otlp::endpoint": "localhost:4317",
	})
	assert.Equal(t, expectedMap, retMap)
	assert.NoError(t, ret.Close(context.Background()))
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestWatcherShutdownBeforeClose(t *testing.T) {
	fp := New()
	watchFunc := func(_ *confmap.ChangeEvent) {}
	ret, err := fp.Retrieve(context.Background(), fileSchemePrefix+absolutePath(t, filepath.Join("testdata", "default-config.yaml")), watchFunc)
	require.NoError(t, err)
	assert.NoError(t, fp.Shutdown(context.Background()))
	assert.NoError(t, ret.Close(context.Background()))
}

func TestWatcherConfigChange(t *testing.T) {
	fp := New().(*provider)
	fp.filePollInterval = time.Millisecond

	events := make(chan *confmap.ChangeEvent)
	watchFunc := func(event *confmap.ChangeEvent) {
		events <- event
	}

	absolutePath := absolutePath(t, filepath.Join("testdata", "default-config.yaml"))
	testFile := copyTestFile(t, absolutePath)

	ret, err := fp.Retrieve(context.Background(), fileSchemePrefix+testFile.Name(), watchFunc)
	require.NoError(t, err)

	// change the file, we should get an update event
	_, err = testFile.WriteString("\n")
	require.NoError(t, err)
	event := getEventWithTimeout(events, fp.filePollInterval*100)
	require.NotNil(t, event)
	assert.Nil(t, event.Error)

	// delete the file, we should get an error
	err = testFile.Close()
	require.NoError(t, err)
	err = os.Remove(testFile.Name())
	require.NoError(t, err)
	event = getEventWithTimeout(events, fp.filePollInterval*100)
	require.NotNil(t, event)
	assert.Error(t, event.Error)

	assert.NoError(t, ret.Close(context.Background()))
	require.NoError(t, fp.Shutdown(context.Background()))
}

func absolutePath(t *testing.T, relativePath string) string {
	dir, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(dir, relativePath)
}

func copyTestFile(t *testing.T, path string) *os.File {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	require.NoError(t, err)
	tempFile, err := os.CreateTemp(t.TempDir(), "*")
	require.NoError(t, err)
	_, err = tempFile.Write(data)
	require.NoError(t, err)
	return tempFile
}
