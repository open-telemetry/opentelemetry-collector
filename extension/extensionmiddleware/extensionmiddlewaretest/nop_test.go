// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddlewaretest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

func TestNopClient(t *testing.T) {
	client := NewNop()

	httpClient, ok := client.(extensionmiddleware.HTTPClient)
	require.True(t, ok)
	rt, err := httpClient.GetHTTPRoundTripper(nil)
	require.NoError(t, err)
	require.Nil(t, rt)

	grpcClient, ok := client.(extensionmiddleware.GRPCClient)
	require.True(t, ok)
	grpcOpts, err := grpcClient.GetGRPCClientOptions()
	require.NoError(t, err)
	require.Nil(t, grpcOpts)
}

func TestNopServer(t *testing.T) {
	client := NewNop()

	httpServer, ok := client.(extensionmiddleware.HTTPServer)
	require.True(t, ok)
	rt, err := httpServer.GetHTTPHandler(nil)
	require.NoError(t, err)
	require.Nil(t, rt)

	grpcServer, ok := client.(extensionmiddleware.GRPCServer)
	require.True(t, ok)
	grpcOpts, err := grpcServer.GetGRPCServerOptions()
	require.NoError(t, err)
	require.Nil(t, grpcOpts)
}

func TestRoundTripperFunc(t *testing.T) {
	called := false
	req := &http.Request{}
	resp := &http.Response{}

	f := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, r, req)
		called = true
		return resp, nil
	})

	result, _ := f.RoundTrip(req)
	require.True(t, called)
	require.Equal(t, resp, result)
}
