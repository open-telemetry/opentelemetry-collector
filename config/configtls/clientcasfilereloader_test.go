// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configtls

import (
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCannotShutdownIfNotWatching(t *testing.T) {
	reloader, _, _ := createReloader(t)
	err := reloader.shutdown()
	assert.Error(t, err)
}

func TestCannotStartIfAlreadyWatching(t *testing.T) {
	reloader, _, _ := createReloader(t)

	err := reloader.startWatching()
	assert.NoError(t, err)

	err = reloader.startWatching()
	assert.Error(t, err)

	err = reloader.shutdown()
	assert.NoError(t, err)
}

func TestClosingWatcherDoesntBreakReloader(t *testing.T) {
	reloader, loader, _ := createReloader(t)

	err := reloader.startWatching()
	assert.NoError(t, err)

	assert.Equal(t, 1, loader.reloadNumber())

	err = reloader.watcher.Close()
	assert.NoError(t, err)

	err = reloader.shutdown()
	assert.NoError(t, err)
}

func TestErrorRecordedIfFileDeleted(t *testing.T) {
	reloader, loader, filePath := createReloader(t)

	err := reloader.startWatching()
	assert.NoError(t, err)

	assert.Equal(t, 1, loader.reloadNumber())

	loader.returnErrorOnSubsequentCalls("test error on reload")

	err = os.WriteFile(filePath, []byte("some_data"), 0600)
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		return loader.reloadNumber() > 1 && reloader.getLastError() != nil
	}, 5*time.Second, 10*time.Millisecond)

	lastErr := reloader.getLastError()
	assert.Equal(t, "test error on reload", fmt.Sprint(lastErr))

	err = reloader.shutdown()
	assert.NoError(t, err)
}

func createReloader(t *testing.T) (*clientCAsFileReloader, *testLoader, string) {
	tmpClientCAsFilePath := createTempFile(t)
	loader := &testLoader{}
	reloader, _ := newClientCAsReloader(tmpClientCAsFilePath, loader)
	return reloader, loader, tmpClientCAsFilePath
}

func createTempFile(t *testing.T) string {
	tmpCa, err := os.CreateTemp("", "clientCAs.crt")
	assert.NoError(t, err)
	tmpCaPath, err := filepath.Abs(tmpCa.Name())
	assert.NoError(t, err)
	assert.NoError(t, tmpCa.Close())
	return tmpCaPath
}

type testLoader struct {
	err     error
	counter atomic.Uint32
}

func (r *testLoader) loadClientCAFile() (*x509.CertPool, error) {
	r.counter.Add(1)
	return nil, r.err
}

func (r *testLoader) returnErrorOnSubsequentCalls(msg string) {
	r.err = fmt.Errorf(msg)
}

func (r *testLoader) reloadNumber() int {
	return int(r.counter.Load())
}
