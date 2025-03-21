// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limitermiddlewareextension // import "go.opentelemetry.io/collector/extension/limitermiddlewareextension"

import (
	"context"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configlimiter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

// oneRequestWeights represents the weights to apply for each request
var oneRequestWeights = []extensionlimiter.Weight{
	{extensionlimiter.WeightKeyRequestCount, 1},
}

// limiterMiddleware implements rate limiting across various transports.
type limiterMiddleware struct {
	id      configlimiter.Limiter
	limiter extensionlimiter.Limiter
}

// newLimiterMiddleware creates a new limiter middleware instance.
func newLimiterMiddleware(_ context.Context, cfg *Config, _ extension.Settings) (*limiterMiddleware, error) {
	return &limiterMiddleware{
		id: cfg.Limiter,
	}, nil
}

// Ensure limiterMiddleware implements all required interfaces.
var _ extensionmiddleware.HTTPClient = &limiterMiddleware{}
var _ extensionmiddleware.HTTPServer = &limiterMiddleware{}
var _ extensionmiddleware.GRPCClient = &limiterMiddleware{}
var _ extensionmiddleware.GRPCServer = &limiterMiddleware{}
var _ component.Component = &limiterMiddleware{}

// Start initializes the limiter by getting it from host extensions.
func (lm *limiterMiddleware) Start(ctx context.Context, host component.Host) error {
	limiter, err := lm.id.GetLimiter(ctx, host.GetExtensions())
	if err != nil {
		return err
	}
	lm.limiter = limiter
	return nil
}

// Shutdown cleans up resources used by the limiter middleware.
func (lm *limiterMiddleware) Shutdown(_ context.Context) error {
	return nil
}

type limiterRoundTripper struct {
	base    http.RoundTripper
	limiter extensionlimiter.Limiter
}

// limitExceeded creates an HTTP 429 (Too Many Requests) response from the client.
func limitExceeded(req *http.Request) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusTooManyRequests,
		Status:        http.StatusText(http.StatusTooManyRequests),
		Request:       req,
		ContentLength: 0,
		Body:          http.NoBody,
	}
}

// ClientRoundTripper returns an HTTP roundtripper that applies rate limiting.
func (lm *limiterMiddleware) ClientRoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	return &limiterRoundTripper{
		base:    base,
		limiter: lm.limiter,
	}, nil
}

// RoundTrip implements http.RoundTripper with rate limiting.
func (lrt *limiterRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rel, err := lrt.limiter.Acquire(req.Context(), oneRequestWeights)
	if err != nil {
		// Return HTTP 429 Too Many Requests status from the server
		return limitExceeded(req), nil
	}
	defer rel()
	return lrt.base.RoundTrip(req)
}

// ServerHandler wraps an HTTP handler with rate limiting.
func (lm *limiterMiddleware) ServerHandler(base http.Handler) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel, err := lm.limiter.Acquire(r.Context(), oneRequestWeights)
		if err != nil {
			// Return HTTP 429 Too Many Requests status
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		defer rel()
		base.ServeHTTP(w, r)
	}), nil
}

// ClientUnaryInterceptor returns a gRPC interceptor for unary client calls.
func (lm *limiterMiddleware) ClientUnaryInterceptor() (grpc.UnaryClientInterceptor, error) {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		rel, err := lm.limiter.Acquire(ctx, oneRequestWeights)
		if err != nil {
			return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
		}
		defer rel()
		return invoker(ctx, method, req, reply, cc, opts...)
	}, nil
}

// ClientStreamInterceptor returns a gRPC interceptor for streaming client calls.
func (lm *limiterMiddleware) ClientStreamInterceptor() (grpc.StreamClientInterceptor, error) {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		cstream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}
		return lm.wrapClientStream(cstream, method), nil
	}, nil
}

// ClientStatsHandler returns a gRPC stats handler for client-side operations.
func (lm *limiterMiddleware) ClientStatsHandler() (stats.Handler, error) {
	return &limiterStatsHandler{
		limiter:  lm.limiter,
		isClient: true,
	}, nil
}

// ServerUnaryInterceptor returns a gRPC interceptor for unary server calls.
func (lm *limiterMiddleware) ServerUnaryInterceptor() (grpc.UnaryServerInterceptor, error) {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		rel, err := lm.limiter.Acquire(ctx, oneRequestWeights)
		if err != nil {
			return nil, status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
		}
		defer rel()
		return handler(ctx, req)
	}, nil
}

// ServerStreamInterceptor returns a gRPC interceptor for streaming server calls.
func (lm *limiterMiddleware) ServerStreamInterceptor() (grpc.StreamServerInterceptor, error) {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		return handler(srv, lm.wrapServerStream(ss, info))
	}, nil
}

// ServerStatsHandler returns a gRPC stats handler for server-side operations.
func (lm *limiterMiddleware) ServerStatsHandler() (stats.Handler, error) {
	return &limiterStatsHandler{
		limiter:  lm.limiter,
		isClient: false,
	}, nil
}

// limiterStatsHandler implements the stats.Handler interface for rate limiting.
type limiterStatsHandler struct {
	limiter  extensionlimiter.Limiter
	isClient bool
}

// TagRPC can attach context information to the given context.
func (h *limiterStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

// HandleRPC processes the RPC stats.
func (h *limiterStatsHandler) HandleRPC(ctx context.Context, _ stats.RPCStats) {
	// Empty implementation for now
}

// TagConn can attach context information to the given context.
func (h *limiterStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleConn processes the connection stats.
func (h *limiterStatsHandler) HandleConn(ctx context.Context, _ stats.ConnStats) {
	// Empty implementation for now
}

type serverStream struct {
	grpc.ServerStream
	limiter extensionlimiter.Limiter
}

// RecvMsg applies rate limiting to server stream message receiving.
func (s *serverStream) RecvMsg(m any) error {
	rel, err := s.limiter.Acquire(s.Context(), oneRequestWeights)
	if err != nil {
		return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
	}
	defer rel()
	return s.ServerStream.RecvMsg(m)
}

// wrapServerStream wraps a gRPC server stream with rate limiting.
func (lm *limiterMiddleware) wrapServerStream(ss grpc.ServerStream, _ *grpc.StreamServerInfo) grpc.ServerStream {
	return &serverStream{
		ServerStream: ss,
		limiter:      lm.limiter,
	}
}

type clientStream struct {
	grpc.ClientStream
	limiter extensionlimiter.Limiter
}

// SendMsg applies rate limiting to client stream message sending.
func (s *clientStream) SendMsg(m any) error {
	rel, err := s.limiter.Acquire(s.Context(), oneRequestWeights)
	if err != nil {
		return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
	}
	defer rel()
	return s.ClientStream.SendMsg(m)
}

// wrapClientStream wraps a gRPC client stream with rate limiting.
func (lm *limiterMiddleware) wrapClientStream(cs grpc.ClientStream, _ string) grpc.ClientStream {
	return &clientStream{
		ClientStream: cs,
		limiter:      lm.limiter,
	}
}
