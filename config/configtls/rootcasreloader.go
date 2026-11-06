// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"sync"
	"time"
)

// rootCAsReloader keeps the CA pool loaded from Config.CAFile and reloads it
// from disk once ReloadInterval has elapsed since the last load.
//
// crypto/tls offers no hook to swap tls.Config.RootCAs for a live client
// (https://github.com/golang/go/issues/35887), so the reloaded pool is applied
// by verifying the server certificate chain ourselves in verifyConnection.
type rootCAsReloader struct {
	cfg Config

	lock       sync.Mutex
	pool       *x509.CertPool
	nextReload time.Time
}

func newRootCAsReloader(cfg Config, pool *x509.CertPool) *rootCAsReloader {
	return &rootCAsReloader{
		cfg:        cfg,
		pool:       pool,
		nextReload: time.Now().Add(cfg.ReloadInterval),
	}
}

// certPool returns the current CA pool, reloading it if the reload interval
// has elapsed. If reloading fails, the previously loaded pool is kept so that
// a transiently unreadable or partially written file does not break exports.
func (r *rootCAsReloader) certPool() *x509.CertPool {
	r.lock.Lock()
	defer r.lock.Unlock()
	now := time.Now()
	if r.nextReload.Before(now) {
		r.nextReload = now.Add(r.cfg.ReloadInterval)
		if pool, err := r.cfg.loadCertFile(r.cfg.CAFile); err == nil {
			r.pool = pool
		}
	}
	return r.pool
}

// verifyConnection performs the same server certificate verification that
// crypto/tls does by default (chain and hostname), but against the current
// reloaded CA pool. It must only be used with InsecureSkipVerify set, which
// disables the built-in verification that this replaces.
func (r *rootCAsReloader) verifyConnection(cs tls.ConnectionState) error {
	if len(cs.PeerCertificates) == 0 {
		return errors.New("tls: server did not present a certificate")
	}
	intermediates := x509.NewCertPool()
	for _, cert := range cs.PeerCertificates[1:] {
		intermediates.AddCert(cert)
	}
	opts := x509.VerifyOptions{
		Roots:         r.certPool(),
		DNSName:       cs.ServerName,
		Intermediates: intermediates,
	}
	if _, err := cs.PeerCertificates[0].Verify(opts); err != nil {
		return fmt.Errorf("tls: failed to verify certificate: %w", err)
	}
	return nil
}
