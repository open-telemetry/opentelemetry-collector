// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type exportHandler func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error)

// An interceptor is free to reject a request without ever invoking the handler that takes the
// bound State. The export handlers must still drop the binding, or every rejected request would
// pin a State, its arenas and the cloned wire buffer for the life of the process.
func TestExportHandlerReleasesStateWhenInterceptorRejects(t *testing.T) {
	handlers := map[string]exportHandler{
		"logs":     logsServiceExportHandler,
		"traces":   traceServiceExportHandler,
		"metrics":  metricsServiceExportHandler,
		"profiles": profilesServiceExportHandler,
	}

	errRejected := errors.New("rejected by interceptor")
	reject := func(_ context.Context, _ any, _ *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
		return nil, errRejected
	}

	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			_, err := handler(nil, context.Background(), func(any) error { return nil }, reject)
			require.ErrorIs(t, err, errRejected)

			bindings := 0
			grpcRequestStates.Range(func(_, _ any) bool {
				bindings++
				return true
			})
			assert.Zero(t, bindings, "rejected request left a State bound")
		})
	}
}
