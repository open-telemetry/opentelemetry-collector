// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/extension/extensionauth"
)

func TestErrorClient(t *testing.T) {
	client := NewErr(errors.New("error"))

	httpClient, ok := client.(extensionauth.HTTPClient)
	require.True(t, ok)
	_, err := httpClient.RoundTripper(nil)
	require.Error(t, err)

	grpcClient, ok := client.(extensionauth.GRPCClient)
	require.True(t, ok)
	_, err = grpcClient.PerRPCCredentials()
	require.Error(t, err)

	server, ok := client.(extensionauth.Server)
	require.True(t, ok)
	_, err = server.Authenticate(context.Background(), map[string][]string{})
	require.Error(t, err)
}
