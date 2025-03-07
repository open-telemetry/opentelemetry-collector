// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package configauth implements the configuration settings to
// ensure authentication on incoming requests, and allows
// exporters to add authentication on outgoing requests.
package configauth // import "go.opentelemetry.io/collector/config/configauth"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

var (
	errAuthenticatorNotFound = errors.New("authenticator not found")
	errNotClient             = errors.New("requested authenticator is not a client authenticator")
	errNotHTTPClient         = errors.New("requested authenticator is not a HTTP client authenticator")
	errNotGRPCClient         = errors.New("requested authenticator is not a gRPC client authenticator")
	errNotServer             = errors.New("requested authenticator is not a server authenticator")
)

// Authentication defines the auth settings for the receiver.
type Authentication struct {
	// AuthenticatorID specifies the name of the extension to use in order to authenticate the incoming data point.
	AuthenticatorID component.ID `mapstructure:"authenticator,omitempty"`
}

// GetServerAuthenticator attempts to select the appropriate extensionauth.Server from the list of extensions,
// based on the requested extension name. If an authenticator is not found, an error is returned.
func (a Authentication) GetServerAuthenticator(_ context.Context, extensions map[component.ID]component.Component) (extensionauth.Server, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if server, ok := ext.(extensionauth.Server); ok {
			return server, nil
		}
		return nil, errNotServer
	}

	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}

// GetClientAuthenticator attempts to select the appropriate extensionauth.Client from the list of extensions,
// based on the component id of the extension. If an authenticator is not found, an error is returned.
// Deprecated: [v0.122.0] Use GetHTTPClientAuthenticator or GetGRPCClientAuthenticator instead.
func (a Authentication) GetClientAuthenticator(_ context.Context, extensions map[component.ID]component.Component) (extensionauth.Client, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if client, ok := ext.(extensionauth.Client); ok {
			return client, nil
		}
		return nil, errNotClient
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}

// GetHTTPClientAuthenticator attempts to select the appropriate extensionauth.Client from the list of extensions,
// based on the component id of the extension. If an authenticator is not found, an error is returned.
// This should be only used by HTTP clients.
func (a Authentication) GetHTTPClientAuthenticator(_ context.Context, extensions map[component.ID]component.Component) (extensionauth.HTTPClient, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if client, ok := ext.(extensionauth.HTTPClient); ok {
			return client, nil
		}
		return nil, errNotHTTPClient
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}

// GetGRPCClientAuthenticator attempts to select the appropriate extensionauth.Client from the list of extensions,
// based on the component id of the extension. If an authenticator is not found, an error is returned.
// This should be only used by gRPC clients.
func (a Authentication) GetGRPCClientAuthenticator(_ context.Context, extensions map[component.ID]component.Component) (extensionauth.GRPCClient, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if client, ok := ext.(extensionauth.GRPCClient); ok {
			return client, nil
		}
		return nil, errNotGRPCClient
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}
