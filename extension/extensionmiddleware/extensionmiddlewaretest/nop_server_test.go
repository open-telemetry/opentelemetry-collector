// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddlewaretest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

func TestNopServer(t *testing.T) {
	client := NewNopServer()

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
