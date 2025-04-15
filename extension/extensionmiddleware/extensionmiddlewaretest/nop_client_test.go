// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddlewaretest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

func TestNopClient(t *testing.T) {
	client := NewNopClient()

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
