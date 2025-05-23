package httplimiter

import (
	"context"
	"io"
	"net/http"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"
	"go.uber.org/multierr"
)

func NewClientLimiter(ext extensionlimiter.BaseLimiterProvider) (extensionmiddleware.HTTPClient, error) {
	wp, err1 := limiterhelper.MiddlewareToLimiterWrapperProvider(ext)
	rp, err2 := limiterhelper.MiddlewareToRateLimiterProvider(ext)
	if err := multierr.Append(err1, err2); err != nil {
		return nil, err
	}
	requestLimiter, err3 := wp.GetLimiterWrapper(extensionlimiter.WeightKeyRequestCount)
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

			var resp *http.Response
			err := requestLimiter.LimitCall(
				req.Context(),
				1,
				func(_ context.Context) error {
					if bytesLimiter != nil && req.Body != nil && req.Body != http.NoBody {
						// If bytes are limited, create a limited ReadCloser body.
						req.Body = &rateLimitedBody{
							body:    req.Body,
							limiter: limiterhelper.NewBlockingRateLimiter(bytesLimiter),
							ctx:     req.Context(),
						}
					}
					var err error
					resp, err = base.RoundTrip(req)
					return err
				})
			return resp, err
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

func NewServerLimiter(ext extensionlimiter.BaseLimiterProvider) (extensionmiddleware.HTTPServer, error) {
	wp, err1 := limiterhelper.MiddlewareToLimiterWrapperProvider(ext)
	rp, err2 := limiterhelper.MiddlewareToRateLimiterProvider(ext)
	if err := multierr.Append(err1, err2); err != nil {
		return nil, err
	}
	requestLimiter, err3 := wp.GetLimiterWrapper(extensionlimiter.WeightKeyRequestCount)
	bytesLimiter, err4 := rp.GetRateLimiter(extensionlimiter.WeightKeyNetworkBytes)
	if err := multierr.Append(err3, err4); err != nil {
		return nil, err
	}

	handler := func(base http.Handler) (http.Handler, error) {
		if requestLimiter == nil && bytesLimiter == nil {
			return base, nil
		}

		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			_ = requestLimiter.LimitCall(
				req.Context(),
				1,
				func(_ context.Context) error {
					if bytesLimiter != nil && req.Body != nil && req.Body != http.NoBody {
						// If bytes are limited, create a limited ReadCloser body.
						req.Body = &rateLimitedBody{
							body:    req.Body,
							limiter: limiterhelper.NewBlockingRateLimiter(bytesLimiter),
							ctx:     req.Context(),
						}
					}
					base.ServeHTTP(w, req)
					return nil
				})
		}), nil
	}
	return extensionmiddleware.GetHTTPHandlerFunc(handler), nil
}
