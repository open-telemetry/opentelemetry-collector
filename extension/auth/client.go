// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package auth // import "go.opentelemetry.io/collector/extension/auth"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

// Client is an Extension that can be used as an authenticator for the configauth.Authentication option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the Authentication configuration.
// Deprecated: [v0.121.0] Use extensionauth.Client instead.
type Client = extensionauth.Client

// ClientOption represents the possible options for NewClient.
// Deprecated: [v0.121.0] Use extensionauth.Client instead.
type ClientOption = extensionauth.ClientOption

// ClientRoundTripperFunc specifies the function that returns a RoundTripper that can be used to authenticate HTTP requests.
// Deprecated: [v0.121.0] Use extensionauth.ClientRoundTripperFunc instead.
type ClientRoundTripperFunc = extensionauth.ClientRoundTripperFunc

// ClientPerRPCCredentialsFunc specifies the function that returns a PerRPCCredentials that can be used to authenticate gRPC requests.
// Deprecated: [v0.121.0] Use extensionauth.ClientPerRPCCredentialsFunc instead.
type ClientPerRPCCredentialsFunc = extensionauth.ClientPerRPCCredentialsFunc

// WithClientStart overrides the default `Start` function for a component.Component.
// The default always returns nil.
// Deprecated: [v0.121.0] Use extensionauth.WithClientStart instead.
func WithClientStart(startFunc component.StartFunc) ClientOption {
	return extensionauth.WithClientStart(startFunc)
}

// WithClientShutdown overrides the default `Shutdown` function for a component.Component.
// The default always returns nil.
// Deprecated: [v0.121.0] Use extensionauth.WithClientShutdown instead.
func WithClientShutdown(shutdownFunc component.ShutdownFunc) ClientOption {
	return extensionauth.WithClientShutdown(shutdownFunc)
}

// WithClientRoundTripper provides a `RoundTripper` function for this client authenticator.
// The default round tripper is no-op.
// Deprecated: [v0.121.0] Use extensionauth.WithClientRoundTripper instead.
func WithClientRoundTripper(roundTripperFunc ClientRoundTripperFunc) ClientOption {
	return extensionauth.WithClientRoundTripper(roundTripperFunc)
}

// WithClientPerRPCCredentials provides a `PerRPCCredentials` function for this client authenticator.
// There's no default.
// Deprecated: [v0.121.0] Use extensionauth.WithClientPerRPCCredentials instead.
func WithClientPerRPCCredentials(perRPCCredentialsFunc ClientPerRPCCredentialsFunc) ClientOption {
	return extensionauth.WithClientPerRPCCredentials(perRPCCredentialsFunc)
}

// NewClient returns a Client configured with the provided options.
// Deprecated: [v0.121.0] Use extensionauth.NewClient instead.
func NewClient(options ...ClientOption) Client {
	return extensionauth.NewClient(options...)
}
