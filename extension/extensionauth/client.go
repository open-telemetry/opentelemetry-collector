// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauth // import "go.opentelemetry.io/collector/extension/extensionauth"

import (
	"net/http"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// Client is an Extension that can be used as an authenticator for the configauth.Authentication option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the Authentication configuration.
type Client interface {
	extension.Extension

	// RoundTripper returns a RoundTripper that can be used to authenticate HTTP requests.
	RoundTripper(base http.RoundTripper) (http.RoundTripper, error)

	// PerRPCCredentials returns a PerRPCCredentials that can be used to authenticate gRPC requests.
	PerRPCCredentials() (credentials.PerRPCCredentials, error)
}

// ClientOption represents the possible options for NewClient.
type ClientOption interface {
	apply(*defaultClient)
}

type clientOptionFunc func(*defaultClient)

func (of clientOptionFunc) apply(e *defaultClient) {
	of(e)
}

// ClientRoundTripperFunc specifies the function that returns a RoundTripper that can be used to authenticate HTTP requests.
type ClientRoundTripperFunc func(base http.RoundTripper) (http.RoundTripper, error)

// ClientPerRPCCredentialsFunc specifies the function that returns a PerRPCCredentials that can be used to authenticate gRPC requests.
type ClientPerRPCCredentialsFunc func() (credentials.PerRPCCredentials, error)

var _ Client = (*defaultClient)(nil)

type defaultClient struct {
	component.StartFunc
	component.ShutdownFunc
	clientRoundTripperFunc      ClientRoundTripperFunc
	clientPerRPCCredentialsFunc ClientPerRPCCredentialsFunc
}

// PerRPCCredentials implements Client.
func (d *defaultClient) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	if d.clientPerRPCCredentialsFunc == nil {
		return nil, nil
	}
	return d.clientPerRPCCredentialsFunc()
}

// RoundTripper implements Client.
func (d *defaultClient) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	if d.clientRoundTripperFunc == nil {
		return base, nil
	}
	return d.clientRoundTripperFunc(base)
}

// WithClientStart overrides the default `Start` function for a component.Component.
// The default always returns nil.
func WithClientStart(startFunc component.StartFunc) ClientOption {
	return clientOptionFunc(func(o *defaultClient) {
		o.StartFunc = startFunc
	})
}

// WithClientShutdown overrides the default `Shutdown` function for a component.Component.
// The default always returns nil.
func WithClientShutdown(shutdownFunc component.ShutdownFunc) ClientOption {
	return clientOptionFunc(func(o *defaultClient) {
		o.ShutdownFunc = shutdownFunc
	})
}

// WithClientRoundTripper provides a `RoundTripper` function for this client authenticator.
// The default round tripper is no-op.
func WithClientRoundTripper(roundTripperFunc ClientRoundTripperFunc) ClientOption {
	return clientOptionFunc(func(o *defaultClient) {
		o.clientRoundTripperFunc = roundTripperFunc
	})
}

// WithClientPerRPCCredentials provides a `PerRPCCredentials` function for this client authenticator.
// There's no default.
func WithClientPerRPCCredentials(perRPCCredentialsFunc ClientPerRPCCredentialsFunc) ClientOption {
	return clientOptionFunc(func(o *defaultClient) {
		o.clientPerRPCCredentialsFunc = perRPCCredentialsFunc
	})
}

// NewClient returns a Client configured with the provided options.
func NewClient(options ...ClientOption) (Client, error) {
	bc := &defaultClient{}

	for _, op := range options {
		op.apply(bc)
	}

	return bc, nil
}
