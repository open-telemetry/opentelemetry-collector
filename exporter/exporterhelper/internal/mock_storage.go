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
	st             sync.Map
	getClientError error
}

func (m *mockStorageExtension) GetClient(_ context.Context, _ component.Kind, _ component.ID, _ string) (storage.Client, error) {
	if m.getClientError != nil {
		return nil, m.getClientError
	}
	return &mockStorageClient{st: &m.st}, nil
}

func NewMockStorageExtension(getClientError error) storage.Extension {
	return &mockStorageExtension{getClientError: getClientError}
}

type mockStorageClient struct {
	st           *sync.Map
	closeCounter uint64
}

func (m *mockStorageClient) Get(_ context.Context, s string) ([]byte, error) {
	val, found := m.st.Load(s)
	if !found {
		return nil, nil
	}

	return val.([]byte), nil
}

func (m *mockStorageClient) Set(_ context.Context, s string, bytes []byte) error {
	m.st.Store(s, bytes)
	return nil
}

func (m *mockStorageClient) Delete(_ context.Context, s string) error {
	m.st.Delete(s)
	return nil
}

func (m *mockStorageClient) Close(_ context.Context) error {
	m.closeCounter++
	return nil
}

func (m *mockStorageClient) Batch(_ context.Context, ops ...storage.Operation) error {
	for _, op := range ops {
		switch op.Type {
		case storage.Get:
			val, found := m.st.Load(op.Key)
			if !found {
				break
			}
			op.Value = val.([]byte)
		case storage.Set:
			m.st.Store(op.Key, op.Value)
		case storage.Delete:
			m.st.Delete(op.Key)
		default:
			return errors.New("wrong operation type")
		}
	}

	return nil
}

func (m *mockStorageClient) getCloseCount() uint64 {
	return m.closeCounter
}
