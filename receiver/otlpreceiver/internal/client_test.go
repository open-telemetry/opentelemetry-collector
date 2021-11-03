// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/client"
	"google.golang.org/grpc/peer"
)

func TestContextWithClient(t *testing.T) {
	testCases := []struct {
		desc     string
		input    context.Context
		expected *client.Client
	}{
		{
			desc:     "no peer information, empty client",
			input:    context.Background(),
			expected: &client.Client{},
		},
		{
			desc: "existing client with IP, no peer information",
			input: client.NewContext(context.Background(), &client.Client{
				IP: "1.2.3.4",
			}),
			expected: &client.Client{
				IP: "1.2.3.4",
			},
		},
		{
			desc: "empty client, with peer information",
			input: peer.NewContext(context.Background(), &peer.Peer{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 4),
				},
			}),
			expected: &client.Client{
				IP: "1.2.3.4",
			},
		},
		{
			desc: "existing client, existing IP gets overridden with peer information",
			input: peer.NewContext(client.NewContext(context.Background(), &client.Client{
				IP: "1.2.3.4",
			}), &peer.Peer{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 5),
				},
			}),
			expected: &client.Client{
				IP: "1.2.3.5",
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cl := client.FromContext(ContextWithClient(tC.input))
			assert.Equal(t, tC.expected, cl)
		})
	}
}
