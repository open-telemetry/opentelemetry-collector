// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limitermiddlewareextension // import "go.opentelemetry.io/collector/extension/limitermiddlewareextension"

import (
	"context"
	"io"
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

// tooManyRequestsMsg is the standard text for 429 status code
var tooManyRequestsMsg = http.StatusText(http.StatusTooManyRequests)

// oneRequestWeights represents the weights to apply for each request
var oneRequestWeights = [1]extensionlimiter.Weight{
	{Key: extensionlimiter.WeightKeyRequestCount, Value: 1},
}

// networkByteWeights represents the weights to apply for network bytes
var networkByteWeights = func(bytes int) [1]extensionlimiter.Weight {
	return [1]extensionlimiter.Weight{
		{Key: extensionlimiter.WeightKeyNetworkBytes, Value: uint64(bytes)},
	}
}

// limiterMiddleware implements rate limiting across various transports.
type limiterMiddleware struct {
	id              configlimiter.Limiter
	resourceLimiter extensionlimiter.ResourceLimiter
	rateLimiter     extensionlimiter.RateLimiter
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
	resourceLimiter, err := lm.id.GetResourceLimiter(ctx, host.GetExtensions())
	if err != nil {
		return err
	}
	lm.resourceLimiter = resourceLimiter

	rateLimiter, err := lm.id.GetRateLimiter(ctx, host.GetExtensions())
	if err != nil {
		return err
	}
	lm.rateLimiter = rateLimiter

	return nil
}

// Shutdown cleans up resources used by the limiter middleware.
func (lm *limiterMiddleware) Shutdown(_ context.Context) error {
	return nil
}

type limiterRoundTripper struct {
	base            http.RoundTripper
	resourceLimiter extensionlimiter.ResourceLimiter
	rateLimiter     extensionlimiter.RateLimiter
}

// limitExceeded creates an HTTP 429 (Too Many Requests) response from the client.
func limitExceeded(req *http.Request) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusTooManyRequests,
		Status:        tooManyRequestsMsg,
		Request:       req,
		ContentLength: 0,
		Body:          http.NoBody,
	}
}

// ClientRoundTripper returns an HTTP roundtripper that applies rate limiting.
func (lm *limiterMiddleware) ClientRoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	return &limiterRoundTripper{
		base:            base,
		resourceLimiter: lm.resourceLimiter,
		rateLimiter:     lm.rateLimiter,
	}, nil
}

// rateLimitedBody wraps an http.Request.Body to track bytes and call the rate limiter
type rateLimitedBody struct {
	body        io.ReadCloser
	rateLimiter extensionlimiter.RateLimiter
	ctx         context.Context
}

// Read implements io.Reader interface, counting bytes as they are read
func (rb *rateLimitedBody) Read(p []byte) (n int, err error) {
	n, err = rb.body.Read(p)
	if n > 0 {
		// Apply rate limiting based on network bytes after they are read
		weights := networkByteWeights(n)
		limitErr := rb.rateLimiter.Limit(rb.ctx, weights[:])
		if limitErr != nil {
			// If the rate limiter rejects the bytes, return the error
			return n, limitErr
		}
	}
	return n, err
}

// Close implements io.Closer interface
func (rb *rateLimitedBody) Close() error {
	return rb.body.Close()
}

// RoundTrip implements http.RoundTripper with rate limiting.
func (lrt *limiterRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Apply resource limit check for request count
	rel, err := lrt.resourceLimiter.Acquire(req.Context(), oneRequestWeights[:])
	if err != nil {
		return limitExceeded(req), nil
	}

	// Create a new request with a body that tracks network bytes
	newReq := req.Clone(req.Context())
	if req.Body != nil && req.Body != http.NoBody {
		newReq.Body = &rateLimitedBody{
			body:        req.Body,
			rateLimiter: lrt.rateLimiter,
			ctx:         req.Context(),
		}
	}

	if rel != nil {
		defer rel()
	}

	return lrt.base.RoundTrip(newReq)
}

// ServerHandler wraps an HTTP handler with rate limiting.
func (lm *limiterMiddleware) ServerHandler(base http.Handler) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Apply resource limit check for request count
		rel, err := lm.resourceLimiter.Acquire(r.Context(), oneRequestWeights[:])
		if err != nil {
			http.Error(w, tooManyRequestsMsg, http.StatusTooManyRequests)
			return
		}

		// Create a new request with a body that tracks network bytes
		newReq := r.Clone(r.Context())
		if r.Body != nil && r.Body != http.NoBody {
			newReq.Body = &rateLimitedBody{
				body:        r.Body,
				rateLimiter: lm.rateLimiter,
				ctx:         r.Context(),
			}
		}

		if rel != nil {
			defer rel()
		}

		base.ServeHTTP(w, newReq)
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
		rel, err := lm.resourceLimiter.Acquire(ctx, oneRequestWeights[:])
		if err != nil {
			return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
		}
		if rel != nil {
			defer rel()
		}
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
		rateLimiter: lm.rateLimiter,
		isClient:    true,
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
		rel, err := lm.resourceLimiter.Acquire(ctx, oneRequestWeights[:])
		if err != nil {
			return nil, status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
		}
		if rel != nil {
			defer rel()
		}
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
		rateLimiter: lm.rateLimiter,
		isClient:    false,
	}, nil
}

// limiterStatsHandler implements the stats.Handler interface for rate limiting.
type limiterStatsHandler struct {
	rateLimiter extensionlimiter.RateLimiter
	isClient    bool
}

func (h *limiterStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *limiterStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	// Check for payload messages to apply network byte rate limiting
	var wireBytes int
	switch payload := s.(type) {
	case *stats.InPayload:
		// Server receiving payload (or client receiving response)
		if !h.isClient {
			wireBytes = payload.WireLength
		}
	case *stats.OutPayload:
		// Client sending payload (or server sending response)
		if h.isClient {
			wireBytes = payload.WireLength
		}
	default:
		// Not a payload message, no rate limiting to apply
		return
	}

	if wireBytes == 0 {
		return
	}
	// Apply rate limiting based on network bytes
	weights := networkByteWeights(wireBytes)
	h.rateLimiter.Limit(ctx, weights[:])
}

func (h *limiterStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *limiterStatsHandler) HandleConn(ctx context.Context, _ stats.ConnStats) {
}

type serverStream struct {
	grpc.ServerStream
	limiter extensionlimiter.ResourceLimiter
}

// RecvMsg applies rate limiting to server stream message receiving.
func (s *serverStream) RecvMsg(m any) error {
	rel, err := s.limiter.Acquire(s.Context(), oneRequestWeights[:])
	if err != nil {
		return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
	}
	if rel != nil {
		defer rel()
	}
	return s.ServerStream.RecvMsg(m)
}

// wrapServerStream wraps a gRPC server stream with rate limiting.
func (lm *limiterMiddleware) wrapServerStream(ss grpc.ServerStream, _ *grpc.StreamServerInfo) grpc.ServerStream {
	return &serverStream{
		ServerStream: ss,
		limiter:      lm.resourceLimiter,
	}
}

type clientStream struct {
	grpc.ClientStream
	limiter extensionlimiter.ResourceLimiter
}

// SendMsg applies rate limiting to client stream message sending.
func (s *clientStream) SendMsg(m any) error {
	rel, err := s.limiter.Acquire(s.Context(), oneRequestWeights[:])
	if err != nil {
		return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
	}
	if rel != nil {
		defer rel()
	}
	return s.ClientStream.SendMsg(m)
}

// wrapClientStream wraps a gRPC client stream with rate limiting.
func (lm *limiterMiddleware) wrapClientStream(cs grpc.ClientStream, _ string) grpc.ClientStream {
	return &clientStream{
		ClientStream: cs,
		limiter:      lm.resourceLimiter,
	}
}
