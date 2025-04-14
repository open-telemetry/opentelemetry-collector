// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/config/confighttp/internal"

import (
	"io"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
