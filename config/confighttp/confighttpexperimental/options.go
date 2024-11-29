package confighttpexperimental // import "go.opentelemetry.io/collector/config/confighttp/confighttpexperimental"

import (
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confighttp/internal"
)

// ToServerOption is a functional option for ToServer.
type toServerOptions = internal.ToServerOptions

// WithOtelHTTPOptions allows providing (or overriding) options passed
// to the otelhttp.NewHandler() function.
func WithOtelHTTPOptions(httpopts ...otelhttp.Option) confighttp.ToServerOption {
	return nil
	// return func(opts *toServerOptions) {
	// 	opts.OtelhttpOpts = append(opts.OtelhttpOpts, httpopts...)
	// }
}
