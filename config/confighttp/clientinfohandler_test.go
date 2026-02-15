// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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
				IP: net.IPv4(1, 2, 3, 4).To4(),
			},
		},
		{
			name:  "addr:port",
			input: "1.2.3.4:33455",
			expected: &net.IPAddr{
				IP: net.IPv4(1, 2, 3, 4).To4(),
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
		{
			name:  "IPv6 addr",
			input: "::1",
			expected: &net.IPAddr{
				IP: net.ParseIP("::1"),
			},
		},
		{
			name:  "IPv6 [addr]:port",
			input: "[::1]:12345",
			expected: &net.IPAddr{
				IP: net.ParseIP("::1"),
			},
		},
		{
			name:  "IPv6 addr with zone",
			input: "fe80::1%eth0",
			expected: &net.IPAddr{
				IP:   net.ParseIP("fe80::1"),
				Zone: "eth0",
			},
		},
		{
			name:  "IPv6 [addr%zone]:port",
			input: "[fe80::1%eth0]:12345",
			expected: &net.IPAddr{
				IP:   net.ParseIP("fe80::1"),
				Zone: "eth0",
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseIP(tt.input))
		})
	}
}
