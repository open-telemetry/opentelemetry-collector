// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfighttp // import "go.opentelemetry.io/collector/config/confighttp/xconfighttp"

import (
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confighttp/internal"
)

// WithOtelHTTPOptions allows providing (or overriding) options passed
// to the otelhttp.NewHandler() function.
//
// This is located in the experimental sub-package because the otelhttp library
// has not reached v1.x yet and exposing its types in confighttp public API
// could lead to breaking changes in the future.
// See https://github.com/open-telemetry/opentelemetry-collector/pull/11769
func WithOtelHTTPOptions(httpopts ...otelhttp.Option) confighttp.ToServerOption {
	return internal.ToServerOptionFunc(func(opts *internal.ToServerOptions) {
		opts.OtelhttpOpts = append(opts.OtelhttpOpts, httpopts...)
	})
}
