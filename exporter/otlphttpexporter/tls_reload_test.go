// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlphttpexporter

import (
	"context"
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

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter/internal/metadata"
	"go.opentelemetry.io/collector/pdata/plog"
)

// testPKI is a root CA, an intermediate CA and a server certificate for
// 127.0.0.1 signed by the intermediate, generated at test time.
type testPKI struct {
	rootPEM    []byte
	serverCert tls.Certificate
}

func newTestPKI(t *testing.T, name string) testPKI {
	t.Helper()
	now := time.Now()
	newKey := func() *ecdsa.PrivateKey {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		return key
	}
	sign := func(tmpl, parent *x509.Certificate, pub *ecdsa.PublicKey, signer *ecdsa.PrivateKey) *x509.Certificate {
		der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, pub, signer)
		require.NoError(t, err)
		cert, err := x509.ParseCertificate(der)
		require.NoError(t, err)
		return cert
	}
	caTemplate := func(serial int64, cn string) *x509.Certificate {
		return &x509.Certificate{
			SerialNumber:          big.NewInt(serial),
			Subject:               pkix.Name{CommonName: cn},
			NotBefore:             now.Add(-time.Hour),
			NotAfter:              now.Add(time.Hour),
			KeyUsage:              x509.KeyUsageCertSign,
			BasicConstraintsValid: true,
			IsCA:                  true,
		}
	}

	rootKey := newKey()
	rootTmpl := caTemplate(1, name+" root")
	root := sign(rootTmpl, rootTmpl, &rootKey.PublicKey, rootKey)

	intKey := newKey()
	intermediate := sign(caTemplate(2, name+" intermediate"), root, &intKey.PublicKey, rootKey)

	leafKey := newKey()
	leaf := sign(&x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: name + " server"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}, intermediate, &leafKey.PublicKey, intKey)

	return testPKI{
		rootPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: root.Raw}),
		serverCert: tls.Certificate{
			Certificate: [][]byte{leaf.Raw, intermediate.Raw},
			PrivateKey:  leafKey,
			Leaf:        leaf,
		},
	}
}

// TestExportAfterCARotation reproduces
// https://github.com/open-telemetry/opentelemetry-collector/issues/11265:
// the server rotates to a certificate issued by a new CA and the CA file
// referenced by tls::ca_file is updated on disk. With reload_interval set, the
// exporter must pick up the new CA without being restarted.
func TestExportAfterCARotation(t *testing.T) {
	oldPKI, newPKI := newTestPKI(t, "old"), newTestPKI(t, "new")

	var serverCert atomic.Pointer[tls.Certificate]
	serverCert.Store(&oldPKI.serverCert)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	// Wrap the listener directly instead of using StartTLS: StartTLS installs a
	// default certificate that takes precedence over GetCertificate when the
	// client sends no SNI, which is the case when dialing an IP address.
	srv.Listener = tls.NewListener(srv.Listener, &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return serverCert.Load(), nil
		},
	})
	srv.Start()
	defer srv.Close()
	endpoint := "https://" + srv.Listener.Addr().String() + "/v1/logs"

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caFile, oldPKI.rootPEM, 0o600))

	const reloadInterval = 10 * time.Millisecond
	cfg := &Config{
		Encoding:     EncodingProto,
		LogsEndpoint: endpoint,
		ClientConfig: confighttp.ClientConfig{
			TLS: configtls.ClientConfig{
				Config: configtls.Config{
					CAFile:         caFile,
					ReloadInterval: reloadInterval,
				},
			},
		},
	}
	exp, err := createLogs(context.Background(), exportertest.NewNopSettings(metadata.Type), cfg)
	require.NoError(t, err)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(context.Background())) })

	export := func() error {
		return exp.ConsumeLogs(context.Background(), plog.NewLogs())
	}

	require.NoError(t, export(), "export must succeed against the initial CA")

	// Server rotates to the new CA. Drop pooled connections so the next export
	// performs a fresh handshake.
	serverCert.Store(&newPKI.serverCert)
	srv.CloseClientConnections()
	require.ErrorContains(t, export(), "certificate signed by unknown authority",
		"export must fail while the client still trusts only the old CA")

	// The CA file is rotated on disk; after reload_interval the exporter must
	// trust the new CA.
	require.NoError(t, os.WriteFile(caFile, newPKI.rootPEM, 0o600))
	time.Sleep(2 * reloadInterval)
	srv.CloseClientConnections()
	require.NoError(t, export(), "export must succeed after the CA file was reloaded")
}
