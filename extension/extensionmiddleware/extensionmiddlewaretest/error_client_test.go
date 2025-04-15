// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddlewaretest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

func TestErrorClient(t *testing.T) {
	client := NewErrorClient(errors.New("error"))

	httpClient, ok := client.(extensionmiddleware.HTTPClient)
	require.True(t, ok)
	_, err := httpClient.GetHTTPRoundTripper(nil)
	require.Error(t, err)

	grpcClient, ok := client.(extensionmiddleware.GRPCClient)
	require.True(t, ok)
	_, err = grpcClient.GetGRPCClientOptions()
	require.Error(t, err)
}
