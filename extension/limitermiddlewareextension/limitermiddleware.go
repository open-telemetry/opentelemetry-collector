// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limitermiddlewareextension // import "go.opentelemetry.io/collector/extension/limitermiddlewareextension"

import (
	"context"
	"net/http"

	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configlimiter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

type limiterMiddleware struct {
	id      configlimiter.Limiter
	limiter extensionlimiter.Limiter
}

func newLimiterMiddleware(_ context.Context, cfg *Config, _ extension.Settings) (*limiterMiddleware, error) {
	return &limiterMiddleware{
		id: cfg.Limiter,
	}, nil
}

var _ extensionmiddleware.HTTPClient = &limiterMiddleware{}
var _ extensionmiddleware.HTTPServer = &limiterMiddleware{}
var _ extensionmiddleware.GRPCClient = &limiterMiddleware{}
var _ extensionmiddleware.GRPCServer = &limiterMiddleware{}
var _ component.Component = &limiterMiddleware{}

func (lm *limiterMiddleware) Start(ctx context.Context, host component.Host) error {
	limiter, err := lm.id.GetLimiter(ctx, host.GetExtensions())
	if err != nil {
		return err
	}
	lm.limiter = limiter
	return nil
}

func (lm *limiterMiddleware) Shutdown(_ context.Context) error {
	return nil
}

type limiterRoundTripper struct {
	base    http.RoundTripper
	limiter extensionlimiter.Limiter
}

func (lm *limiterMiddleware) ClientRoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	return &limiterRoundTripper{
		base:    base,
		limiter: lm.limiter,
	}, nil
}

func (lrt *limiterRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return lrt.base.RoundTrip(req)
}

func (lm *limiterMiddleware) ServerHandler(base http.Handler) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base.ServeHTTP(w, r)
	}), nil
}

func (lm *limiterMiddleware) ClientUnaryInterceptor() (grpc.UnaryClientInterceptor, error) {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return invoker(ctx, method, req, reply, cc, opts...)
	}, nil
}

func (lm *limiterMiddleware) ClientStreamInterceptor() (grpc.StreamClientInterceptor, error) {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		return streamer(ctx, desc, cc, method, opts...)
	}, nil
}

func (lm *limiterMiddleware) ServerUnaryInterceptor() (grpc.UnaryServerInterceptor, error) {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		return handler(ctx, req)
	}, nil
}

func (lm *limiterMiddleware) ServerStreamInterceptor() (grpc.StreamServerInterceptor, error) {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		return handler(srv, ss)
	}, nil
}
