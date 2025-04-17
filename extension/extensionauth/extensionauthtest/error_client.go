// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest // import "go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"

import (
	"net/http"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

var (
	_ extension.Extension      = (*errorClient)(nil)
	_ extensionauth.HTTPClient = (*errorClient)(nil)
	_ extensionauth.GRPCClient = (*errorClient)(nil)
)

type errorClient struct {
	component.StartFunc
	component.ShutdownFunc
	extensionauth.ClientPerRPCCredentialsFunc
	extensionauth.ClientRoundTripperFunc
}

// NewErrorClient returns a new [extension.Extension] that implements the [extensionauth.HTTPClient] and [extensionauth.GRPCClient] and always returns an error on both methods.
func NewErrorClient(err error) extension.Extension {
	return &errorClient{
		ClientRoundTripperFunc: func(http.RoundTripper) (http.RoundTripper, error) {
			return nil, err
		},
		ClientPerRPCCredentialsFunc: func() (credentials.PerRPCCredentials, error) {
			return nil, err
		},
	}
}
