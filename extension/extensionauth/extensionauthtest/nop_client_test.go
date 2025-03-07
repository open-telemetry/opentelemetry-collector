// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/extension/extensionauth"
)

func TestNopClient(t *testing.T) {
	client := NewNopClient()

	httpClient, ok := client.(extensionauth.HTTPClient)
	require.True(t, ok)
	rt, err := httpClient.RoundTripper(nil)
	require.NoError(t, err)
	assert.Nil(t, rt)

	grpcClient, ok := client.(extensionauth.GRPCClient)
	require.True(t, ok)
	grpcAuth, err := grpcClient.PerRPCCredentials()
	require.NoError(t, err)
	assert.Nil(t, grpcAuth)
}
