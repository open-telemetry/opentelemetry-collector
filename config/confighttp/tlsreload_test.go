// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtls"
)

// newTestCA returns a PEM encoded CA certificate and a server certificate
// issued by it for the given IP address.
func newTestCA(t *testing.T, ip string) ([]byte, *tls.Certificate) {
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

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}),
		&tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: leafKey}
}

// startTLSServer starts an HTTP/2 capable TLS server on 127.0.0.1 serving the
// certificate stored in cert. It records the protocol of the last request.
func startTLSServer(t *testing.T, cert *atomic.Pointer[tls.Certificate], proto *atomic.Value) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proto.Store(r.Proto)
		w.WriteHeader(http.StatusOK)
	}))
	srv.EnableHTTP2 = true
	// Wrap the listener instead of using StartTLS: StartTLS installs a default
	// certificate that takes precedence over GetCertificate when the client
	// sends no SNI, which is the case when dialing an IP address.
	srv.Listener = tls.NewListener(srv.Listener, &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP256},
		NextProtos:       []string{"h2", "http/1.1"},
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return cert.Load(), nil
		},
	})
	srv.Start()
	t.Cleanup(srv.Close)
	return srv, "https://" + srv.Listener.Addr().String()
}

// fipsCurves keeps the handshake valid under GODEBUG=fips140=only, as used by
// `make test`.
var fipsCurves = []string{"P256"}

func get(t *testing.T, client *http.Client, target string) error {
	t.Helper()
	resp, err := client.Get(target)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func TestClientReloadsCAFile(t *testing.T) {
	oldCA, oldCert := newTestCA(t, "127.0.0.1")
	newCA, newCert := newTestCA(t, "127.0.0.1")

	var cert atomic.Pointer[tls.Certificate]
	cert.Store(oldCert)
	var proto atomic.Value
	srv, url := startTLSServer(t, &cert, &proto)

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caFile, oldCA, 0o600))

	cfg := NewDefaultClientConfig()
	cfg.ForceAttemptHTTP2 = true
	cfg.TLS = configtls.ClientConfig{ReloadCAFile: true, Config: configtls.Config{CAFile: caFile, CurvePreferences: fipsCurves}}
	client, err := cfg.ToClient(t.Context(), nil, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	require.NoError(t, get(t, client, url))
	assert.Equal(t, "HTTP/2.0", proto.Load(), "HTTP/2 must still be negotiated")

	// The server rotates to the new CA before the client CA file is updated.
	cert.Store(newCert)
	srv.CloseClientConnections()
	// The first request may still hit the connection being closed.
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.ErrorContains(c, get(t, client, url), "certificate signed by unknown authority")
	}, 5*time.Second, 10*time.Millisecond)

	require.NoError(t, os.WriteFile(caFile, newCA, 0o600))
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		srv.CloseClientConnections()
		assert.NoError(c, get(t, client, url))
	}, 5*time.Second, 100*time.Millisecond)
}

func TestClientReloadCAFileVerifiesIPAddress(t *testing.T) {
	// The certificate is issued by the trusted CA, but for a different IP
	// address than the one being dialed.
	ca, cert := newTestCA(t, "127.0.0.2")
	var certPtr atomic.Pointer[tls.Certificate]
	certPtr.Store(cert)
	var proto atomic.Value
	_, url := startTLSServer(t, &certPtr, &proto)

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caFile, ca, 0o600))

	cfg := NewDefaultClientConfig()
	cfg.TLS = configtls.ClientConfig{ReloadCAFile: true, Config: configtls.Config{CAFile: caFile, CurvePreferences: fipsCurves}}
	client, err := cfg.ToClient(t.Context(), nil, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	require.ErrorContains(t, get(t, client, url), "certificate is valid for 127.0.0.2, not 127.0.0.1")
}

func TestClientReloadCAFileWarnsWithProxy(t *testing.T) {
	ca, _ := newTestCA(t, "127.0.0.1")
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caFile, ca, 0o600))

	core, logs := observer.New(zapcore.WarnLevel)
	set := componenttest.NewNopTelemetrySettings()
	set.Logger = zap.New(core)

	cfg := NewDefaultClientConfig()
	cfg.ProxyURL = "http://proxy.invalid:8080"
	cfg.TLS = configtls.ClientConfig{ReloadCAFile: true, Config: configtls.Config{CAFile: caFile}}
	_, err := cfg.ToClient(t.Context(), nil, set)
	require.NoError(t, err)
	assert.Equal(t, 1, logs.FilterMessageSnippet("ca_file_reload has no effect").Len())
}

func TestDialTLSWithReloadedRootCAsErrors(t *testing.T) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	dial := dialTLSWithReloadedRootCAs(transport, x509.NewCertPool)

	_, err := dial(t.Context(), "tcp", "missing-port")
	require.ErrorContains(t, err, "missing port in address")

	// Nothing listens on a port that was just released.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	_, err = dial(t.Context(), "tcp", addr)
	require.Error(t, err)
}

func TestClientReloadCAFileInvalidFile(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caFile, []byte("not a certificate"), 0o600))

	cfg := NewDefaultClientConfig()
	cfg.TLS = configtls.ClientConfig{ReloadCAFile: true, Config: configtls.Config{CAFile: caFile}}
	_, err := cfg.ToClient(t.Context(), nil, componenttest.NewNopTelemetrySettings())
	require.ErrorContains(t, err, "failed to load CA file")
}
