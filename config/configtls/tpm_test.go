// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls // import "go.opentelemetry.io/collector/config/configtls"
import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"testing"
	"time"

	keyfile "github.com/foxboron/go-tpm-keyfiles"
	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
	"github.com/google/go-tpm/tpm2/transport/simulator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTPMGeneratedKey(t *testing.T) {
	tpm, err := simulator.OpenSimulator()
	require.NoError(t, err)
	defer tpm.Close()

	// create a TPM key and certificate
	// the TPM key will be stored in the simulator
	tpmKey, cert := createTPMKeyCert(t, tpm)

	// lod the key and certificate from TPM
	tpmCfg := TPMConfig{}
	tlsCertificate, err := tpmCfg.tpmCertificate(tpmKey, cert, func() (transport.TPMCloser, error) {
		return tpm, nil
	})
	require.NoError(t, err)

	h := crypto.SHA256.New()
	h.Write([]byte("message"))
	b := h.Sum(nil)

	// this is delegated to the TPM
	signer := tlsCertificate.PrivateKey.(crypto.Signer)
	signature, _ := signer.Sign((io.Reader)(nil), b, crypto.SHA256)

	// load the key again to get access to verify function
	loadedTPMKey, err := keyfile.Decode(tpmKey)
	require.NoError(t, err)
	ok, err := loadedTPMKey.Verify(crypto.SHA256, b, signature)
	require.NoError(t, err)
	assert.True(t, ok)
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
