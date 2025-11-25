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

func TestGetIP(t *testing.T) {
	testCases := []struct {
		name     string
		header   http.Header
		keys     []string
		expected *net.IPAddr
	}{
		{
			name: "valid ip in first key",
			header: http.Header{
				"X-Forwarded-For": []string{"1.2.3.4"},
			},
			keys: []string{"X-Forwarded-For", "X-Real-IP"},
			expected: &net.IPAddr{
				IP: net.IPv4(1, 2, 3, 4),
			},
		},
		{
			name: "valid ip in second key",
			header: func() http.Header {
				h := make(http.Header)
				h.Set("X-Real-IP", "5.6.7.8")
				return h
			}(),
			keys: []string{"X-Forwarded-For", "X-Real-IP"},
			expected: &net.IPAddr{
				IP: net.IPv4(5, 6, 7, 8),
			},
		},
		{
			name: "valid ip with port",
			header: http.Header{
				"X-Forwarded-For": []string{"1.2.3.4:8080"},
			},
			keys: []string{"X-Forwarded-For"},
			expected: &net.IPAddr{
				IP: net.IPv4(1, 2, 3, 4),
			},
		},
		{
			name: "invalid ip falls to next key",
			header: func() http.Header {
				h := make(http.Header)
				h.Set("X-Forwarded-For", "invalid")
				h.Set("X-Real-IP", "9.10.11.12")
				return h
			}(),
			keys: []string{"X-Forwarded-For", "X-Real-IP"},
			expected: &net.IPAddr{
				IP: net.IPv4(9, 10, 11, 12),
			},
		},
		{
			name: "no valid ip",
			header: http.Header{
				"X-Forwarded-For": []string{"invalid"},
				"X-Real-IP":       []string{"also-invalid"},
			},
			keys:     []string{"X-Forwarded-For", "X-Real-IP"},
			expected: nil,
		},
		{
			name:     "empty keys",
			header:   http.Header{"X-Forwarded-For": []string{"1.2.3.4"}},
			keys:     []string{},
			expected: nil,
		},
		{
			name:     "empty header",
			header:   http.Header{},
			keys:     []string{"X-Forwarded-For", "X-Real-IP"},
			expected: nil,
		},
		{
			name: "missing key",
			header: http.Header{
				"Other-Header": []string{"1.2.3.4"},
			},
			keys:     []string{"X-Forwarded-For", "X-Real-IP"},
			expected: nil,
		},
		{
			name: "empty string value skips to next key",
			header: func() http.Header {
				h := make(http.Header)
				h.Set("X-Forwarded-For", "")
				h.Set("X-Real-IP", "10.11.12.13")
				return h
			}(),
			keys: []string{"X-Forwarded-For", "X-Real-IP"},
			expected: &net.IPAddr{
				IP: net.IPv4(10, 11, 12, 13),
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getIP(tt.header, tt.keys))
		})
	}
}
