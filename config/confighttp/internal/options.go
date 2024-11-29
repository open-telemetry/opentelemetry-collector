package internal // import "go.opentelemetry.io/collector/config/confighttp/internal"

import (
	"io"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// toServerOptions has options that change the behavior of the HTTP server
// returned by ServerConfig.ToServer().
type ToServerOptions struct {
	ErrHandler   func(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int)
	Decoders     map[string]func(body io.ReadCloser) (io.ReadCloser, error)
	OtelhttpOpts []otelhttp.Option
}
