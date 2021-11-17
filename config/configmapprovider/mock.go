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

package configmapprovider // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"context"

	"go.opentelemetry.io/collector/config"
)

// mockProvider is a mock implementation of Provider, useful for testing.
type mockProvider struct {
	retrieved   Retrieved
	retrieveErr error
}

var _ Provider = &mockProvider{}

func (m *mockProvider) Retrieve(ctx context.Context, onChange func(*ChangeEvent)) (Retrieved, error) {
	return m.retrieved, m.retrieveErr
}

func (mockProvider) Shutdown(ctx context.Context) error { return nil }

type mockRetrieved struct {
	got    *config.Map
	getErr error
}

var _ Retrieved = &mockRetrieved{}

func (sr *mockRetrieved) Get(ctx context.Context) (*config.Map, error) {
	return sr.got, sr.getErr
}

func (mockRetrieved) Close(ctx context.Context) error { return nil }
