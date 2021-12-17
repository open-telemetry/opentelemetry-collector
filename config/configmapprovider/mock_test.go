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

package configmapprovider

import (
	"context"

	"go.opentelemetry.io/collector/config"
)

// mockProvider is a mock implementation of Provider, useful for testing.
type mockProvider struct {
	retrieved   Retrieved
	retrieveErr error
	shutdownErr error
}

var _ Provider = &mockProvider{}

func (m *mockProvider) Retrieve(context.Context, func(*ChangeEvent)) (Retrieved, error) {
	if m.retrieveErr != nil {
		return nil, m.retrieveErr
	}
	if m.retrieved == nil {
		return &mockRetrieved{}, nil
	}
	return m.retrieved, nil
}

func (m *mockProvider) Shutdown(context.Context) error {
	return m.shutdownErr
}

type mockRetrieved struct {
	cfg      *config.Map
	getErr   error
	closeErr error
}

var _ Retrieved = &mockRetrieved{}

func (sr *mockRetrieved) Get(context.Context) (*config.Map, error) {
	if sr.getErr != nil {
		return nil, sr.getErr
	}
	if sr.cfg == nil {
		return config.NewMap(), nil
	}
	return sr.cfg, nil
}

func (sr *mockRetrieved) Close(context.Context) error {
	return sr.closeErr
}
