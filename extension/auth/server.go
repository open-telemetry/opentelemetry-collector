// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package auth // import "go.opentelemetry.io/collector/extension/auth"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

// Server is an Extension that can be used as an authenticator for the configauth.Authentication option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the Authentication configuration. Each Server is free to define its own behavior and configuration options,
// but note that the expectations that come as part of Extensions exist here as well. For instance, multiple instances of the same
// authenticator should be possible to exist under different names.
// Deprecated: [v0.121.0] Use extensionauth.Server instead.
type Server = extensionauth.Server

// ServerOption represents the possible options for NewServer.
// Deprecated: [v0.121.0] Use extensionauth.ServerOption instead.
type ServerOption = extensionauth.ServerOption

// ServerAuthenticateFunc defines the signature for the function responsible for performing the authentication based
// on the given sources map. See Server.Authenticate.
// Deprecated: [v0.121.0] Use extensionauth.ServerAuthenticateFunc instead.
type ServerAuthenticateFunc = extensionauth.ServerAuthenticateFunc

// WithServerAuthenticate specifies which function to use to perform the authentication.
// Deprecated: [v0.121.0] Use extensionauth.WithServerAuthenticate instead.
func WithServerAuthenticate(authFunc ServerAuthenticateFunc) ServerOption {
	return extensionauth.WithServerAuthenticate(authFunc)
}

// WithServerStart overrides the default `Start` function for a component.Component.
// The default always returns nil.
// Deprecated: [v0.121.0] Use extensionauth.WithServerStart instead.
func WithServerStart(startFunc component.StartFunc) ServerOption {
	return extensionauth.WithServerStart(startFunc)
}

// WithServerShutdown overrides the default `Shutdown` function for a component.Component.
// The default always returns nil.
// Deprecated: [v0.121.0] Use extensionauth.WithServerShutdown instead.
func WithServerShutdown(shutdownFunc component.ShutdownFunc) ServerOption {
	return extensionauth.WithServerShutdown(shutdownFunc)
}

// NewServer returns a Server configured with the provided options.
// Deprecated: [v0.121.0] Use extensionauth.NewServer instead.
func NewServer(options ...ServerOption) Server {
	return extensionauth.NewServer(options...)
}
