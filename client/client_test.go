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

// Package client contains generic representations of clients connecting to different receivers
package client

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/peer"
)

func TestClientContext(t *testing.T) {
	ips := []string{
		"1.1.1.1", "127.0.0.1", "1111", "ip",
	}
	for _, ip := range ips {
		ctx := NewContext(context.Background(), &Client{ip})
		c, ok := FromContext(ctx)
		assert.True(t, ok)
		assert.NotNil(t, c)
		assert.Equal(t, c.IP, ip)
	}
}

func TestParsingGRPC(t *testing.T) {
	grpcCtx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("192.168.1.1"),
			Port: 80,
		},
	})

	client, ok := FromGRPC(grpcCtx)
	assert.True(t, ok)
	assert.NotNil(t, client)
	assert.Equal(t, client.IP, "192.168.1.1")
}

func TestParsingHTTP(t *testing.T) {
	client, ok := FromHTTP(&http.Request{RemoteAddr: "192.168.1.2"})
	assert.True(t, ok)
	assert.NotNil(t, client)
	assert.Equal(t, client.IP, "192.168.1.2")
}
