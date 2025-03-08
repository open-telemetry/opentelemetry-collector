// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorClient(t *testing.T) {
	client, err := NewErrorClient()
	require.NoError(t, err)

	_, err = client.RoundTripper(nil)
	require.Error(t, err)

	_, err = client.PerRPCCredentials()
	require.Error(t, err)
}
