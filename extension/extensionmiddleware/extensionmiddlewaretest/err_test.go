// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddlewaretest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

func TestErrClient(t *testing.T) {
	client := NewErr(errors.New("error"))

	httpClient, ok := client.(extensionmiddleware.HTTPClient)
	require.True(t, ok)
	_, err := httpClient.GetHTTPRoundTripper(nil)
	require.Error(t, err)

	grpcClient, ok := client.(extensionmiddleware.GRPCClient)
	require.True(t, ok)
	_, err = grpcClient.GetGRPCClientOptions()
	require.Error(t, err)
}

func TestErrServer(t *testing.T) {
	server := NewErr(errors.New("error"))

	httpServer, ok := server.(extensionmiddleware.HTTPServer)
	require.True(t, ok)
	_, err := httpServer.GetHTTPHandler(nil)
	require.Error(t, err)

	grpcServer, ok := server.(extensionmiddleware.GRPCServer)
	require.True(t, ok)
	_, err = grpcServer.GetGRPCServerOptions()
	require.Error(t, err)
}
