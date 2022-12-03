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
