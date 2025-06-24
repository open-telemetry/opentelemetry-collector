// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package httplimiter

import (
	"context"
	"io"
	"net/http"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"
)

// TODO: network bytes can be implemented after compression middleware, requires
// integration with confighttp.

func NewClientLimiter(ext extensionlimiter.AnyProvider) (extensionmiddleware.HTTPClient, error) {
	wp, err1 := limiterhelper.AnyToWrapperProvider(ext)
	rp, err2 := limiterhelper.AnyToRateLimiterProvider(ext)
	if err := multierr.Append(err1, err2); err != nil {
		return nil, err
	}
	requestLimiter, err3 := wp.GetWrapper(extensionlimiter.WeightKeyRequestCount)
	bytesLimiter, err4 := rp.GetRateLimiter(extensionlimiter.WeightKeyNetworkBytes)
	if err := multierr.Append(err3, err4); err != nil {
		return nil, err
	}

	roundtrip := func(base http.RoundTripper) (http.RoundTripper, error) {
		return extensionmiddlewaretest.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if requestLimiter == nil && bytesLimiter == nil {
				// If no limiters are configured, return the base round tripper
				return base.RoundTrip(req)
			}

			// Ensure limiter tracking exists in context
			ctx, tracking := extensionlimiter.EnsureLimiterTracking(req.Context())
			req = req.WithContext(ctx)

			// Determine what types of limiting to check/apply
			var shouldLimitRequests, shouldLimitBytes bool
			if requestLimiter != nil && !tracking.HasBeenLimited(extensionlimiter.WeightKeyRequestCount) {
				shouldLimitRequests = true
			}
			if bytesLimiter != nil && !tracking.HasBeenLimited(extensionlimiter.WeightKeyNetworkBytes) {
				shouldLimitBytes = true
			}

			if !shouldLimitRequests && !shouldLimitBytes {
				// Skip limiting since limiters were already applied for these weight keys
				return base.RoundTrip(req)
			}

			var resp *http.Response
			if shouldLimitRequests && requestLimiter != nil {
				err := requestLimiter.LimitCall(
					ctx,
					1,
					func(ctx context.Context) error {
						// Mark request limiting as applied
						tracking.AddRequest(extensionlimiter.WeightKeyRequestCount, 1)
						if shouldLimitBytes && bytesLimiter != nil && req.Body != nil && req.Body != http.NoBody {
							// If bytes are limited, create a limited ReadCloser body.
							req.Body = &rateLimitedBody{
								body:    req.Body,
								limiter: limiterhelper.NewBlockingRateLimiter(bytesLimiter),
								ctx:     ctx,
							}
							// Mark byte limiting as applied too
							tracking.AddRequest(extensionlimiter.WeightKeyNetworkBytes, 0)
						}
						var err error
						resp, err = base.RoundTrip(req.WithContext(ctx))
						return err
					})
				return resp, err
			} else if shouldLimitBytes && bytesLimiter != nil {
				// No request limiting, but apply byte limiting
				if req.Body != nil && req.Body != http.NoBody {
					req.Body = &rateLimitedBody{
						body:    req.Body,
						limiter: limiterhelper.NewBlockingRateLimiter(bytesLimiter),
						ctx:     ctx,
					}
				}
				// Mark byte limiting as applied
				tracking.AddRequest(extensionlimiter.WeightKeyNetworkBytes, 0)
				return base.RoundTrip(req)
			} else {
				// Neither type should be limited
				return base.RoundTrip(req)
			}
		}), multierr.Append(err1, err2)
	}
	return extensionmiddleware.GetHTTPRoundTripperFunc(roundtrip), nil
}

// rateLimitedBody wraps an http.Request.Body to track bytes and call the rate limiter
type rateLimitedBody struct {
	body    io.ReadCloser
	limiter limiterhelper.BlockingRateLimiter
	ctx     context.Context
}

var _ io.ReadCloser = &rateLimitedBody{}

// Read implements io.Reader interface, limiting bytes as they are read
func (rb *rateLimitedBody) Read(p []byte) (n int, err error) {
	n, err = rb.body.Read(p)
	if n > 0 {
		// Apply rate limiting based on network bytes after they are read
		limitErr := rb.limiter.WaitFor(rb.ctx, n)
		if limitErr != nil {
			// If the rate limiter rejects the bytes, return the error
			return n, limitErr // TODO: How to return HTTP 429?
		}
	}
	return n, err
}

// Close implements io.Closer interface
func (rb *rateLimitedBody) Close() error {
	return rb.body.Close()
}

func NewServerLimiter(ext extensionlimiter.AnyProvider) (extensionmiddleware.HTTPServer, error) {
	wp, err1 := limiterhelper.AnyToWrapperProvider(ext)
	rp, err2 := limiterhelper.AnyToRateLimiterProvider(ext)
	if err := multierr.Append(err1, err2); err != nil {
		return nil, err
	}
	requestLimiter, err3 := wp.GetWrapper(extensionlimiter.WeightKeyRequestCount)
	bytesLimiter, err4 := rp.GetRateLimiter(extensionlimiter.WeightKeyNetworkBytes)
	if err := multierr.Append(err3, err4); err != nil {
		return nil, err
	}

	handler := func(base http.Handler) (http.Handler, error) {
		if requestLimiter == nil && bytesLimiter == nil {
			return base, nil
		}

		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Ensure limiter tracking exists in context
			ctx, tracking := extensionlimiter.EnsureLimiterTracking(req.Context())
			req = req.WithContext(ctx)

			// Determine what types of limiting to check/apply
			var shouldLimitRequests, shouldLimitBytes bool
			if requestLimiter != nil && !tracking.HasBeenLimited(extensionlimiter.WeightKeyRequestCount) {
				shouldLimitRequests = true
			}
			if bytesLimiter != nil && !tracking.HasBeenLimited(extensionlimiter.WeightKeyNetworkBytes) {
				shouldLimitBytes = true
			}

			if !shouldLimitRequests && !shouldLimitBytes {
				// Skip limiting since limiters were already applied for these weight keys
				base.ServeHTTP(w, req)
				return
			}

			// Apply request count limiting if configured
			if shouldLimitRequests && requestLimiter != nil {
				err := requestLimiter.LimitCall(
					ctx,
					1,
					func(ctx context.Context) error {
						// Mark request limiting as applied
						tracking.AddRequest(extensionlimiter.WeightKeyRequestCount, 1)
						if shouldLimitBytes && bytesLimiter != nil && req.Body != nil && req.Body != http.NoBody {
							// If bytes are limited, create a limited ReadCloser body.
							req.Body = &rateLimitedBody{
								body:    req.Body,
								limiter: limiterhelper.NewBlockingRateLimiter(bytesLimiter),
								ctx:     ctx,
							}
							// Mark byte limiting as applied too
							tracking.AddRequest(extensionlimiter.WeightKeyNetworkBytes, 0)
						}
						base.ServeHTTP(w, req.WithContext(ctx))
						return nil
					})
				if err != nil {
					http.Error(w, "Request rate limited", http.StatusTooManyRequests)
					return
				}
			} else if shouldLimitBytes && bytesLimiter != nil {
				// No request limiting, but apply byte limiting
				if req.Body != nil && req.Body != http.NoBody {
					req.Body = &rateLimitedBody{
						body:    req.Body,
						limiter: limiterhelper.NewBlockingRateLimiter(bytesLimiter),
						ctx:     ctx,
					}
				}
				// Mark byte limiting as applied
				tracking.AddRequest(extensionlimiter.WeightKeyNetworkBytes, 0)
				base.ServeHTTP(w, req)
			} else {
				// Neither type should be limited
				base.ServeHTTP(w, req)
			}
		}), nil
	}
	return extensionmiddleware.GetHTTPHandlerFunc(handler), nil
}
