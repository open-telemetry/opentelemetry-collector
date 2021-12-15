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

package configauth

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestDefaultValues(t *testing.T) {
	// prepare
	e := NewServerAuthenticator()

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

func TestWithAuthenticateFunc(t *testing.T) {
	// prepare
	authCalled := false
	e := NewServerAuthenticator(
		WithAuthenticate(func(ctx context.Context, headers map[string][]string) (context.Context, error) {
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

func TestWithGRPCStreamInterceptor(t *testing.T) {
	called := false
	e := NewServerAuthenticator(WithGRPCStreamInterceptor(func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler, authenticate AuthenticateFunc) error {
		called = true
		return nil
	}))

	// test
	err := e.GRPCStreamServerInterceptor(nil, nil, nil, nil)

	// verify
	assert.True(t, called)
	assert.NoError(t, err)
}

func TestWithGRPCUnaryInterceptor(t *testing.T) {
	called := false
	e := NewServerAuthenticator(WithGRPCUnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler, authenticate AuthenticateFunc) (interface{}, error) {
		called = true
		return nil, nil
	}))

	// test
	_, err := e.GRPCUnaryServerInterceptor(context.Background(), nil, nil, nil)

	// verify
	assert.True(t, called)
	assert.NoError(t, err)
}

func TestWithHTTPInterceptor(t *testing.T) {
	called := false
	e := NewServerAuthenticator(WithHTTPInterceptor(func(handler http.Handler, authenticate AuthenticateFunc) http.Handler {
		called = true
		return handler
	}))

	// test
	e.HTTPInterceptor(nil)

	// verify
	assert.True(t, called)
}

func TestWithStart(t *testing.T) {
	called := false
	e := NewServerAuthenticator(WithStart(func(c context.Context, h component.Host) error {
		called = true
		return nil
	}))

	// test
	err := e.Start(context.Background(), componenttest.NewNopHost())

	// verify
	assert.True(t, called)
	assert.NoError(t, err)
}

func TestWithShutdown(t *testing.T) {
	called := false
	e := NewServerAuthenticator(WithShutdown(func(c context.Context) error {
		called = true
		return nil
	}))

	// test
	err := e.Shutdown(context.Background())

	// verify
	assert.True(t, called)
	assert.NoError(t, err)
}
