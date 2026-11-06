// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// handshake runs a TLS handshake between a client using clientCfg and a server
// presenting the given testdata certificate, returning the client-side error.
func handshake(t *testing.T, clientCfg *tls.Config, serverCert string) error {
	t.Helper()
	cert, err := tls.LoadX509KeyPair(
		filepath.Join("testdata", serverCert+".crt"),
		filepath.Join("testdata", serverCert+".key"),
	)
	require.NoError(t, err)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = tls.Server(serverConn, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}).Handshake()
		_ = serverConn.Close()
	}()
	err = tls.Client(clientConn, clientCfg).Handshake()
	_ = clientConn.Close()
	<-done
	return err
}

func writeCA(t *testing.T, dst, src string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", src)) //nolint:gosec // test fixture path
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dst, data, 0o600)) //nolint:gosec // temp dir path from t.TempDir
}

func TestClientCAFileReload(t *testing.T) {
	tests := []struct {
		name           string
		reloadInterval time.Duration
		expectReloaded bool
	}{
		{name: "reloads CA after reload_interval", reloadInterval: time.Millisecond, expectReloaded: true},
		{name: "does not reload CA without reload_interval", reloadInterval: 0, expectReloaded: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caFile := filepath.Join(t.TempDir(), "ca.crt")
			writeCA(t, caFile, "ca-1.crt")

			cfg := ClientConfig{Config: Config{CAFile: caFile, ReloadInterval: tt.reloadInterval}}
			tlsCfg, err := cfg.LoadTLSConfig(context.Background())
			require.NoError(t, err)

			// Before rotation: server-1 is trusted by ca-1, server-2 is not.
			tlsCfg.ServerName = "example1"
			require.NoError(t, handshake(t, tlsCfg.Clone(), "server-1"))
			tlsCfg.ServerName = "example2"
			require.Error(t, handshake(t, tlsCfg.Clone(), "server-2"))

			// Rotate CA and server certificate.
			writeCA(t, caFile, "ca-2.crt")
			time.Sleep(5 * time.Millisecond)

			err = handshake(t, tlsCfg.Clone(), "server-2")
			if tt.expectReloaded {
				require.NoError(t, err)
				tlsCfg.ServerName = "example1"
				assert.Error(t, handshake(t, tlsCfg.Clone(), "server-1"), "old CA must no longer be trusted")
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestClientCAFileReloadVerifiesHostname(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCA(t, caFile, "ca-1.crt")
	cfg := ClientConfig{Config: Config{CAFile: caFile, ReloadInterval: time.Millisecond}}
	tlsCfg, err := cfg.LoadTLSConfig(context.Background())
	require.NoError(t, err)

	tlsCfg.ServerName = "wrong-host"
	assert.ErrorContains(t, handshake(t, tlsCfg, "server-1"), "failed to verify certificate")
}

func TestClientCAFileReloadKeepsPoolOnBadFile(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCA(t, caFile, "ca-1.crt")
	cfg := ClientConfig{Config: Config{CAFile: caFile, ReloadInterval: time.Millisecond}}
	tlsCfg, err := cfg.LoadTLSConfig(context.Background())
	require.NoError(t, err)

	writeCA(t, caFile, "testCA-bad.txt")
	time.Sleep(5 * time.Millisecond)
	tlsCfg.ServerName = "example1"
	assert.NoError(t, handshake(t, tlsCfg, "server-1"))
}

func TestClientCAFileReloadRespectsInsecureSkipVerify(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCA(t, caFile, "ca-1.crt")
	cfg := ClientConfig{
		Config:             Config{CAFile: caFile, ReloadInterval: time.Millisecond},
		InsecureSkipVerify: true,
	}
	tlsCfg, err := cfg.LoadTLSConfig(context.Background())
	require.NoError(t, err)
	assert.Nil(t, tlsCfg.VerifyConnection)
	assert.True(t, tlsCfg.InsecureSkipVerify)
}

// TestClientCAFileReloadHTTPClient exercises the reloader the way exporters use
// it: through net/http, with ServerName left empty in the config so that the
// transport derives it from the request URL.
func TestClientCAFileReloadHTTPClient(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCA(t, caFile, "ca-1.crt")
	cfg := ClientConfig{Config: Config{CAFile: caFile, ReloadInterval: time.Millisecond}}
	tlsCfg, err := cfg.LoadTLSConfig(context.Background())
	require.NoError(t, err)

	newServer := func(name string) *httptest.Server {
		cert, err := tls.LoadX509KeyPair(filepath.Join("testdata", name+".crt"), filepath.Join("testdata", name+".key"))
		require.NoError(t, err)
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
		srv.StartTLS()
		t.Cleanup(srv.Close)
		return srv
	}
	srv1, srv2 := newServer("server-1"), newServer("server-2")

	// Resolve the certificate hostnames to the matching test servers.
	addrs := map[string]string{
		"example1:443": srv1.Listener.Addr().String(),
		"example2:443": srv2.Listener.Addr().String(),
	}
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: tlsCfg,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addrs[addr])
		},
		DisableKeepAlives: true,
	}}
	get := func(host string) error {
		resp, err := client.Get("https://" + host + "/")
		if err == nil {
			_ = resp.Body.Close()
		}
		return err
	}

	require.NoError(t, get("example1"))
	require.Error(t, get("example2"))

	writeCA(t, caFile, "ca-2.crt")
	time.Sleep(5 * time.Millisecond)

	require.NoError(t, get("example2"))
	require.Error(t, get("example1"))
}
