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

package authtest // import "go.opentelemetry.io/collector/extension/auth/authtest"

import (
	"context"
	"errors"
	"net/http"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/auth"
)

var (
	_            auth.Client = (*MockClient)(nil)
	errMockError             = errors.New("mock Error")
)

// MockClient provides a mock implementation of GRPCClient and HTTPClient interfaces
type MockClient struct {
	ResultRoundTripper      http.RoundTripper
	ResultPerRPCCredentials credentials.PerRPCCredentials
	MustError               bool
}

// Start for the MockClient does nothing
func (m *MockClient) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown for the MockClient does nothing
func (m *MockClient) Shutdown(_ context.Context) error {
	return nil
}

// RoundTripper for the MockClient either returns error if the mock authenticator is forced to or
// returns the supplied resultRoundTripper.
func (m *MockClient) RoundTripper(_ http.RoundTripper) (http.RoundTripper, error) {
	if m.MustError {
		return nil, errMockError
	}
	return m.ResultRoundTripper, nil
}

// PerRPCCredentials for the MockClient either returns error if the mock authenticator is forced to or
// returns the supplied resultPerRPCCredentials.
func (m *MockClient) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	if m.MustError {
		return nil, errMockError
	}
	return m.ResultPerRPCCredentials, nil
}
