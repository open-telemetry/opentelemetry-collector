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

package testutil // import "go.opentelemetry.io/collector/internal/testutil"

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/collector/extension/experimental/storage"
)

// NewInMemoryStorageClient creates a new instance of InMemoryStorageClient
func NewInMemoryStorageClient() storage.Client {
	return &InMemoryStorageClient{
		st: map[string][]byte{},
	}
}

// InMemoryStorageClient is a mocked in memory storage client designed to be used in tests
type InMemoryStorageClient struct {
	mu           sync.Mutex
	st           map[string][]byte
	closeCounter uint64
}

func (m *InMemoryStorageClient) Get(_ context.Context, s string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, found := m.st[s]
	if !found {
		return nil, nil
	}

	return val, nil
}

func (m *InMemoryStorageClient) Set(_ context.Context, s string, bytes []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.st[s] = bytes
	return nil
}

func (m *InMemoryStorageClient) Delete(_ context.Context, s string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.st, s)
	return nil
}

func (m *InMemoryStorageClient) Close(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closeCounter++
	return nil
}

func (m *InMemoryStorageClient) Batch(_ context.Context, ops ...storage.Operation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, op := range ops {
		switch op.Type {
		case storage.Get:
			op.Value = m.st[op.Key]
		case storage.Set:
			m.st[op.Key] = op.Value
		case storage.Delete:
			delete(m.st, op.Key)
		default:
			return errors.New("wrong operation type")
		}
	}

	return nil
}

// GetCloseCount returns the number of times Close was invoked
func (m *InMemoryStorageClient) GetCloseCount() uint64 {
	return m.closeCounter
}
