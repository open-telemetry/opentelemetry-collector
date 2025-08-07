// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest // import "go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
	"go.opentelemetry.io/collector/extension/extensioncapabilities"
)

var (
	_ extension.Extension                              = (*nopClient)(nil)
	_ extensioncapabilities.HTTPClientAuthRoundTripper = (*nopClient)(nil)
	_ extensioncapabilities.GRPCClientAuthenticator    = (*nopClient)(nil)
)

type nopClient struct {
	component.StartFunc
	component.ShutdownFunc
	extensionauth.ClientRoundTripperFunc
	extensionauth.ClientPerRPCCredentialsFunc
}

// NewNopClient returns a new [extension.Extension] that implements the [extensioncapabilities.HTTPClientAuthRoundTripper] and [extensioncapabilities.GRPCClientAuthenticator].
// For HTTP requests it returns the base RoundTripper and for gRPC requests it returns a nil [credentials.PerRPCCredentials].
func NewNopClient() extension.Extension {
	return &nopClient{}
}
