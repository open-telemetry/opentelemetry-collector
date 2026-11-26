// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCAPEM returns a PEM encoded self-signed CA certificate.
func newCAPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func poolFromPEM(t *testing.T, caPEM string) *x509.CertPool {
	t.Helper()
	pool, err := Config{}.loadCertPem([]byte(caPEM))
	require.NoError(t, err)
	return pool
}

func TestLoadRootCAsFuncDisabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  ClientConfig
	}{
		{
			name: "ca_file_reload not set",
			cfg:  ClientConfig{Config: Config{CAFile: filepath.Join("testdata", "ca-1.crt")}},
		},
		{
			name: "no ca_file",
			cfg:  ClientConfig{ReloadCAFile: true, Config: Config{CAPem: "unused"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCAs, err := tt.cfg.LoadRootCAsFunc(t.Context())
			require.NoError(t, err)
			assert.Nil(t, rootCAs)
		})
	}
}

func TestLoadRootCAsFuncInvalidFile(t *testing.T) {
	cfg := ClientConfig{ReloadCAFile: true, Config: Config{CAFile: filepath.Join("testdata", "testCA-bad.txt")}}
	_, err := cfg.LoadRootCAsFunc(t.Context())
	assert.ErrorContains(t, err, "failed to load CA file")
}

func TestLoadRootCAsFuncReloadsModifiedFile(t *testing.T) {
	oldCA, newCA := newCAPEM(t), newCAPEM(t)
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeFile(t, caFile, oldCA)

	cfg := ClientConfig{ReloadCAFile: true, Config: Config{CAFile: caFile}}
	rootCAs, err := cfg.LoadRootCAsFunc(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rootCAs)
	assert.True(t, rootCAs().Equal(poolFromPEM(t, oldCA)))

	writeFile(t, caFile, newCA)
	assert.Eventually(t, func() bool {
		return rootCAs().Equal(poolFromPEM(t, newCA))
	}, 5*defaultClientCAsReloadInterval, 50*time.Millisecond)

	// A file that can no longer be loaded keeps the last good pool.
	writeFile(t, caFile, "not a certificate")
	time.Sleep(2 * defaultClientCAsReloadInterval)
	assert.True(t, rootCAs().Equal(poolFromPEM(t, newCA)))
}
