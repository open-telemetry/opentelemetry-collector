package grpclimiter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

func NewClientLimiter(host component.Host, middleware configmiddleware.Config) (extensionmiddleware.GRPCClient, error) {
	wp, err1 := limiterhelper.MiddlewareToLimiterWrapperProvider(host, middleware)
	rp, err2 := limiterhelper.MiddlewareToRateLimiterProvider(host, middleware)
	if err := multierr.Append(err1, err2); err != nil {
		return nil, err
	}
	requestLimiter, err3 := wp.GetLimiterWrapper(extensionlimiter.WeightKeyRequestCount)
	bytesLimiter, err4 := rp.GetRateLimiter(extensionlimiter.WeightKeyNetworkBytes)
	if err := multierr.Append(err3, err4); err != nil {
		return nil, err
	}

	var gopts []grpc.DialOption
	if requestLimiter != nil {
		gopts = append(gopts, grpc.WithUnaryInterceptor(
			func(
				ctxIn context.Context,
				method string,
				req, reply any,
				cc *grpc.ClientConn,
				invoker grpc.UnaryInvoker,
				opts ...grpc.CallOption,
			) error {
				return requestLimiter.LimitCall(
					ctxIn, 1,
					func(ctx context.Context) error {
						return invoker(ctx, method, req, reply, cc, opts...)
					})
			}),
			grpc.WithStreamInterceptor(
				func(
					ctxIn context.Context,
					desc *grpc.StreamDesc,
					cc *grpc.ClientConn,
					method string,
					streamer grpc.Streamer,
					opts ...grpc.CallOption,
				) (grpc.ClientStream, error) {
					cstream, err := streamer(ctxIn, desc, cc, method, opts...)
					if err != nil {
						return nil, err
					}
					return wrapClientStream(cstream, method, requestLimiter), nil
				}),
		)
	}
	if bytesLimiter != nil {
		gopts = append(gopts, grpc.WithStatsHandler(
			&limiterStatsHandler{
				bytesLimiter: bytesLimiter,
				isClient:     true,
			}))
	}
	return extensionmiddleware.GetGRPCClientOptionsFunc(func() ([]grpc.DialOption, error) {
		return gopts, nil
	}), nil
}

func NewServerLimiter(host component.Host, middleware configmiddleware.Config) (extensionmiddleware.GRPCServer, error) {
	wp, err1 := limiterhelper.MiddlewareToLimiterWrapperProvider(host, middleware)
	rp, err2 := limiterhelper.MiddlewareToRateLimiterProvider(host, middleware)
	if err := multierr.Append(err1, err2); err != nil {
		return nil, err
	}
	requestLimiter, err3 := wp.GetLimiterWrapper(extensionlimiter.WeightKeyRequestCount)
	bytesLimiter, err4 := rp.GetRateLimiter(extensionlimiter.WeightKeyNetworkBytes)
	if err := multierr.Append(err3, err4); err != nil {
		return nil, err
	}

	var gopts []grpc.ServerOption
	if requestLimiter != nil {
		gopts = append(gopts, grpc.ChainUnaryInterceptor(
			func(
				ctxIn context.Context,
				req any,
				info *grpc.UnaryServerInfo,
				handler grpc.UnaryHandler,
			) (any, error) {
				var resp any
				err := requestLimiter.LimitCall(
					ctxIn, 1,
					func(ctx context.Context) error {
						var err error
						resp, err = handler(ctx, req)
						return err
					})
				return resp, err
			}), grpc.ChainStreamInterceptor(
			func(
				srv interface{},
				ss grpc.ServerStream,
				info *grpc.StreamServerInfo,
				handler grpc.StreamHandler,
			) error {
				return handler(srv, wrapServerStream(ss, info, requestLimiter))
			}),
		)
	}
	if bytesLimiter != nil {
		gopts = append(gopts, grpc.StatsHandler(
			&limiterStatsHandler{
				bytesLimiter: bytesLimiter,
				isClient:     false,
			}))
	}

	return extensionmiddleware.GetGRPCServerOptionsFunc(func() ([]grpc.ServerOption, error) {
		return gopts, nil
	}), nil
}

// limiterStatsHandler implements the stats.Handler interface for rate limiting.
type limiterStatsHandler struct {
	bytesLimiter extensionlimiter.RateLimiter
	isClient     bool
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
	// TODO: How does the limiter break the stream?
	_ = h.bytesLimiter.WaitForRate(ctx, uint64(wireBytes))
}

func (h *limiterStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *limiterStatsHandler) HandleConn(ctx context.Context, _ stats.ConnStats) {
}

type serverStream struct {
	grpc.ServerStream
	limiter limiterhelper.LimiterWrapper
}

// RecvMsg applies rate limiting to server stream message receiving.
func (s *serverStream) RecvMsg(m any) error {
	return s.limiter.LimitCall(
		s.Context(), 1,
		func(_ context.Context) error {
			return s.ServerStream.RecvMsg(m)
		})
}

// wrapServerStream wraps a gRPC server stream with rate limiting.
func wrapServerStream(ss grpc.ServerStream, _ *grpc.StreamServerInfo, limiter limiterhelper.LimiterWrapper) grpc.ServerStream {
	return &serverStream{
		ServerStream: ss,
		limiter:      limiter,
	}
}

type clientStream struct {
	grpc.ClientStream
	limiter limiterhelper.LimiterWrapper
}

// SendMsg applies rate limiting to client stream message sending.
func (s *clientStream) SendMsg(m any) error {
	return s.limiter.LimitCall(
		s.Context(), 1,
		func(_ context.Context) error {
			return s.ClientStream.SendMsg(m)
		})
}

// wrapClientStream wraps a gRPC client stream with rate limiting.
func wrapClientStream(cs grpc.ClientStream, _ string, limiter limiterhelper.LimiterWrapper) grpc.ClientStream {
	return &clientStream{
		ClientStream: cs,
		limiter:      limiter,
	}
}
