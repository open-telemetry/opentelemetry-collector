// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component/componenttest"
)

var _ http.Handler = (*clientInfoHandler)(nil)

func TestParseIP(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected *net.IPAddr
	}{
		{
			name:  "addr",
			input: "1.2.3.4",
			expected: &net.IPAddr{
				IP: net.IPv4(1, 2, 3, 4),
			},
		},
		{
			name:  "addr:port",
			input: "1.2.3.4:33455",
			expected: &net.IPAddr{
				IP: net.IPv4(1, 2, 3, 4),
			},
		},
		{
			name:     "protocol://addr:port",
			input:    "http://1.2.3.4:33455",
			expected: nil,
		},
		{
			name:     "addr/path",
			input:    "1.2.3.4/orders",
			expected: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseIP(tt.input))
		})
	}
}

func TestServerClientAddress(t *testing.T) {
	for _, tt := range []struct {
		name     string
		addr     net.Addr
		expected net.Addr
	}{
		{
			name:     "IPv4",
			addr:     &net.TCPAddr{IP: net.IPv4(192, 0, 2, 1), Port: 4318},
			expected: &net.IPAddr{IP: net.IPv4(192, 0, 2, 1)},
		},
		{
			name:     "IPv6",
			addr:     &net.TCPAddr{IP: net.ParseIP("2001:db8::1"), Port: 4318},
			expected: &net.IPAddr{IP: net.ParseIP("2001:db8::1")},
		},
		{
			name:     "IPv6 with zone",
			addr:     &net.TCPAddr{IP: net.ParseIP("fe80::1"), Port: 4318, Zone: "eth0"},
			expected: &net.IPAddr{IP: net.ParseIP("fe80::1"), Zone: "eth0"},
		},
		{
			name:     "Unix socket",
			addr:     &net.UnixAddr{Name: "client.sock", Net: "unix"},
			expected: &net.UnixAddr{Name: "client.sock", Net: "unix"},
		},
		{name: "nil address"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			received := make(chan client.Info, 1)
			cfg := NewDefaultServerConfig()
			cfg.IncludeMetadata = true
			srv, err := cfg.ToServer(t.Context(), nil, componenttest.NewNopTelemetrySettings(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received <- client.FromContext(r.Context())
				w.WriteHeader(http.StatusNoContent)
			}))
			require.NoError(t, err)
			ts := httptest.NewUnstartedServer(nil)
			ts.Config = srv
			ts.Listener = &clientAddressListener{Listener: ts.Listener, addr: tt.addr}
			ts.Start()
			t.Cleanup(ts.Close)

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL, http.NoBody)
			require.NoError(t, err)
			req.Header.Set("X-Client-Test", "metadata")
			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			_, err = io.Copy(io.Discard, resp.Body)
			require.NoError(t, resp.Body.Close())
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode)
			select {
			case info := <-received:
				assert.Equal(t, tt.expected, info.Addr)
				assert.Equal(t, []string{"metadata"}, info.Metadata.Get("X-Client-Test"))
			case <-time.After(time.Second):
				t.Fatal("handler was not called")
			}
		})
	}
}

type clientAddressListener struct {
	net.Listener
	addr net.Addr
}

func (l *clientAddressListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &clientAddressConn{Conn: conn, addr: l.addr}, nil
}

type clientAddressConn struct {
	net.Conn
	addr net.Addr
}

func (c *clientAddressConn) RemoteAddr() net.Addr {
	return c.addr
}

func TestContextWithClientPreservesAddress(t *testing.T) {
	addr := &net.TCPAddr{IP: net.ParseIP("fe80::1"), Port: 4318, Zone: "eth0"}
	ctx := client.NewContext(t.Context(), client.Info{Addr: addr})
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody).WithContext(ctx)
	assert.Equal(t, addr, client.FromContext(contextWithClient(req, false)).Addr)
}

func TestContextWithClientFallsBackToRemoteAddr(t *testing.T) {
	for _, tt := range []struct {
		name       string
		remoteAddr string
		expected   net.Addr
	}{
		{
			name:       "IPv4",
			remoteAddr: "192.0.2.1:4318",
			expected:   &net.IPAddr{IP: net.IPv4(192, 0, 2, 1)},
		},
		{
			name:       "IPv6",
			remoteAddr: "[2001:db8::1]:4318",
			expected:   &net.IPAddr{IP: net.ParseIP("2001:db8::1")},
		},
		{name: "invalid address", remoteAddr: "client.sock"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := client.NewContext(t.Context(), client.Info{})
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody).WithContext(ctx)
			req.RemoteAddr = tt.remoteAddr
			assert.Equal(t, tt.expected, client.FromContext(contextWithClient(req, false)).Addr)
		})
	}
}
