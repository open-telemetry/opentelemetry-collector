// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configauth

import (
	"context"

	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
)

var (
	_ ServerAuthenticator = (*MockAuthenticator)(nil)
	_ component.Extension = (*MockAuthenticator)(nil)
)

// MockAuthenticator provides a testing mock for code dealing with authentication.
type MockAuthenticator struct {
	// AuthenticateFunc to use during the authentication phase of this mock. Optional.
	AuthenticateFunc AuthenticateFunc
	// TODO: implement the other funcs
}

// Authenticate executes the mock's AuthenticateFunc, if provided, or just returns the given context unchanged.
func (m *MockAuthenticator) Authenticate(ctx context.Context, headers map[string][]string) (context.Context, error) {
	if m.AuthenticateFunc == nil {
		return context.Background(), nil
	}
	return m.AuthenticateFunc(ctx, headers)
}

// GRPCUnaryServerInterceptor isn't currently implemented and always returns nil.
func (m *MockAuthenticator) GRPCUnaryServerInterceptor(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error) {
	return nil, nil
}

// GRPCStreamServerInterceptor isn't currently implemented and always returns nil.
func (m *MockAuthenticator) GRPCStreamServerInterceptor(interface{}, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) error {
	return nil
}

// Start isn't currently implemented and always returns nil.
func (m *MockAuthenticator) Start(context.Context, component.Host) error {
	return nil
}

// Shutdown isn't currently implemented and always returns nil.
func (m *MockAuthenticator) Shutdown(ctx context.Context) error {
	return nil
}
