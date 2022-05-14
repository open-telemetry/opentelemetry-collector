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

// This is a test added by us.
package configgrpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

func TestRegisterClientDialOptionHandler(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	gcs := &GRPCClientSettings{}
	opts, err := gcs.ToDialOptions(
		&mockHost{ext: map[config.ComponentID]component.Extension{}},
		tt.TelemetrySettings,
	)
	require.NoError(t, err)

	defaultOptsLen := len(opts)

	RegisterClientDialOptionHandlers(func() grpc.DialOption {
		return grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			return invoker(ctx, method, req, reply, cc, opts...)
		})
	})
	gcs = &GRPCClientSettings{}
	opts, err = gcs.ToDialOptions(
		&mockHost{ext: map[config.ComponentID]component.Extension{}},
		tt.TelemetrySettings,
	)
	assert.NoError(t, err)
	assert.Len(t, opts, defaultOptsLen+1)
}
