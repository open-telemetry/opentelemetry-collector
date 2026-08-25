// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Don't run this test on Windows, as it requires a TPM simulator which depends on openssl headers.
//go:build !windows && !darwin

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	keyfile "github.com/foxboron/go-tpm-keyfiles"
	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
	"github.com/google/go-tpm/tpm2/transport/simulator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
)

func TestTPM_loadCertificate(t *testing.T) {
	testutil.SkipIfFIPSOnly(t, "use of CFB is not allowed in FIPS 140-only mode")
	tpm, err := simulator.OpenSimulator()
	require.NoError(t, err)
	defer tpm.Close()
	tpmSimulator = tpm

	// create a TPM key and certificate
	// the TPM key will be stored in the simulator
	tpmKey, cert := createTPMKeyCert(t, tpm)

	tempFileKey, err := os.CreateTemp(t.TempDir(), "tpmkey.key")
	require.NoError(t, err)
	_, err = tempFileKey.Write(tpmKey)
	require.NoError(t, err)
	tempFileCrt, err := os.CreateTemp(t.TempDir(), "tpmcert.crt")
	require.NoError(t, err)
	_, err = tempFileCrt.Write(cert)
	require.NoError(t, err)
	defer func() {
		tempFileKey.Close()
		os.Remove(tempFileKey.Name())
		tempFileCrt.Close()
		os.Remove(tempFileCrt.Name())
	}()

	tlsCfg := Config{
		CertFile: tempFileCrt.Name(),
		KeyFile:  tempFileKey.Name(),
		TPMConfig: TPMConfig{
			Enabled: true,
			Path:    "simulator",
		},
	}
	tlsCertificate, err := tlsCfg.loadCertificate()
	require.NoError(t, err)
	require.NotNil(t, tlsCertificate)

	h := crypto.SHA256.New()
	h.Write([]byte("message"))
	b := h.Sum(nil)

	// this is delegated to the TPM
	signer := tlsCertificate.PrivateKey.(crypto.Signer)
	signature, _ := signer.Sign(io.Reader(nil), b, crypto.SHA256)

	// load the key again to get access to verify function
	loadedTPMKey, err := keyfile.Decode(tpmKey)
	require.NoError(t, err)
	ok, err := loadedTPMKey.Verify(crypto.SHA256, b, signature)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestTPM_loadCertificate_error(t *testing.T) {
	tlsCfg := Config{
		CertPem: "invalid",
		KeyPem:  "invalid",
		TPMConfig: TPMConfig{
			Enabled: true,
			Path:    "simulator",
		},
	}
	tlsCertificate, err := tlsCfg.loadCertificate()
	assert.Equal(t, "failed to load private key from TPM: failed to load TPM key: not an armored key", err.Error())
	require.NotNil(t, tlsCertificate)
}

func TestTPM_tpmCertificate_errors(t *testing.T) {
	testutil.SkipIfFIPSOnly(t, "use of CFB is not allowed in FIPS 140-only mode")
	tpm, err := simulator.OpenSimulator()
	require.NoError(t, err)
	defer tpm.Close()

	openTPMFunc := func() (transport.TPMCloser, error) {
		return tpm, nil
	}

	key, _ := createTPMKeyCert(t, tpm)

	invalidCert := []byte(`-----BEGIN CERTIFICATE-----
VGhpcyBpcyBub3QgYSBjZXJ0aWZpY2F0ZS4=
-----END CERTIFICATE-----`)

	tests := []struct {
		name    string
		key     string
		openTPM func() (transport.TPMCloser, error)
		cert    string
		err     string
	}{
		{
			name:    "invalid key",
			key:     "invalid",
			openTPM: openTPMFunc,
			err:     "failed to load TPM key: not an armored key",
		},
		{
			name:    "invalid cert",
			key:     string(key),
			cert:    string(invalidCert),
			openTPM: openTPMFunc,
			err:     "failed to parse certificate: x509: malformed certificate",
		},
		{
			name: "failed to open TPM",
			openTPM: func() (transport.TPMCloser, error) {
				return nil, errors.New("failed to open TPM")
			},
			err: "failed to open TPM",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tpmCfg := &TPMConfig{}
			_, err = tpmCfg.tpmCertificate([]byte(test.key), []byte(test.cert), test.openTPM)
			assert.EqualError(t, err, test.err)
		})
	}
}

func TestTPM_open(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "app.sock")
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	tests := []struct {
		path string
		err  string
	}{
		{
			path: "",
			err:  "TPM path is not set",
		},
		{
			path: "/foo",
			err:  "failed to open TPM (/foo): stat /foo: no such file or directory",
		},
		{
			path: socketPath,
		},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			tpm, openErr := openTPM(test.path)()
			if test.err != "" {
				require.Nil(t, tpm)
				assert.Equal(t, test.err, openErr.Error())
			} else {
				require.NoError(t, openErr)
				require.NotNil(t, tpm)
				tpm.Close()
			}
		})
	}
}

func createTPMKeyCert(t *testing.T, tpm transport.TPMCloser) ([]byte, []byte) {
	tpmKey, err := keyfile.NewLoadableKey(tpm, tpm2.TPMAlgECC, 256, []byte(""))
	require.NoError(t, err)
	tpmKeySigner, err := tpmKey.Signer(tpm, []byte(""), []byte(""))
	require.NoError(t, err)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128) // 128-bit random number
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"OpenTelemetry"},
			CommonName:   "localhost",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(366 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
	}

	tpmPublicKey, err := tpmKey.PublicKey()
	require.NoError(t, err)
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, tpmPublicKey, tpmKeySigner)
	require.NoError(t, err)
	certBuffer := &bytes.Buffer{}
	err = pem.Encode(certBuffer, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	require.NoError(t, err)

	return tpmKey.Bytes(), certBuffer.Bytes()
}
