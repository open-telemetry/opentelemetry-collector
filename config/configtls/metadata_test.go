// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/client"
)

func TestMetadataFromPeerCert(t *testing.T) {
	uri1, err := url.Parse("spiffe://example/client")
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    x509.Certificate
		expected map[string][]string
	}{
		{
			name: "subject only",
			input: x509.Certificate{
				Subject: pkix.Name{CommonName: "test", Organization: []string{"example"}},
			},
			expected: map[string][]string{
				client.MetadataTLSPeerSubject: {"CN=test,O=example"},
			},
		},
		{
			name: "no subject, one URI",
			input: x509.Certificate{
				URIs: []*url.URL{uri1},
			},
			expected: map[string][]string{
				client.MetadataTLSPeerSubject: {""},
				client.MetadataTLSPeerURI:     {uri1.String()},
			},
		},
		{
			name: "subject and URIs",
			input: x509.Certificate{
				Subject: pkix.Name{CommonName: "test", Organization: []string{"example"}},
				URIs:    []*url.URL{uri1},
			},
			expected: map[string][]string{
				client.MetadataTLSPeerSubject: {"CN=test,O=example"},
				client.MetadataTLSPeerURI:     {uri1.String()},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := MetadataFromPeerCert(&tt.input)
			assert.Equal(t, tt.expected, md)
		})
	}
}
