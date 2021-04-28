// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configauth

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/component"
)

var (
	errMetadataNotFound = errors.New("no request metadata found")
)

// Authenticator is an Extension that can be used as an authenticator for the configauth.Authentication option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the Authentication configuration. Each Authenticator is free to define its own behavior and configuration options,
// but note that the expectations that come as part of Extensions exist here as well. For instance, multiple instances of the same
// authenticator should be possible to exist under different names.
type Authenticator interface {
	component.Extension

	// Authenticate checks whether the given headers map contains valid auth data. Successfully authenticated calls will always return a nil error.
	// When the authentication fails, an error must be returned and the caller must not retry. This function is typically called from interceptors,
	// on behalf of receivers, but receivers can still call this directly if the usage of interceptors isn't suitable.
	// The deadline and cancellation given to this function must be respected, but note that authentication data has to be part of the map, not context.
	Authenticate(ctx context.Context, headers map[string][]string) error

	// UnaryInterceptor is a helper method to provide a gRPC-compatible UnaryInterceptor, typically calling the authenticator's Authenticate method.
	// Once the authentication succeeds, the interceptor is expected to call the handler.
	UnaryInterceptor(ctx context.Context, req interface{}, srvInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error)

	// StreamInterceptor is a helper method to provide a gRPC-compatible StreamInterceptor, typically calling the authenticator's Authenticate method.
	// Once the authentication succeeds, the interceptor is expected to call the handler.
	StreamInterceptor(srv interface{}, stream grpc.ServerStream, srvInfo *grpc.StreamServerInfo, handler grpc.StreamHandler) error
}

// AuthenticateFunc defines the signature for the function responsible for performing the authentication based on the given headers map.
// See Authenticator.Authenticate.
type AuthenticateFunc func(ctx context.Context, headers map[string][]string) error

// UnaryInterceptorFunc defines the signature for the function intercepting unary gRPC calls.
// See grpc.UnaryServerInterceptor and Authenticator.UnaryInterceptor.
type UnaryInterceptorFunc func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler, authenticate AuthenticateFunc) (interface{}, error)

// StreamInterceptorFunc defines the signature for the function intercepting streaming gRPC calls.
// See grpc.StreamServerInterceptor and Authenticator.StreamInterceptor.
type StreamInterceptorFunc func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler, authenticate AuthenticateFunc) error

// DefaultUnaryInterceptor provides a default implementation of a unary inteceptor, useful for most authenticators. It extracts the headers from the incoming
// request, under the assumption that the credentials will be part of the resulting map.
func DefaultUnaryInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler, authenticate AuthenticateFunc) (interface{}, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMetadataNotFound
	}

	if err := authenticate(ctx, headers); err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

// DefaultStreamInterceptor provides a default implementation of a stream interceptor, useful for most authenticators. It extracts the headers from the incoming
// request, under the assumption that the credentials will be part of the resulting map.
func DefaultStreamInterceptor(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler, authenticate AuthenticateFunc) error {
	ctx := stream.Context()
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errMetadataNotFound
	}

	if err := authenticate(ctx, headers); err != nil {
		return err
	}

	return handler(srv, stream)
}
