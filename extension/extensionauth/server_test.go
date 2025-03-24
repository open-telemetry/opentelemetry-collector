// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerAuthenticateFunc(t *testing.T) {
	var called bool
	var server Server = ServerAuthenticateFunc(func(ctx context.Context, _ map[string][]string) (context.Context, error) {
		called = true
		return ctx, nil
	})

	ctx, err := server.Authenticate(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, called)
	assert.NotNil(t, ctx)
}
