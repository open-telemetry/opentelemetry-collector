// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestDefaultValues(t *testing.T) {
	// prepare
	e := NewServer()

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
	e := NewServer(
		WithServerAuthenticate(func(ctx context.Context, headers map[string][]string) (context.Context, error) {
			authCalled = true
			return ctx, nil
		}),
	)

	// test
	_, err := e.Authenticate(context.Background(), make(map[string][]string))

	// verify
	assert.True(t, authCalled)
	assert.NoError(t, err)
}

func TestWithServerStart(t *testing.T) {
	called := false
	e := NewServer(WithServerStart(func(c context.Context, h component.Host) error {
		called = true
		return nil
	}))

	// test
	err := e.Start(context.Background(), componenttest.NewNopHost())

	// verify
	assert.True(t, called)
	assert.NoError(t, err)
}

func TestWithServerShutdown(t *testing.T) {
	called := false
	e := NewServer(WithServerShutdown(func(c context.Context) error {
		called = true
		return nil
	}))

	// test
	err := e.Shutdown(context.Background())

	// verify
	assert.True(t, called)
	assert.NoError(t, err)
}
