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

package componenthelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"go.opentelemetry.io/collector/component"
)

type mockComponent struct {
	mock.Mock
}

func (mc *mockComponent) Start(ctx context.Context, host component.Host) error {
	args := mc.Called(ctx, host)
	return args.Error(0)
}

func (mc *mockComponent) Shutdown(ctx context.Context) error {
	args := mc.Called(ctx)
	return args.Error(0)
}

// MethodOption allows for additional expecations to be set for a method.
type MethodOption func(*mock.Call)

// WithMethodCalled sets the expectation for the number of called allowed
// for the associated method.
func WithMethodCalled(times int) MethodOption {
	return func(c *mock.Call) {
		if times < 1 {
			c.Maybe()
			times = 0
		}
		c.Times(times)
	}
}

// ComponentOption allows for expectations within a mock to be set
// within a test table configuration.
type ComponentOption func(*mockComponent)

func WithAssertStart(expectedCtx context.Context, expectedHost component.Host, returnedErr error, opts ...MethodOption) ComponentOption {
	return func(mlc *mockComponent) {
		call := mlc.On("Start", expectedCtx, expectedHost).Return(returnedErr)
		for _, opt := range opts {
			opt(call)
		}
	}
}

func WithAssertShutdown(expectedCtx context.Context, returnedErr error, opts ...MethodOption) ComponentOption {
	return func(mlc *mockComponent) {
		call := mlc.On("Shutdown", expectedCtx).Return(returnedErr)
		for _, opt := range opts {
			opt(call)
		}
	}
}

func newMockComponent(tb testing.TB, opts ...ComponentOption) component.Component {
	m := &mockComponent{}
	for _, opt := range opts {
		opt(m)
	}
	tb.Cleanup(func() {
		m.AssertExpectations(tb)
	})
	return m
}
