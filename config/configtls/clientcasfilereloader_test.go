// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReloaderLoadsOnceOnCreation(t *testing.T) {
	reloader, loader, _ := createReloader(t)

	assert.Equal(t, 1, loader.reloadNumber())
	assert.Same(t, loader.lastReturnedPool(), reloader.certPool)
	assert.NoError(t, reloader.getLastError())
}

func TestReloaderDoesNotReloadWhenFileUnchanged(t *testing.T) {
	reloader, loader, _ := createReloader(t)

	for range 3 {
		_, err := reloader.getClientConfig(&tls.Config{})
		require.NoError(t, err)
	}

	assert.Equal(t, 1, loader.reloadNumber())
}

func TestReloaderReloadsWhenFileChanged(t *testing.T) {
	reloader, loader, filePath := createReloader(t)

	writeFile(t, filePath, "updated contents")

	clientCfg, err := reloader.getClientConfig(&tls.Config{})
	require.NoError(t, err)

	assert.Equal(t, 2, loader.reloadNumber())
	assert.Same(t, loader.lastReturnedPool(), clientCfg.ClientCAs)
	assert.NoError(t, reloader.getLastError())
}

func TestReloaderRateLimitsChecks(t *testing.T) {
	reloader, loader, filePath := createReloader(t)
	// A long interval means the change must not be picked up yet.
	reloader.reloadInterval = time.Hour

	writeFile(t, filePath, "updated contents")

	_, err := reloader.getClientConfig(&tls.Config{})
	require.NoError(t, err)
	assert.Equal(t, 1, loader.reloadNumber())

	// Once the interval has elapsed, the change is picked up.
	reloader.reloadInterval = 0

	_, err = reloader.getClientConfig(&tls.Config{})
	require.NoError(t, err)
	assert.Equal(t, 2, loader.reloadNumber())
}

func TestReloaderKeepsPreviousPoolWhenReloadFails(t *testing.T) {
	reloader, loader, filePath := createReloader(t)
	originalPool := loader.lastReturnedPool()

	loader.returnErrorOnSubsequentCalls("test error on reload")
	writeFile(t, filePath, "updated contents")

	clientCfg, err := reloader.getClientConfig(&tls.Config{})
	require.NoError(t, err)

	assert.Equal(t, 2, loader.reloadNumber())
	assert.Same(t, originalPool, clientCfg.ClientCAs)
	require.EqualError(t, reloader.getLastError(), "test error on reload")
}

func TestReloaderRecordsErrorIfFileDeleted(t *testing.T) {
	reloader, loader, filePath := createReloader(t)

	loader.returnErrorOnSubsequentCalls("test error on reload")
	require.NoError(t, os.Remove(filePath))

	_, err := reloader.getClientConfig(&tls.Config{})
	require.NoError(t, err)

	assert.Equal(t, 2, loader.reloadNumber())
	require.EqualError(t, reloader.getLastError(), "test error on reload")

	// While the file stays missing, no further load attempts are made.
	_, err = reloader.getClientConfig(&tls.Config{})
	require.NoError(t, err)
	assert.Equal(t, 2, loader.reloadNumber())
}

func TestReloaderRecoversAfterFileReappears(t *testing.T) {
	reloader, loader, filePath := createReloader(t)

	loader.returnErrorOnSubsequentCalls("test error on reload")
	require.NoError(t, os.Remove(filePath))

	_, err := reloader.getClientConfig(&tls.Config{})
	require.NoError(t, err)
	require.Error(t, reloader.getLastError())

	loader.stopReturningError()
	writeFile(t, filePath, "restored contents")

	clientCfg, err := reloader.getClientConfig(&tls.Config{})
	require.NoError(t, err)

	assert.Same(t, loader.lastReturnedPool(), clientCfg.ClientCAs)
	assert.NoError(t, reloader.getLastError())
}

func TestReloaderPreservesOriginalConfigFields(t *testing.T) {
	reloader, _, _ := createReloader(t)

	original := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		NextProtos: []string{"h2"},
	}

	clientCfg, err := reloader.getClientConfig(original)
	require.NoError(t, err)

	assert.Equal(t, original.MinVersion, clientCfg.MinVersion)
	assert.Equal(t, original.MaxVersion, clientCfg.MaxVersion)
	assert.Equal(t, original.NextProtos, clientCfg.NextProtos)
	assert.Equal(t, tls.RequireAndVerifyClientCert, clientCfg.ClientAuth)
}

// createReloader returns a reloader with change detection unthrottled, so that
// tests observe a reload as soon as the file changes.
func createReloader(t *testing.T) (*clientCAsFileReloader, *testLoader, string) {
	tmpClientCAsFilePath := createTempFile(t)
	loader := &testLoader{}
	reloader, err := newClientCAsReloader(tmpClientCAsFilePath, loader)
	require.NoError(t, err)
	reloader.reloadInterval = 0
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

func writeFile(t *testing.T, path, contents string) {
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
}

type testLoader struct {
	lock    sync.Mutex
	err     error
	pool    *x509.CertPool
	counter int
}

func (r *testLoader) loadClientCAFile() (*x509.CertPool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.counter++
	if r.err != nil {
		return nil, r.err
	}

	// Return a distinct pool per call so tests can assert, by identity, which
	// load the active pool came from.
	r.pool = x509.NewCertPool()
	return r.pool, nil
}

func (r *testLoader) returnErrorOnSubsequentCalls(msg string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.err = errors.New(msg)
}

func (r *testLoader) stopReturningError() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.err = nil
}

func (r *testLoader) reloadNumber() int {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.counter
}

func (r *testLoader) lastReturnedPool() *x509.CertPool {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.pool
}
