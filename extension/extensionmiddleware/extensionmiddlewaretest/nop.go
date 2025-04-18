// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddlewaretest // import "go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"

import (
	"net/http"

	"go.opentelemetry.io/collector/extension"
)

// NewNop returns a new [extension.Extension] that implements
// the all the extensionmiddleware interfaces.  For HTTP requests it
// returns the base RoundTripper and for gRPC requests it returns an
// empty slice of options.
func NewNop() extension.Extension {
	return &baseExtension{}
}

// HTTPClientFunc implements an HTTP client middleware function.
type HTTPClientFunc func(*http.Request) (*http.Response, error)

func (f HTTPClientFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
