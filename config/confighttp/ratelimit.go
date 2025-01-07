// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// RateLimit defines rate limiter settings for the receiver.
type RateLimit struct {
	// RateLimiterID specifies the name of the extension to use in order to rate limit the incoming data point.
	RateLimiterID component.ID `mapstructure:"rate_limiter"`
}

type rateLimiter interface {
	extension.Extension

	Take(context.Context, string, http.Header) error
}

// rateLimiter attempts to select the appropriate rateLimiter from the list of extensions,
// based on the component id of the extension. If a rateLimiter is not found, an error is returned.
func (rl RateLimit) rateLimiter(extensions map[component.ID]component.Component) (rateLimiter, error) {
	if ext, found := extensions[rl.RateLimiterID]; found {
		if limiter, ok := ext.(rateLimiter); ok {
			return limiter, nil
		}
		return nil, fmt.Errorf("extension %q is not a rate limit", rl.RateLimiterID)
	}
	return nil, fmt.Errorf("rate limit %q not found", rl.RateLimiterID)
}

// rateLimitInterceptor adds interceptor for rate limit check.
// It returns TooManyRequests(429) status code if rate limiter rejects the request.
func rateLimitInterceptor(next http.Handler, limiter rateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := limiter.Take(r.Context(), r.URL.Path, r.Header)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
