// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest // import "go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"

import (
	"errors"
	"net/http"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/extension/extensionauth"
)

var errMockError = errors.New("mock Error")

// NewErrorClient returns a new extensionauth.Client that always returns an error on any of the client methods.
func NewErrorClient(opts ...extensionauth.ClientOption) (extensionauth.Client, error) {
	errorOpts := []extensionauth.ClientOption{
		extensionauth.WithClientRoundTripper(func(base http.RoundTripper) (http.RoundTripper, error) {
			return nil, errMockError
		}),
		extensionauth.WithClientPerRPCCredentials(func() (credentials.PerRPCCredentials, error) {
			return nil, errMockError
		}),
	}
	return extensionauth.NewClient(append(errorOpts, opts...)...)
}
