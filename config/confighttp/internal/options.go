// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/config/confighttp/internal"

import (
	"io"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/collector/component"
)

// ToServerOptions has options that change the behavior of the HTTP server
// returned by ServerConfig.ToServer().
type ToServerOptions struct {
	ErrHandler   func(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int)
	Decoders     map[string]func(body io.ReadCloser) (io.ReadCloser, error)
	OtelhttpOpts []otelhttp.Option
}

func (tso *ToServerOptions) Apply(opts ...ToServerOption) {
	for _, o := range opts {
		o.apply(tso)
	}
}

// ToServerOption is an option to change the behavior of the HTTP server
// returned by ServerConfig.ToServer().
type ToServerOption interface {
	apply(*ToServerOptions)
}

// ToServerOptionFunc converts a function into ToServerOption interface.
type ToServerOptionFunc func(*ToServerOptions)

func (of ToServerOptionFunc) apply(e *ToServerOptions) {
	of(e)
}

// ToListenerOptions has options that change the behavior of the HTTP listener
// returned by ServerConfig.ToListenerWithOptions().
type ToListenerOptions struct {
	Extensions map[component.ID]component.Component
}

func (tlo *ToListenerOptions) Apply(opts ...ToListenerOption) {
	for _, o := range opts {
		o.apply(tlo)
	}
}

// ToListenerOption is an option to change the behavior of the HTTP server
// returned by ServerConfig.ToServer().
type ToListenerOption interface {
	apply(*ToListenerOptions)
}

// ToListenerOptionFunc converts a function into ToListenerOption interface.
type ToListenerOptionFunc func(*ToListenerOptions)

func (of ToListenerOptionFunc) apply(e *ToListenerOptions) {
	of(e)
}
