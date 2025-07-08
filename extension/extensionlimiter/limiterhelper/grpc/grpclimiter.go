// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package grpclimiter

import (
	"context"
	"time"

	"go.uber.org/multierr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

// checkRateLimiterError checks if there's a prior rate limiter error in the context.
func checkRateLimiterError(ctx context.Context) error {
	if state := extensionlimiter.GetLimiterTracking(ctx); state != nil {
		return state.GetRateLimiterError()
	}
	return nil
}

// setRateLimiterError sets a rate limiter error in the context
func setRateLimiterError(ctx context.Context, err error) {
	if state := extensionlimiter.GetLimiterTracking(ctx); state != nil {
		state.SetRateLimiterError(err)
	}
}

func NewClientLimiter(ext extensionlimiter.AnyProvider) (extensionmiddleware.GRPCClient, error) {
	wp, err1 := limiterhelper.AnyToWrapperProvider(ext)
	rp, err2 := limiterhelper.AnyToRateLimiterProvider(ext)
	if err := multierr.Append(err1, err2); err != nil {
		return nil, err
	}
	requestLimiter, err3 := wp.GetWrapper(extensionlimiter.WeightKeyRequestCount)
	compressedLimiter, err4 := rp.GetRateLimiter(extensionlimiter.WeightKeyNetworkBytes)
	uncompressedLimiter, err5 := rp.GetRateLimiter(extensionlimiter.WeightKeyRequestBytes)

	if err := multierr.Append(err3, multierr.Append(err4, err5)); err != nil {
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
				// Ensure limiter tracking exists in context and get tracking state
				ctxIn, tracking := extensionlimiter.EnsureLimiterTracking(ctxIn)

				if tracking.HasBeenLimited(extensionlimiter.WeightKeyRequestCount) {
					// Skip limiting since a limiter was already applied for request count
					return invoker(ctxIn, method, req, reply, cc, opts...)
				}

				return requestLimiter.LimitCall(
					ctxIn, 1,
					func(ctx context.Context) error {
						// Mark this weight key as applied
						tracking.AddRequest(extensionlimiter.WeightKeyRequestCount, 1)
						if err := checkRateLimiterError(ctx); err != nil {
							return err
						}
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
	if compressedLimiter != nil || uncompressedLimiter != nil {
		gopts = append(gopts, grpc.WithStatsHandler(
			&limiterStatsHandler{
				compressedLimiter:   compressedLimiter,
				uncompressedLimiter: uncompressedLimiter,
				isClient:            true,
			}))
	}
	return extensionmiddleware.GetGRPCClientOptionsFunc(func() ([]grpc.DialOption, error) {
		return gopts, nil
	}), nil
}

func NewServerLimiter(ext extensionlimiter.AnyProvider) (extensionmiddleware.GRPCServer, error) {
	wp, err1 := limiterhelper.AnyToWrapperProvider(ext)
	rp, err2 := limiterhelper.AnyToRateLimiterProvider(ext)
	if err := multierr.Append(err1, err2); err != nil {
		return nil, err
	}
	requestLimiter, err3 := wp.GetWrapper(extensionlimiter.WeightKeyRequestCount)
	compressedLimiter, err4 := rp.GetRateLimiter(extensionlimiter.WeightKeyNetworkBytes)
	uncompressedLimiter, err5 := rp.GetRateLimiter(extensionlimiter.WeightKeyRequestBytes)
	if err := multierr.Append(err3, multierr.Append(err4, err5)); err != nil {
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
				// Ensure limiter tracking exists in context and get tracking state
				ctxIn, tracking := extensionlimiter.EnsureLimiterTracking(ctxIn)

				if tracking.HasBeenLimited(extensionlimiter.WeightKeyRequestCount) {
					// Skip limiting since a limiter was already applied for request count
					return handler(ctxIn, req)
				}

				var resp any
				err := requestLimiter.LimitCall(
					ctxIn, 1,
					func(ctx context.Context) error {
						// Mark this weight key as applied
						tracking.AddRequest(extensionlimiter.WeightKeyRequestCount, 1)
						if err := checkRateLimiterError(ctx); err != nil {
							return err
						}
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
	if compressedLimiter != nil || uncompressedLimiter != nil {
		gopts = append(gopts, grpc.StatsHandler(
			&limiterStatsHandler{
				compressedLimiter:   compressedLimiter,
				uncompressedLimiter: uncompressedLimiter,
				isClient:            false,
			}))
	}

	return extensionmiddleware.GetGRPCServerOptionsFunc(func() ([]grpc.ServerOption, error) {
		return gopts, nil
	}), nil
}

// limiterStatsHandler implements the stats.Handler interface for rate limiting.
type limiterStatsHandler struct {
	compressedLimiter   extensionlimiter.RateLimiter
	uncompressedLimiter extensionlimiter.RateLimiter
	isClient            bool
}

func (h *limiterStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	// Create a new context with limiter tracking (includes rate limiter error state)
	ctx, tracking := extensionlimiter.EnsureLimiterTracking(ctx)
	if tracking.HasBeenLimited(extensionlimiter.WeightKeyNetworkBytes) {

	}
	return ctx
}

func (h *limiterStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	// Check for payload messages to apply network byte rate limiting
	var wireBytes int
	var reqBytes int
	switch payload := s.(type) {
	case *stats.InPayload:
		// Server receiving payload (or client receiving response)
		if !h.isClient {
			wireBytes = payload.WireLength
			reqBytes = payload.Length
		}
	case *stats.OutPayload:
		// Client sending payload (or server sending response)
		if h.isClient {
			wireBytes = payload.WireLength
			reqBytes = payload.Length
		}
	default:
		// Not a payload message, no rate limiting to apply
		return
	}

	// Ensure limiter tracking exists in context and get tracking state
	tracking := extensionlimiter.GetLimiterTracking(ctx)
	if tracking == nil {
		// Misconfiguration
		return
	}


	// Implement 1 or 2 rate limits in parallel
	var err1, err2 error
	var res1, res2 extensionlimiter.RateReservation

	if wireBytes != 0 && h.compressedLimiter != nil &&
		!tracking.HasBeenLimited(extensionlimiter.WeightKeyNetworkBytes) {
		res1, err1 = h.compressedLimiter.ReserveRate(ctx, wireBytes)
		if err1 == nil {
			tracking.AddRequest(extensionlimiter.WeightKeyNetworkBytes, wireBytes)
		}
	}
	if reqBytes != 0 && h.uncompressedLimiter != nil &&
		!tracking.HasBeenLimited(extensionlimiter.WeightKeyRequestBytes) {
		res2, err2 = h.uncompressedLimiter.ReserveRate(ctx, reqBytes)
		if err2 == nil {
			tracking.AddRequest(extensionlimiter.WeightKeyRequestBytes, reqBytes)
		}
	}

	var wait1, wait2 time.Duration

	if res1 != nil {
		wait1 = res1.WaitTime()
		defer res1.Cancel()
	}
	if res2 != nil {
		wait2 = res2.WaitTime()
		defer res2.Cancel()
	}

	if err := multierr.Append(err1, err2); err != nil {
		setRateLimiterError(ctx, err)
		return
	}

	wait := max(wait1, wait2)
	if wait == 0 {
		return
	}
	select {
	case <-ctx.Done():
	case <-time.After(wait):
	}
}

func (h *limiterStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *limiterStatsHandler) HandleConn(ctx context.Context, _ stats.ConnStats) {
}

type serverStream struct {
	grpc.ServerStream
	limiter limiterhelper.Wrapper
}

// RecvMsg applies rate limiting to server stream message receiving.
func (s *serverStream) RecvMsg(m any) error {
	return s.limiter.LimitCall(
		s.Context(), 1,
		func(ctx context.Context) error {
			if err := checkRateLimiterError(ctx); err != nil {
				return err
			}
			return s.ServerStream.RecvMsg(m)
		})
}

// wrapServerStream wraps a gRPC server stream with rate limiting.
func wrapServerStream(ss grpc.ServerStream, _ *grpc.StreamServerInfo, limiter limiterhelper.Wrapper) grpc.ServerStream {
	return &serverStream{
		ServerStream: ss,
		limiter:      limiter,
	}
}

type clientStream struct {
	grpc.ClientStream
	limiter limiterhelper.Wrapper
}

// SendMsg applies rate limiting to client stream message sending.
func (s *clientStream) SendMsg(m any) error {
	return s.limiter.LimitCall(
		s.Context(), 1,
		func(ctx context.Context) error {
			if err := checkRateLimiterError(ctx); err != nil {
				return err
			}
			return s.ClientStream.SendMsg(m)
		})
}

// wrapClientStream wraps a gRPC client stream with rate limiting.
func wrapClientStream(cs grpc.ClientStream, _ string, limiter limiterhelper.Wrapper) grpc.ClientStream {
	return &clientStream{
		ClientStream: cs,
		limiter:      limiter,
	}
}
