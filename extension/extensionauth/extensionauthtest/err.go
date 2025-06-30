// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest // import "go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"

import (
	"context"
	"net/http"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

var (
	_ extension.Extension      = (*errClient)(nil)
	_ extensionauth.HTTPClient = (*errClient)(nil)
	_ extensionauth.GRPCClient = (*errClient)(nil)
)

type errClient struct {
	component.StartFunc
	component.ShutdownFunc
	extensionauth.ClientPerRPCCredentialsFunc
	extensionauth.ClientRoundTripperFunc
	extensionauth.ServerAuthenticateFunc
}

// NewErr returns a new [extension.Extension] that implements all
// extensionauth interface and always returns an error.
func NewErr(err error) extension.Extension {
	return &errClient{
		ClientRoundTripperFunc: func(http.RoundTripper) (http.RoundTripper, error) {
			return nil, err
		},
		ClientPerRPCCredentialsFunc: func() (credentials.PerRPCCredentials, error) {
			return nil, err
		},
		ServerAuthenticateFunc: func(ctx context.Context, _ map[string][]string) (context.Context, error) { return ctx, err },
	}
}
