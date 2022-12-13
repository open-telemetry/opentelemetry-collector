// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestClientDefaultValues(t *testing.T) {
	// prepare
	e := NewClient()

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
	e := NewClient(WithClientStart(func(c context.Context, h component.Host) error {
		called = true
		return nil
	}))

	// test
	err := e.Start(context.Background(), componenttest.NewNopHost())

	// verify
	assert.True(t, called)
	assert.NoError(t, err)
}

func TestWithClientShutdown(t *testing.T) {
	called := false
	e := NewClient(WithClientShutdown(func(c context.Context) error {
		called = true
		return nil
	}))

	// test
	err := e.Shutdown(context.Background())

	// verify
	assert.True(t, called)
	assert.NoError(t, err)
}

func TestWithClientRoundTripper(t *testing.T) {
	called := false
	e := NewClient(WithClientRoundTripper(func(base http.RoundTripper) (http.RoundTripper, error) {
		called = true
		return base, nil
	}))

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
	e := NewClient(WithClientPerRPCCredentials(func() (credentials.PerRPCCredentials, error) {
		called = true
		return &customPerRPCCredentials{}, nil
	}))

	// test
	p, err := e.PerRPCCredentials()

	// verify
	assert.True(t, called)
	assert.NotNil(t, p)
	assert.NoError(t, err)

}
