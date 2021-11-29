// Copyright  The OpenTelemetry Authors
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
	"io/ioutil"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWatchFile(t *testing.T) {
	// Create a temporary file.
	file, err := ioutil.TempFile("", "file_watcher_test")
	require.NoError(t, err)
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	received := atomic.Value{}
	received.Store(false)

	// Write some initial content.
	_, err = file.WriteString("hello")
	require.NoError(t, err)
	require.NoError(t, file.Sync())

	// Setup the watch.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watchFile(ctx, file.Name(), testingOnChange(t, &received))
	time.Sleep(time.Second * 2)

	// Update the file and verify we see the updated content.
	_, err = file.WriteString(" world")
	require.NoError(t, err)
	require.NoError(t, file.Sync())

	require.Eventually(t, func() bool {
		return received.Load().(bool)
	}, time.Second*10, time.Second)

	// Cancel the context.
	cancel()
}

func TestWatchFile_ReloadError(t *testing.T) {
	// Create then delete a temporary file so we have a filename that we know can't be opened.
	file, err := ioutil.TempFile("", "file_watcher_test")
	require.NoError(t, err)
	_ = file.Close()
	require.NoError(t, os.Remove(file.Name()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watchFile(ctx, file.Name(), func(event *ChangeEvent) {})
}

func testingOnChange(t *testing.T, r *atomic.Value) func(c *ChangeEvent) {
	return func(c *ChangeEvent) {
		require.Nil(t, c.Error)
		r.Store(true)
	}
}
