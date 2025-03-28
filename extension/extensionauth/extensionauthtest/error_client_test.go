// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/extension/extensionauth"
)

func TestErrorClient(t *testing.T) {
	client := NewErrorClient(errors.New("error"))

	httpClient, ok := client.(extensionauth.HTTPClient)
	require.True(t, ok)
	_, err := httpClient.RoundTripper(nil)
	require.Error(t, err)

	grpcClient, ok := client.(extensionauth.GRPCClient)
	require.True(t, ok)
	_, err = grpcClient.PerRPCCredentials()
	require.Error(t, err)
}
