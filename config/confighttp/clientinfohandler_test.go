// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/client"
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

func TestContextWithClientIncludesTLSInfo(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://localhost/v1/traces", http.NoBody)
	assert.NoError(t, err)
	req.RemoteAddr = "1.2.3.4:33455"
	peerCert := &x509.Certificate{DNSNames: []string{"client.example.com"}}
	req.TLS = &tls.ConnectionState{
		ServerName:       "service.example.com",
		PeerCertificates: []*x509.Certificate{peerCert},
	}

	got := client.FromContext(contextWithClient(req, false))

	assert.Equal(t, &client.TLSInfo{
		ServerName:       "service.example.com",
		PeerCertificates: []*x509.Certificate{peerCert},
	}, got.TLS)
}
