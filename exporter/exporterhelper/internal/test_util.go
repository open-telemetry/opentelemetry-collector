package internal

import (
	"context"
	"sync"

	"errors"
	"go.opentelemetry.io/collector/extension/experimental/storage"
)

// NewMockStorageClient creates a new instance of MockStorageClient
func NewMockStorageClient() storage.Client {
	return &MockStorageClient{
		st: map[string][]byte{},
	}
}

// MockStorageClient is a mocked in memory storage client designed to be used in tests
type MockStorageClient struct {
	st           map[string][]byte
	mux          sync.Mutex
	closeCounter uint64
}

func (m *MockStorageClient) Get(_ context.Context, s string) ([]byte, error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	val, found := m.st[s]
	if !found {
		return nil, nil
	}

	return val, nil
}

func (m *MockStorageClient) Set(_ context.Context, s string, bytes []byte) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.st[s] = bytes
	return nil
}

func (m *MockStorageClient) Delete(_ context.Context, s string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	delete(m.st, s)
	return nil
}

func (m *MockStorageClient) Close(_ context.Context) error {
	m.closeCounter++
	return nil
}

func (m *MockStorageClient) Batch(_ context.Context, ops ...storage.Operation) error {
	m.mux.Lock()
	defer m.mux.Unlock()

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

func (m *MockStorageClient) getCloseCount() uint64 {
	return m.closeCounter
}
