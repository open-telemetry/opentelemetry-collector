// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/pdata/internal"
)

func TestEncodeDecodeContext(t *testing.T) {
	spanCtx := fakeSpanContext(t)
	clientMetadata := client.NewMetadata(map[string][]string{
		"key1": {"value1"},
		"key2": {"value2", "value3"},
	})
	tests := []struct {
		name       string
		clientInfo client.Info
	}{
		{
			name:       "without_client_address",
			clientInfo: client.Info{Metadata: clientMetadata},
		},
		{
			name: "with_client_IP_address",
			clientInfo: client.Info{
				Metadata: clientMetadata,
				Addr: &net.IPAddr{
					IP:   net.IPv6loopback,
					Zone: "eth0",
				},
			},
		},
		{
			name: "with_client_TCP_address",
			clientInfo: client.Info{
				Metadata: clientMetadata,
				Addr: &net.TCPAddr{
					IP:   net.IPv4(127, 0, 0, 1),
					Port: 8080,
				},
			},
		},
		{
			name: "with_client_UDP_address",
			clientInfo: client.Info{
				Metadata: clientMetadata,
				Addr: &net.UDPAddr{
					IP:   net.IPv4(127, 0, 0, 1),
					Port: 8080,
				},
			},
		},
		{
			name: "with_client_unix_address",
			clientInfo: client.Info{
				Metadata: clientMetadata,
				Addr: &net.UnixAddr{
					Name: "/var/run/test.sock",
					Net:  "unixpacket",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode a context with a span and client metadata
			ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)
			ctx = client.NewContext(ctx, tt.clientInfo)
			reqCtx := encodeContext(ctx)
			buf := make([]byte, reqCtx.SizeProto())
			reqCtx.MarshalProto(buf)

			// Decode the context
			gotReqCtx := internal.RequestContext{}
			require.NoError(t, gotReqCtx.UnmarshalProto(buf))
			gotCtx := decodeContext(context.Background(), &gotReqCtx)
			assert.Equal(t, spanCtx, trace.SpanContextFromContext(gotCtx))
			assert.Equal(t, tt.clientInfo, client.FromContext(gotCtx))
		})
	}

	// Decode a nil context
	assert.Equal(t, context.Background(), decodeContext(context.Background(), nil))
}
