// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
	"github.com/stretchr/testify/require"
)

func TestCannotShutdownIfNotWatching(t *testing.T) {
	reloader, _, _ := createReloader(t)
	err := reloader.shutdown()
	assert.Error(t, err)
}

func TestCannotStartIfAlreadyWatching(t *testing.T) {
	reloader, _, _ := createReloader(t)

	err := reloader.startWatching()
	require.NoError(t, err)

	err = reloader.startWatching()
	require.Error(t, err)

	err = reloader.shutdown()
	assert.NoError(t, err)
}

func TestClosingWatcherDoesntBreakReloader(t *testing.T) {
	reloader, loader, _ := createReloader(t)

	err := reloader.startWatching()
	require.NoError(t, err)

	assert.Equal(t, 1, loader.reloadNumber())

	err = reloader.watcher.Close()
	require.NoError(t, err)

	err = reloader.shutdown()
	assert.NoError(t, err)
}

func TestErrorRecordedIfFileDeleted(t *testing.T) {
	reloader, loader, filePath := createReloader(t)

	err := reloader.startWatching()
	require.NoError(t, err)

	assert.Equal(t, 1, loader.reloadNumber())

	loader.returnErrorOnSubsequentCalls("test error on reload")

	err = os.WriteFile(filePath, []byte("some_data"), 0o600)
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		return loader.reloadNumber() > 1 && reloader.getLastError() != nil
	}, 5*time.Second, 10*time.Millisecond)

	lastErr := reloader.getLastError()
	require.EqualError(t, lastErr, "test error on reload")

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
	tmpCa, err := os.CreateTemp(t.TempDir(), "clientCAs.crt")
	require.NoError(t, err)
	tmpCaPath, err := filepath.Abs(tmpCa.Name())
	assert.NoError(t, err)
	assert.NoError(t, tmpCa.Close())
	return tmpCaPath
}

type testLoader struct {
	err     atomic.Value
	counter atomic.Uint32
}

func (r *testLoader) loadClientCAFile() (*x509.CertPool, error) {
	r.counter.Add(1)

	v := r.err.Load()
	if v == nil {
		return nil, nil
	}

	return nil, v.(error)
}

func (r *testLoader) returnErrorOnSubsequentCalls(msg string) {
	r.err.Store(fmt.Errorf("%s", msg))
}

func (r *testLoader) reloadNumber() int {
	return int(r.counter.Load())
}
