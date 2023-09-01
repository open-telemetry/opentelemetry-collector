// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/experimental/storage"
)

type mockStorageExtension struct {
	component.StartFunc
	component.ShutdownFunc
	getClientError error
}

func (m mockStorageExtension) GetClient(_ context.Context, _ component.Kind, _ component.ID, _ string) (storage.Client, error) {
	if m.getClientError != nil {
		return nil, m.getClientError
	}
	return &mockStorageClient{st: map[string][]byte{}}, nil
}

func NewMockStorageExtension(getClientError error) storage.Extension {
	return &mockStorageExtension{getClientError: getClientError}
}

type mockStorageClient struct {
	st           map[string][]byte
	mux          sync.Mutex
	closeCounter uint64
}

func (m *mockStorageClient) Get(_ context.Context, s string) ([]byte, error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	val, found := m.st[s]
	if !found {
		return nil, nil
	}

	return val, nil
}

func (m *mockStorageClient) Set(_ context.Context, s string, bytes []byte) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.st[s] = bytes
	return nil
}

func (m *mockStorageClient) Delete(_ context.Context, s string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	delete(m.st, s)
	return nil
}

func (m *mockStorageClient) Close(_ context.Context) error {
	m.closeCounter++
	return nil
}

func (m *mockStorageClient) Batch(_ context.Context, ops ...storage.Operation) error {
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

func (m *mockStorageClient) getCloseCount() uint64 {
	return m.closeCounter
}
