// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// defaultClientCAsReloadInterval bounds how often the client CA file is checked
// for modifications. The check only happens while TLS handshakes are being
// served, so an idle server does no work at all.
const defaultClientCAsReloadInterval = time.Second

type clientCAsFileReloader struct {
	clientCAsFile string
	loader        clientCAsFileLoader
	// reloadInterval is the minimum duration between two checks of
	// clientCAsFile. Overridden by tests.
	reloadInterval time.Duration

	lock            sync.Mutex
	certPool        *x509.CertPool
	lastReloadError error
	// lastCheck is when clientCAsFile was most recently checked for changes.
	lastCheck time.Time
	// fileID identifies the contents of clientCAsFile as of the last check.
	fileID fileIdentity
}

// fileIdentity identifies the contents of a file by hash.
//
// Hashing the contents rather than comparing os.Stat metadata keeps detection
// independent of how the file came to be updated. In particular, Kubernetes
// projects ConfigMap and Secret volumes as a symlink to a timestamped directory
// and swaps that symlink on update, so the path being read resolves to a
// different underlying inode each time; and file modification timestamps carry
// filesystem-dependent granularity. Reading the file each check sidesteps both.
// The read is bounded by reloadInterval and CA bundles are small.
type fileIdentity struct {
	exists bool
	sum    [sha256.Size]byte
}

// identifyFile returns the identity of the file at path, following symlinks. A
// path that cannot be read yields the zero identity, which never compares equal
// to that of a readable file, so a file that disappears or reappears is always
// treated as a change.
func identifyFile(path string) fileIdentity {
	contents, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fileIdentity{}
	}
	return fileIdentity{exists: true, sum: sha256.Sum256(contents)}
}

type clientCAsFileLoader interface {
	loadClientCAFile() (*x509.CertPool, error)
}

func newClientCAsReloader(clientCAsFile string, loader clientCAsFileLoader) (*clientCAsFileReloader, error) {
	// Identify the file before loading it, so that a change racing with the
	// initial load is detected by the next check rather than being missed.
	fileID := identifyFile(clientCAsFile)

	certPool, err := loader.loadClientCAFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load client CA CertPool: %w", err)
	}

	return &clientCAsFileReloader{
		clientCAsFile:  clientCAsFile,
		loader:         loader,
		reloadInterval: defaultClientCAsReloadInterval,
		certPool:       certPool,
		lastCheck:      time.Now(),
		fileID:         fileID,
	}, nil
}

func (r *clientCAsFileReloader) getClientConfig(original *tls.Config) (*tls.Config, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.reloadIfModified(time.Now())

	return &tls.Config{
		RootCAs:              original.RootCAs,
		GetCertificate:       original.GetCertificate,
		GetClientCertificate: original.GetClientCertificate,
		MinVersion:           original.MinVersion,
		MaxVersion:           original.MaxVersion,
		NextProtos:           original.NextProtos,
		ClientCAs:            r.certPool,
		ClientAuth:           tls.RequireAndVerifyClientCert,
	}, nil
}

// reloadIfModified reloads the client CA CertPool if clientCAsFile has changed
// since the last check. Checks are rate limited to one per reloadInterval. A
// failed reload is recorded and leaves the previously loaded CertPool in place,
// so that a truncated or malformed file cannot break client authentication.
//
// The caller must hold r.lock.
func (r *clientCAsFileReloader) reloadIfModified(now time.Time) {
	if now.Sub(r.lastCheck) < r.reloadInterval {
		return
	}
	r.lastCheck = now

	fileID := identifyFile(r.clientCAsFile)
	if fileID == r.fileID {
		return
	}
	r.fileID = fileID

	certPool, err := r.loader.loadClientCAFile()
	if err != nil {
		r.lastReloadError = err
		return
	}
	r.certPool = certPool
	r.lastReloadError = nil
}

func (r *clientCAsFileReloader) getLastError() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.lastReloadError
}
