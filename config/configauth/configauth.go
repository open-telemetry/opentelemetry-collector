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
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

var (
	errAuthenticatorNotFound = errors.New("authenticator not found")
	errNotHTTPClient         = errors.New("requested authenticator is not a HTTP client authenticator")
	errNotGRPCClient         = errors.New("requested authenticator is not a gRPC client authenticator")
	errNotServer             = errors.New("requested authenticator is not a server authenticator")
	errMetadataNotFound      = errors.New("no request metadata found")
)

// Config defines the auth settings for the receiver.
type Config struct {
	// AuthenticatorID specifies the name of the extension to use in order to authenticate the incoming data point.
	AuthenticatorID component.ID `mapstructure:"authenticator,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// Deprecated: [v0.123.0] use GetGRPCServerOptions or GetHTTPHandler.
func (a Config) GetServerAuthenticator(_ context.Context, extensions map[component.ID]component.Component) (extensionauth.Server, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if server, ok := ext.(extensionauth.Server); ok {
			return server, nil
		}
		return nil, errNotServer
	}

	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}

// Deprecated: [v0.123.0] use GetHTTPRoundTripper.
func (a Config) GetHTTPClientAuthenticator(_ context.Context, extensions map[component.ID]component.Component) (extensionauth.HTTPClient, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if client, ok := ext.(extensionauth.HTTPClient); ok {
			return client, nil
		}
		return nil, errNotHTTPClient
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}

// Deprecated: [v0.123.0] Use GetGRPCDialOptions.
func (a Config) GetGRPCClientAuthenticator(_ context.Context, extensions map[component.ID]component.Component) (extensionauth.GRPCClient, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if client, ok := ext.(extensionauth.GRPCClient); ok {
			return client, nil
		}
		return nil, errNotGRPCClient
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}

// GetGRPCServerOptions attempts to select the appropriate extensionauth.Server from the list of extensions,
// based on the requested extension name and return the grpc.ServerOption to be used with the grpc.Server.
// If an authenticator is not found, an error is returned.
func (a Config) GetGRPCServerOptions(_ context.Context, extensions map[component.ID]component.Component) ([]grpc.ServerOption, error) {
	ext, found := extensions[a.AuthenticatorID]
	if !found {
		return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
	}

	eauth, ok := ext.(extensionauth.Server)
	if !ok {
		return nil, errNotServer
	}

	uInterceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		return authServerUnaryInterceptor(ctx, req, info, handler, eauth)
	}
	sInterceptors := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return authServerStreamInterceptor(srv, ss, info, handler, eauth)
	}

	return []grpc.ServerOption{grpc.ChainUnaryInterceptor(uInterceptor), grpc.ChainStreamInterceptor(sInterceptors)}, nil
}

// GetHTTPHandler attempts to select the appropriate extensionauth.Server from the list of extensions,
// based on the requested extension name and return the http.Handler to be used with the http.Server.
// If an authenticator is not found, an error is returned.
func (a Config) GetHTTPHandler(_ context.Context, extensions map[component.ID]component.Component, next http.Handler, reqParams []string) (http.Handler, error) {
	ext, found := extensions[a.AuthenticatorID]
	if !found {
		return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
	}

	eauth, ok := ext.(extensionauth.Server)
	if !ok {
		return nil, errNotServer
	}

	return authInterceptor(next, eauth, reqParams), nil
}

// GetHTTPRoundTripper attempts to select the appropriate extensionauth.Client from the list of extensions,
// based on the component id of the extension and return the http.RoundTripper to be used with the http.Client.
// If an authenticator is not found, an error is returned. This should be only used by HTTP clients.
func (a Config) GetHTTPRoundTripper(_ context.Context, extensions map[component.ID]component.Component, base http.RoundTripper) (http.RoundTripper, error) {
	ext, found := extensions[a.AuthenticatorID]
	if !found {
		return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
	}

	// Currently only support `extensionauth.HTTPClient`.
	client, ok := ext.(extensionauth.HTTPClient)
	if !ok {
		return nil, errNotHTTPClient
	}

	return client.RoundTripper(base)
}

// GetGRPCDialOptions attempts to select the appropriate extensionauth.Client from the list of extensions,
// based on the component id of the extension and return the grpc.DialOptions to be used with grpc.ClientConn.
// If an authenticator is not found, an error is returned. This should be only used by gRPC clients.
func (a Config) GetGRPCDialOptions(_ context.Context, extensions map[component.ID]component.Component) ([]grpc.DialOption, error) {
	ext, found := extensions[a.AuthenticatorID]
	if !found {
		return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
	}

	// Currently only support `extensionauth.GRPCClient`.
	client, ok := ext.(extensionauth.GRPCClient)
	if !ok {
		return nil, errNotGRPCClient
	}

	perRPCCredentials, err := client.PerRPCCredentials()
	if err != nil {
		return nil, err
	}

	return []grpc.DialOption{grpc.WithPerRPCCredentials(perRPCCredentials)}, nil
}

func authServerUnaryInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler, eauth extensionauth.Server) (any, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMetadataNotFound
	}

	ctx, err := eauth.Authenticate(ctx, headers)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return handler(ctx, req)
}

func authServerStreamInterceptor(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler, eauth extensionauth.Server) error {
	ctx := stream.Context()
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errMetadataNotFound
	}

	ctx, err := eauth.Authenticate(ctx, headers)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	return handler(srv, wrapServerStream(ctx, stream))
}

func authInterceptor(next http.Handler, eauth extensionauth.Server, requestParams []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sources := r.Header
		query := r.URL.Query()
		for _, param := range requestParams {
			if val, ok := query[param]; ok {
				sources[param] = val
			}
		}
		ctx, err := eauth.Authenticate(r.Context(), sources)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// wrappedServerStream is a thin wrapper around grpc.ServerStream that allows modifying context.
type wrappedServerStream struct {
	grpc.ServerStream
	// wrappedContext is the wrapper's own Context. You can assign it.
	wrappedCtx context.Context
}

// Context returns the wrapper's wrappedContext, overwriting the nested grpc.ServerStream.Context()
func (w *wrappedServerStream) Context() context.Context {
	return w.wrappedCtx
}

// wrapServerStream returns a ServerStream with the new context.
func wrapServerStream(wrappedCtx context.Context, stream grpc.ServerStream) *wrappedServerStream {
	if existing, ok := stream.(*wrappedServerStream); ok {
		existing.wrappedCtx = wrappedCtx
		return existing
	}
	return &wrappedServerStream{ServerStream: stream, wrappedCtx: wrappedCtx}
}
