// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauth

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestClientDefaultValues(t *testing.T) {
	// prepare
	e, err := NewClient()
	require.NoError(t, err)

	// test
	t.Run("start", func(t *testing.T) {
		err := e.Start(context.Background(), componenttest.NewNopHost())
		assert.NoError(t, err)
	})

	t.Run("roundtripper", func(t *testing.T) {
		ctx, err := e.RoundTripper(http.DefaultTransport)
		assert.NotNil(t, ctx)
		assert.NoError(t, err)
	})

	t.Run("per-rpc-credentials", func(t *testing.T) {
		p, err := e.PerRPCCredentials()
		assert.Nil(t, p)
		assert.NoError(t, err)
	})

	t.Run("shutdown", func(t *testing.T) {
		err := e.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

func TestWithClientStart(t *testing.T) {
	called := false
	e, err := NewClient(WithClientStart(func(context.Context, component.Host) error {
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

func TestWithClientShutdown(t *testing.T) {
	called := false
	e, err := NewClient(WithClientShutdown(func(context.Context) error {
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

func TestWithClientRoundTripper(t *testing.T) {
	called := false
	e, err := NewClient(WithClientRoundTripper(func(base http.RoundTripper) (http.RoundTripper, error) {
		called = true
		return base, nil
	}))
	require.NoError(t, err)

	// test
	rt, err := e.RoundTripper(http.DefaultTransport)

	// verify
	assert.True(t, called)
	assert.NotNil(t, rt)
	assert.NoError(t, err)
}

type customPerRPCCredentials struct{}

var _ credentials.PerRPCCredentials = (*customPerRPCCredentials)(nil)

func (c *customPerRPCCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return nil, nil
}

func (c *customPerRPCCredentials) RequireTransportSecurity() bool {
	return true
}

func TestWithPerRPCCredentials(t *testing.T) {
	called := false
	e, err := NewClient(WithClientPerRPCCredentials(func() (credentials.PerRPCCredentials, error) {
		called = true
		return &customPerRPCCredentials{}, nil
	}))
	require.NoError(t, err)

	// test
	p, err := e.PerRPCCredentials()

	// verify
	assert.True(t, called)
	assert.NotNil(t, p)
	assert.NoError(t, err)
}
