// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIP(t *testing.T) {
	testCases := []struct {
		desc     string
		input    string
		expected *net.IPAddr
	}{
		{
			desc:  "addr",
			input: "1.2.3.4",
			expected: &net.IPAddr{
				IP: net.IPv4(1, 2, 3, 4),
			},
		},
		{
			desc:  "addr:port",
			input: "1.2.3.4:33455",
			expected: &net.IPAddr{
				IP: net.IPv4(1, 2, 3, 4),
			},
		},
		{
			desc:     "protocol://addr:port",
			input:    "http://1.2.3.4:33455",
			expected: nil,
		},
		{
			desc:     "addr/path",
			input:    "1.2.3.4/orders",
			expected: nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			assert.Equal(t, tC.expected, parseIP(tC.input))
		})
	}
}
