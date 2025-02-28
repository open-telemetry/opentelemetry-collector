// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestDefaultValues(t *testing.T) {
	// prepare
	e, err := NewServer()
	require.NoError(t, err)

	// test
	t.Run("start", func(t *testing.T) {
		err := e.Start(context.Background(), componenttest.NewNopHost())
		assert.NoError(t, err)
	})

	t.Run("authenticate", func(t *testing.T) {
		ctx, err := e.Authenticate(context.Background(), make(map[string][]string))
		assert.NotNil(t, ctx)
		assert.NoError(t, err)
	})

	t.Run("shutdown", func(t *testing.T) {
		err := e.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

func TestWithServerAuthenticateFunc(t *testing.T) {
	// prepare
	authCalled := false
	e, err := NewServer(
		WithServerAuthenticate(func(ctx context.Context, _ map[string][]string) (context.Context, error) {
			authCalled = true
			return ctx, nil
		}),
	)
	require.NoError(t, err)

	// test
	_, err = e.Authenticate(context.Background(), make(map[string][]string))

	// verify
	assert.True(t, authCalled)
	assert.NoError(t, err)
}

func TestWithServerStart(t *testing.T) {
	called := false
	e, err := NewServer(WithServerStart(func(context.Context, component.Host) error {
		called = true
		return nil
	}))
	require.NoError(t, err)

	// test
	err = e.Start(context.Background(), componenttest.NewNopHost())

	// verify
	assert.True(t, called)
	assert.NoError(t, err)
}

func TestWithServerShutdown(t *testing.T) {
	called := false
	e, err := NewServer(WithServerShutdown(func(context.Context) error {
		called = true
		return nil
	}))
	require.NoError(t, err)

	// test
	err = e.Shutdown(context.Background())

	// verify
	assert.True(t, called)
	assert.NoError(t, err)
}
