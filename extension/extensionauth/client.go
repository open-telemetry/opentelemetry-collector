// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauth // import "go.opentelemetry.io/collector/extension/extensionauth"

import (
	"net/http"

	"google.golang.org/grpc/credentials"
)

// HTTPClient is an optional Extension interface that can be used as an HTTP authenticator for the configauth.Config option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the [configauth.Config] configuration.
type HTTPClient interface {
	// RoundTripper returns a RoundTripper that can be used to authenticate HTTP requests.
	RoundTripper(base http.RoundTripper) (http.RoundTripper, error)
}

// GRPCClient is an optional Extension interface that can be used as a gRPC authenticator for the configauth.Config option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the [configauth.Config] configuration.
type GRPCClient interface {
	// PerRPCCredentials returns a PerRPCCredentials that can be used to authenticate gRPC requests.
	PerRPCCredentials() (credentials.PerRPCCredentials, error)
}

var _ HTTPClient = (*ClientRoundTripperFunc)(nil)

// ClientRoundTripperFunc specifies the function that returns a RoundTripper that can be used to authenticate HTTP requests.
type ClientRoundTripperFunc func(base http.RoundTripper) (http.RoundTripper, error)

func (f ClientRoundTripperFunc) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	if f == nil {
		return base, nil
	}
	return f(base)
}

var _ GRPCClient = (*ClientPerRPCCredentialsFunc)(nil)

// ClientPerRPCCredentialsFunc specifies the function that returns a PerRPCCredentials that can be used to authenticate gRPC requests.
type ClientPerRPCCredentialsFunc func() (credentials.PerRPCCredentials, error)

func (f ClientPerRPCCredentialsFunc) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	if f == nil {
		return nil, nil
	}
	return f()
}
