// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtls"
)

// newTestCA returns a CA certificate (DER) and a server certificate issued by
// it for the given IP address.
func newTestCA(t *testing.T, ip string) ([]byte, tls.Certificate) {
	t.Helper()
	newKey := func() *ecdsa.PrivateKey {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		return key
	}
	now := time.Now()
	caKey := newKey()
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	ca, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	leafKey := newKey()
	leafDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test server"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP(ip)},
	}, ca, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	return caDER, tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: leafKey}
}

func poolOf(t *testing.T, caDER []byte) *x509.CertPool {
	t.Helper()
	ca, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	pool.AddCert(ca)
	return pool
}

// handshake performs a client handshake with creds against a TLS server on
// 127.0.0.1 presenting cert.
func handshake(t *testing.T, creds credentials.TransportCredentials, cert tls.Certificate) error {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	go func() {
		conn, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		defer conn.Close()
		_ = tls.Server(conn, &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
			// P-256 keeps the handshake valid under GODEBUG=fips140=only.
			CurvePreferences: []tls.CurveID{tls.CurveP256},
			NextProtos:       []string{"h2"},
		}).Handshake()
	}()

	rawConn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer rawConn.Close()
	conn, _, err := creds.ClientHandshake(t.Context(), ln.Addr().String(), rawConn)
	if err == nil {
		_ = conn.Close()
	}
	return err
}

func TestReloadingRootCAsCredentials(t *testing.T) {
	oldCA, oldCert := newTestCA(t, "127.0.0.1")
	newCA, newCert := newTestCA(t, "127.0.0.1")

	var pool atomic.Pointer[x509.CertPool]
	pool.Store(poolOf(t, oldCA))
	base := &tls.Config{MinVersion: tls.VersionTLS12, CurvePreferences: []tls.CurveID{tls.CurveP256}}
	creds := newReloadingRootCAsCredentials(base, pool.Load)

	require.NoError(t, handshake(t, creds, oldCert))
	require.ErrorContains(t, handshake(t, creds, newCert), "certificate signed by unknown authority")

	pool.Store(poolOf(t, newCA))
	require.NoError(t, handshake(t, creds, newCert))
	require.NoError(t, handshake(t, creds.Clone(), newCert), "a clone must keep reloading")
	assert.Equal(t, "tls", creds.Info().SecurityProtocol)
}

func TestReloadingRootCAsCredentialsVerifiesIPAddress(t *testing.T) {
	// The certificate is issued by the trusted CA, but for a different IP
	// address than the one being dialed.
	ca, cert := newTestCA(t, "127.0.0.2")
	pool := poolOf(t, ca)
	base := &tls.Config{MinVersion: tls.VersionTLS12, CurvePreferences: []tls.CurveID{tls.CurveP256}}
	creds := newReloadingRootCAsCredentials(base, func() *x509.CertPool { return pool })

	require.ErrorContains(t, handshake(t, creds, cert), "certificate is valid for 127.0.0.2, not 127.0.0.1")
}

func TestToClientConnReloadCAFile(t *testing.T) {
	caDER, _ := newTestCA(t, "127.0.0.1")
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}), 0o600))

	cfg := NewDefaultClientConfig()
	cfg.Endpoint = "localhost:1234"
	cfg.TLS = configtls.ClientConfig{ReloadCAFile: true, Config: configtls.Config{CAFile: caFile}}
	conn, err := cfg.ToClientConn(t.Context(), nil, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}
