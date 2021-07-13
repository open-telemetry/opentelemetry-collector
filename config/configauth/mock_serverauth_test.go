// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthenticateFunc(t *testing.T) {
	// prepare
	m := &MockAuthenticator{}
	called := false
	m.AuthenticateFunc = func(c context.Context, m map[string][]string) (context.Context, error) {
		called = true
		return context.Background(), nil
	}

	// test
	ctx, err := m.Authenticate(context.Background(), nil)

	// verify
	assert.NoError(t, err)
	assert.True(t, called)
	assert.NotNil(t, ctx)
}

func TestNilOperations(t *testing.T) {
	// prepare
	m := &MockAuthenticator{}

	// test and verify
	origCtx := context.Background()

	{
		ctx, err := m.Authenticate(origCtx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, ctx)
	}

	{
		ret, err := m.GRPCUnaryServerInterceptor(origCtx, nil, nil, nil)
		assert.Nil(t, ret)
		assert.NoError(t, err)
	}

	{
		err := m.GRPCStreamServerInterceptor(nil, nil, nil, nil)
		assert.NoError(t, err)
	}

	{
		err := m.Start(origCtx, nil)
		assert.NoError(t, err)
	}

	{
		err := m.Shutdown(origCtx)
		assert.NoError(t, err)
	}

}
