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
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	errMetadataNotFound = errors.New("no request metadata found")
)

// Authenticator will authenticate the incoming request/RPC
type Authenticator interface {
	// Authenticate checks whether the given context contains valid auth data. Successfully authenticated calls will always return a nil error and a context with the auth data.
	Authenticate(context.Context, map[string][]string) (context.Context, error)

	// UnaryInterceptor is a helper method to provide a gRPC-compatible UnaryInterceptor, typically calling the authenticator's Authenticate method.
	UnaryInterceptor(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error)

	// StreamInterceptor is a helper method to provide a gRPC-compatible StreamInterceptor, typically calling the authenticator's Authenticate method.
	StreamInterceptor(interface{}, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) error

	// ToServerOptions is a helper method that builds a set of server options ready to be used by the gRPC server
	ToServerOptions() ([]grpc.ServerOption, error)
}

type AuthenticateFunc func(context.Context, map[string][]string) (context.Context, error)
type UnaryInterceptorFunc func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler, authenticate AuthenticateFunc) (interface{}, error)
type StreamInterceptorFunc func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler, authenticate AuthenticateFunc) error

func DefaultUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler, authenticate AuthenticateFunc) (interface{}, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMetadataNotFound
	}

	ctx, err := authenticate(ctx, headers)
	if err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

func DefaultStreamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler, authenticate AuthenticateFunc) error {
	ctx := stream.Context()
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errMetadataNotFound
	}

	// TODO: how to replace the context from the stream?
	_, err := authenticate(ctx, headers)
	if err != nil {
		return err
	}

	return handler(srv, stream)
}

var authRegistry = map[string]Authenticator{}

func AddAuthenticatorToRegistry(name string, auth Authenticator) error {
	if _, ok := authRegistry[name]; ok {
		return fmt.Errorf("Authenticator with name %s already registered", name)
	}
	authRegistry[name] = auth
	return nil
}

func GetAuthenticatorFromRegistry(name string) (Authenticator, error) {
	auth, ok := authRegistry[name]
	if !ok {
		return nil, fmt.Errorf("Authenticator not found with name %s", name)
	}
	return auth, nil
}
