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

// limiterMiddleware implements rate limiting across various transports.
type limiterMiddleware struct {
	id       configlimiter.Limiter
	provider extensionlimiter.Provider
}

// newLimiterMiddleware creates a new limiter middleware instance.
func newLimiterMiddleware(_ context.Context, cfg *Config, _ extension.Settings) (*limiterMiddleware, error) {
	return &limiterMiddleware{
		id: cfg.Limiter,
	}, nil
}

// Ensure limiterMiddleware implements all required middleware interfaces.
var (
	_ extensionmiddleware.HTTPClient = &limiterMiddleware{}
	_ extensionmiddleware.HTTPServer = &limiterMiddleware{}
	_ extensionmiddleware.GRPCClient = &limiterMiddleware{}
	_ extensionmiddleware.GRPCServer = &limiterMiddleware{}
	_ component.Component            = &limiterMiddleware{}
)

// limiterMiddleware also implements the extensionlimiter interface.
var _ extensionlimiter.Provider = &limiterMiddleware{}

// RateLimiter implements extensionlimiter.Provider.
func (lm *limiterMiddleware) RateLimiter(key extensionlimiter.WeightKey) extensionlimiter.RateLimiter {
	return lm.provider.RateLimiter(key)
}

// ResourceLimiter implements extensionlimiter.Provider
func (lm *limiterMiddleware) ResourceLimiter(key extensionlimiter.WeightKey) extensionlimiter.ResourceLimiter {
	return lm.provider.ResourceLimiter(key)
}

// Start initializes the limiter by getting it from host extensions.
func (lm *limiterMiddleware) Start(ctx context.Context, host component.Host) error {
	provider, err := lm.id.GetProvider(ctx, host.GetExtensions())
	if err != nil {
		return err
	}
	lm.provider = provider
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

// GetHTTPRoundTripper returns an HTTP roundtripper that applies rate limiting.
func (lm *limiterMiddleware) GetHTTPRoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	resourceLimiter := lm.provider.ResourceLimiter(extensionlimiter.WeightKeyRequestCount)
	rateLimiter := lm.provider.RateLimiter(extensionlimiter.WeightKeyNetworkBytes)

	if resourceLimiter == nil && rateLimiter == nil {
		// If no limiters are configured, return the base round tripper
		return base, nil
	}

	return &limiterRoundTripper{
		base:            base,
		resourceLimiter: resourceLimiter,
		rateLimiter:     rateLimiter,
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
		limitErr := rb.rateLimiter.Limit(rb.ctx, uint64(n))
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
	if lrt.resourceLimiter != nil {
		rel, err := lrt.resourceLimiter.Acquire(req.Context(), 1)
		if err != nil {
			return limitExceeded(req), nil
		}
		if rel != nil {
			defer rel()
		}
	}

	// Create a new request with a body that tracks network bytes
	newReq := req.Clone(req.Context())
	if lrt.rateLimiter != nil && req.Body != nil && req.Body != http.NoBody {
		newReq.Body = &rateLimitedBody{
			body:        req.Body,
			rateLimiter: lrt.rateLimiter,
			ctx:         req.Context(),
		}
	}

	return lrt.base.RoundTrip(newReq)
}

// GetHTTPHandler wraps an HTTP handler with rate limiting.
func (lm *limiterMiddleware) GetHTTPHandler(base http.Handler) (http.Handler, error) {
	resourceLimiter := lm.provider.ResourceLimiter(extensionlimiter.WeightKeyRequestCount)
	rateLimiter := lm.provider.RateLimiter(extensionlimiter.WeightKeyNetworkBytes)

	if resourceLimiter == nil && rateLimiter == nil {
		return base, nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Apply resource limit check for request count
		if resourceLimiter != nil {
			rel, err := resourceLimiter.Acquire(r.Context(), 1)
			if err != nil {
				http.Error(w, tooManyRequestsMsg, http.StatusTooManyRequests)
				return
			}

			if rel != nil {
				defer rel()
			}
		}

		if rateLimiter != nil {
			// Create a new request with a body that tracks network bytes
			newReq := r.Clone(r.Context())
			if r.Body != nil && r.Body != http.NoBody {
				newReq.Body = &rateLimitedBody{
					body:        r.Body,
					rateLimiter: rateLimiter,
					ctx:         r.Context(),
				}
			}
			r = newReq
		}

		base.ServeHTTP(w, r)
	}), nil
}

func (lm *limiterMiddleware) GetGRPCClientOptions() (options []grpc.DialOption, _ error) {
	if resourceLimiter := lm.provider.ResourceLimiter(extensionlimiter.WeightKeyRequestCount); resourceLimiter != nil {
		options = append(options, grpc.WithUnaryInterceptor(
			func(
				ctx context.Context,
				method string,
				req, reply any,
				cc *grpc.ClientConn,
				invoker grpc.UnaryInvoker,
				opts ...grpc.CallOption,
			) error {
				rel, err := resourceLimiter.Acquire(ctx, 1)
				if err != nil {
					return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
				}
				if rel != nil {
					defer rel()
				}
				return invoker(ctx, method, req, reply, cc, opts...)
			}),
			grpc.WithStreamInterceptor(
				func(
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
					return lm.wrapClientStream(cstream, method, resourceLimiter), nil
				}),
		)
	}
	if rateLimiter := lm.provider.RateLimiter(extensionlimiter.WeightKeyNetworkBytes); rateLimiter != nil {
		options = append(options, grpc.WithStatsHandler(
			&limiterStatsHandler{
				rateLimiter: rateLimiter,
				isClient:    true,
			}))
	}
	return options, nil
}

// ServerUnaryInterceptor returns a gRPC interceptor for unary server calls.
func (lm *limiterMiddleware) GetGRPCServerOptions() (options []grpc.ServerOption, _ error) {
	if resourceLimiter := lm.provider.ResourceLimiter(extensionlimiter.WeightKeyRequestCount); resourceLimiter != nil {
		options = append(options, grpc.ChainUnaryInterceptor(
			// The unary resource case
			func(
				ctx context.Context,
				req any,
				info *grpc.UnaryServerInfo,
				handler grpc.UnaryHandler,
			) (any, error) {
				rel, err := resourceLimiter.Acquire(ctx, 1)
				if err != nil {
					return nil, status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
				}
				if rel != nil {
					defer rel()
				}
				return handler(ctx, req)
			}), grpc.ChainStreamInterceptor(
			// The stream resource case
			func(
				srv interface{},
				ss grpc.ServerStream,
				info *grpc.StreamServerInfo,
				handler grpc.StreamHandler,
			) error {
				return handler(srv, lm.wrapServerStream(ss, info, resourceLimiter))
			}),
		)
	}
	if rateLimiter := lm.provider.RateLimiter(extensionlimiter.WeightKeyNetworkBytes); rateLimiter != nil {
		options = append(options, grpc.StatsHandler(
			&limiterStatsHandler{
				rateLimiter: rateLimiter,
				isClient:    false,
			}))
	}

	return options, nil
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
	h.rateLimiter.Limit(ctx, uint64(wireBytes))
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
	rel, err := s.limiter.Acquire(s.Context(), 1)
	if err != nil {
		return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
	}
	if rel != nil {
		defer rel()
	}
	return s.ServerStream.RecvMsg(m)
}

// wrapServerStream wraps a gRPC server stream with rate limiting.
func (lm *limiterMiddleware) wrapServerStream(ss grpc.ServerStream, _ *grpc.StreamServerInfo, limiter extensionlimiter.ResourceLimiter) grpc.ServerStream {
	return &serverStream{
		ServerStream: ss,
		limiter:      limiter,
	}
}

type clientStream struct {
	grpc.ClientStream
	limiter extensionlimiter.ResourceLimiter
}

// SendMsg applies rate limiting to client stream message sending.
func (s *clientStream) SendMsg(m any) error {
	rel, err := s.limiter.Acquire(s.Context(), 1)
	if err != nil {
		return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
	}
	if rel != nil {
		defer rel()
	}
	return s.ClientStream.SendMsg(m)
}

// wrapClientStream wraps a gRPC client stream with rate limiting.
func (lm *limiterMiddleware) wrapClientStream(cs grpc.ClientStream, _ string, limiter extensionlimiter.ResourceLimiter) grpc.ClientStream {
	return &clientStream{
		ClientStream: cs,
		limiter:      limiter,
	}
}
