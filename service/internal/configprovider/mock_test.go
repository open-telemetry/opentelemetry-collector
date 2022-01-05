// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configprovider

import (
	"context"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
)

// mockProvider is a mock implementation of Provider, useful for testing.
type mockProvider struct {
	retrieved   configmapprovider.Retrieved
	retrieveErr error
	shutdownErr error
}

var _ configmapprovider.Provider = &mockProvider{}

func (m *mockProvider) Retrieve(context.Context, func(*configmapprovider.ChangeEvent)) (configmapprovider.Retrieved, error) {
	if m.retrieveErr != nil {
		return nil, m.retrieveErr
	}
	if m.retrieved == nil {
		return configmapprovider.NewRetrieved(func(ctx context.Context) (*config.Map, error) { return config.NewMap(), nil })
	}
	return m.retrieved, nil
}

func (m *mockProvider) Shutdown(context.Context) error {
	return m.shutdownErr
}

func newErrGetRetrieved(getErr error) configmapprovider.Retrieved {
	ret, _ := configmapprovider.NewRetrieved(func(ctx context.Context) (*config.Map, error) { return nil, getErr })
	return ret
}

func newErrCloseRetrieved(closeErr error) configmapprovider.Retrieved {
	ret, _ := configmapprovider.NewRetrieved(
		func(ctx context.Context) (*config.Map, error) { return config.NewMap(), nil },
		configmapprovider.WithClose(func(ctx context.Context) error { return closeErr }))
	return ret
}
