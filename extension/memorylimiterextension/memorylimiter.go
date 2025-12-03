// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterextension // import "go.opentelemetry.io/collector/extension/memorylimiterextension"

import (
	"context"
	"net/http"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/internal/memorylimiter"
)

var (
	_ extensionmiddleware.GRPCServer = (*memoryLimiterExtension)(nil)
	_ extensionmiddleware.HTTPServer = (*memoryLimiterExtension)(nil)
)

type memoryLimiterExtension struct {
	memLimiter *memorylimiter.MemoryLimiter
}

// newMemoryLimiter returns a new memorylimiter extension.
func newMemoryLimiter(cfg *Config, logger *zap.Logger) (*memoryLimiterExtension, error) {
	ml, err := memorylimiter.NewMemoryLimiter(cfg, logger)
	if err != nil {
		return nil, err
	}

	return &memoryLimiterExtension{memLimiter: ml}, nil
}

func (ml *memoryLimiterExtension) Start(ctx context.Context, host component.Host) error {
	return ml.memLimiter.Start(ctx, host)
}

func (ml *memoryLimiterExtension) Shutdown(ctx context.Context) error {
	return ml.memLimiter.Shutdown(ctx)
}

// MustRefuse returns if the caller should deny because memory has reached it's configured limits
func (ml *memoryLimiterExtension) MustRefuse() bool {
	return ml.memLimiter.MustRefuse()
}

func (ml *memoryLimiterExtension) GetHTTPHandler(base http.Handler) (http.Handler, error) {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if ml.MustRefuse() {
			http.Error(resp, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		base.ServeHTTP(resp, req)
	}), nil
}

func (ml *memoryLimiterExtension) GetGRPCServerOptions() ([]grpc.ServerOption, error) {
	return []grpc.ServerOption{grpc.UnaryInterceptor(
		func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
			if ml.MustRefuse() {
				return nil, status.Errorf(codes.ResourceExhausted, "RESOURCE_EXHAUSTED")
			}
			return handler(ctx, req)
		},
	)}, nil
}
