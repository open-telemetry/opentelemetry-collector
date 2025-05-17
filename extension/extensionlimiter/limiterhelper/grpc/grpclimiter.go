package grpclimiter

import (
	"context"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

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
				release, err := resourceLimiter.Acquire(ctx, 1)
				defer release()
				if err != nil {
					return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
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
				release, err := resourceLimiter.Acquire(ctx, 1)
				defer release()
				if err != nil {
					return nil, status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
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
	release, err := s.limiter.Acquire(s.Context(), 1)
	defer release()
	if err != nil {
		return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
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
	release, err := s.limiter.Acquire(s.Context(), 1)
	defer release()
	if err != nil {
		return status.Errorf(codes.ResourceExhausted, "limit exceeded: %v", err)
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
