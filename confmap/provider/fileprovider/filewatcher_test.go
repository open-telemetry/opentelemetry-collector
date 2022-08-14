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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

func TestStartClose(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*")
	require.NoError(t, err)
	defer f.Close()
	watcher := newPollingFileWatcher(f.Name(), time.Second)

	// start and then close
	assert.NoError(t, watcher.Start())
	watcher.Close()

	// start again twice
	assert.NoError(t, watcher.Start())
	assert.NoError(t, watcher.Start())

	// and then close again twice
	watcher.Close()
	watcher.Close()
}

func TestCloseEvenIfBlockedOnEventChannel(t *testing.T) {
	pollInterval := time.Millisecond
	fileContent := "content"
	f, err := os.CreateTemp(t.TempDir(), "*")
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(fileContent)
	require.NoError(t, err)

	watcher := newPollingFileWatcher(f.Name(), pollInterval)
	require.NoError(t, watcher.Start())

	// write to the file, we should get an event, but we don't receive it, and just close the watcher
	_, err = f.WriteString("\n")
	require.NoError(t, err)
	time.Sleep(pollInterval * 2)
	watcher.Close()
}

func TestNoFile(t *testing.T) {
	watcher := newPollingFileWatcher(filepath.Join(t.TempDir(), "nonexistent"), time.Second)

	// start returns an error
	assert.Error(t, watcher.Start())
}

func TestNoEventsIfNoChanges(t *testing.T) {
	pollInterval := time.Millisecond
	fileContent := "content"
	f, err := os.CreateTemp(t.TempDir(), "*")
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(fileContent)
	require.NoError(t, err)

	watcher := newPollingFileWatcher(f.Name(), pollInterval)
	require.NoError(t, watcher.Start())
	defer watcher.Close()
	assert.Nil(t, getEventWithTimeout(watcher.Events, pollInterval*2))

	// change the access times, this doesn't change the content, but does change the metadata
	require.NoError(t, os.Chtimes(f.Name(), time.Now(), time.Now()))
	assert.Nil(t, getEventWithTimeout(watcher.Events, pollInterval*2))
}

func TestEventIfFileContentChanged(t *testing.T) {
	pollInterval := time.Millisecond
	fileContent := "content"
	f, err := os.CreateTemp(t.TempDir(), "*")
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(fileContent)
	require.NoError(t, err)

	watcher := newPollingFileWatcher(f.Name(), pollInterval)
	require.NoError(t, watcher.Start())
	defer watcher.Close()
	assert.Nil(t, getEventWithTimeout(watcher.Events, pollInterval*100))

	// write to the file, we should get an event
	_, err = f.WriteString("\n")
	require.NoError(t, err)
	event := getEventWithTimeout(watcher.Events, pollInterval*100)
	require.NotNil(t, event)
	require.Nil(t, event.Error)
}

func TestEventIfFileRemoved(t *testing.T) {
	pollInterval := time.Millisecond
	fileContent := "content"
	f, err := os.CreateTemp(t.TempDir(), "*")
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(fileContent)
	require.NoError(t, err)

	watcher := newPollingFileWatcher(f.Name(), pollInterval)
	require.NoError(t, watcher.Start())
	defer watcher.Close()
	assert.Nil(t, getEventWithTimeout(watcher.Events, pollInterval*100))

	// remove the file, we should get an error event
	err = f.Close()
	require.NoError(t, err)
	err = os.Remove(f.Name())
	require.NoError(t, err)
	event := getEventWithTimeout(watcher.Events, pollInterval*100)
	require.NotNil(t, event)
	require.NotNil(t, event.Error)
}

func BenchmarkCheckFileForChangesNoChange(b *testing.B) {
	fileContent := "content"
	f, err := os.CreateTemp(b.TempDir(), "*")
	require.NoError(b, err)
	defer f.Close()
	_, err = f.WriteString(fileContent)
	require.NoError(b, err)

	watcher := newPollingFileWatcher(f.Name(), time.Hour)
	require.NoError(b, watcher.Start())
	defer watcher.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.checkFileForChanges()
	}
}

func BenchmarkCheckFileForChangesMetadataChanged(b *testing.B) {
	fileContent := make([]byte, 1024)
	f, err := os.CreateTemp(b.TempDir(), "*")
	require.NoError(b, err)
	defer f.Close()
	_, err = f.Write(fileContent)
	require.NoError(b, err)

	watcher := newPollingFileWatcher(f.Name(), time.Hour)
	require.NoError(b, watcher.Start())
	defer watcher.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		require.NoError(b, os.Chtimes(f.Name(), time.Now(), time.Now()))
		b.StartTimer()
		watcher.checkFileForChanges()
	}
}

func getEventWithTimeout(events chan *confmap.ChangeEvent, waitTime time.Duration) *confmap.ChangeEvent {
	select {
	case event, ok := <-events:
		if !ok {
			return nil
		}
		return event
	case <-time.After(waitTime):
		return nil
	}
}
