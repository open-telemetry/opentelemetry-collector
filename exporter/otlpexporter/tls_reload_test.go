// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter

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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
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

type nopLogsServer struct {
	plogotlp.UnimplementedGRPCServer
}

func (*nopLogsServer) Export(context.Context, plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	return plogotlp.NewExportResponse(), nil
}

// startTLSLogsServer starts an OTLP logs gRPC server on addr presenting cert.
func startTLSLogsServer(t *testing.T, addr string, cert tls.Certificate) *grpc.Server {
	t.Helper()
	ln, err := net.Listen("tcp", addr)
	require.NoError(t, err)
	srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	})))
	plogotlp.RegisterGRPCServer(srv, &nopLogsServer{})
	go func() { _ = srv.Serve(ln) }()
	return srv
}

// TestExportAfterCARotation reproduces
// https://github.com/open-telemetry/opentelemetry-collector/issues/11265 for
// the gRPC exporter: the server rotates to a certificate issued by a new CA and
// the CA file referenced by tls::ca_file is updated on disk. With
// reload_interval set, the exporter must pick up the new CA without being
// restarted.
func TestExportAfterCARotation(t *testing.T) {
	oldPKI, newPKI := newTestPKI(t, "old"), newTestPKI(t, "new")

	// Reserve a port so the server can be restarted on the same address.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	srv := startTLSLogsServer(t, addr, oldPKI.serverCert)
	t.Cleanup(func() { srv.Stop() })

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caFile, oldPKI.rootPEM, 0o600))

	const reloadInterval = 10 * time.Millisecond
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	// Disable queuing and retries so that ConsumeLogs reports the export result.
	cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	cfg.RetryConfig.Enabled = false
	cfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: addr,
		TLS: configtls.ClientConfig{
			Config: configtls.Config{
				CAFile:         caFile,
				ReloadInterval: reloadInterval,
			},
		},
	}
	exp, err := factory.CreateLogs(context.Background(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(context.Background())) })

	export := func() error {
		return exp.ConsumeLogs(context.Background(), plog.NewLogs())
	}

	require.NoError(t, export(), "export must succeed against the initial CA")

	// Server rotates to the new CA. Restarting it drops the existing connection
	// so the exporter has to perform a fresh handshake.
	srv.Stop()
	srv = startTLSLogsServer(t, addr, newPKI.serverCert)
	require.Eventually(t, func() bool {
		err := export()
		return err != nil && strings.Contains(err.Error(), "certificate signed by unknown authority")
	}, 10*time.Second, 50*time.Millisecond,
		"export must fail while the client still trusts only the old CA")

	// The CA file is rotated on disk; after reload_interval the exporter must
	// trust the new CA.
	require.NoError(t, os.WriteFile(caFile, newPKI.rootPEM, 0o600))
	require.Eventually(t, func() bool {
		return export() == nil
	}, 10*time.Second, 50*time.Millisecond,
		"export must succeed after the CA file was reloaded")
}
