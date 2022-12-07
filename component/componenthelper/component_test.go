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

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

type Operation func(c component.Component) error

func StartOperation(ctx context.Context, host component.Host) Operation {
	return func(c component.Component) error {
		return c.Start(ctx, host)
	}
}

func ShutdownOperation(ctx context.Context) Operation {
	return func(c component.Component) error {
		return c.Shutdown(ctx)
	}
}

func TestComponentLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	host := componenttest.NewNopHost()

	for _, tc := range []struct {
		name       string
		operations []Operation
		compOpts   []ComponentOption
	}{
		{
			name: "simple lifecycle",
			operations: []Operation{
				StartOperation(ctx, host),
				ShutdownOperation(ctx),
			},
			compOpts: []ComponentOption{
				WithAssertStart(ctx, host, nil, WithMethodCalled(1)),
				WithAssertShutdown(ctx, nil, WithMethodCalled(1)),
			},
		},
		{
			name: "restarted component",
			operations: []Operation{
				StartOperation(ctx, host),
				ShutdownOperation(ctx),
				StartOperation(ctx, host),
				ShutdownOperation(ctx),
			},
			compOpts: []ComponentOption{
				WithAssertStart(ctx, host, nil, WithMethodCalled(2)),
				WithAssertShutdown(ctx, nil, WithMethodCalled(2)),
			},
		},
		{
			name: "not started component",
			operations: []Operation{
				ShutdownOperation(ctx),
				ShutdownOperation(ctx),
			},
			compOpts: []ComponentOption{},
		},
		{
			name: "repeated calls to start",
			operations: []Operation{
				StartOperation(ctx, host),
				StartOperation(ctx, host),
				ShutdownOperation(ctx),
			},
			compOpts: []ComponentOption{
				WithAssertStart(ctx, host, nil, WithMethodCalled(1)),
				WithAssertShutdown(ctx, nil, WithMethodCalled(1)),
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newMockComponent(t, tc.compOpts...)
			comp := NewComponent(m.Start, m.Shutdown)

			for _, op := range tc.operations {
				assert.NoError(t, op(comp), "Must not error when processing operations")
			}
		})
	}
}
