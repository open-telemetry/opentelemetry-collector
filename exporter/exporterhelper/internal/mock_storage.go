// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

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
	return &mockStorageClient{st: &m.st, closed: &atomic.Bool{}}, nil
}

func NewMockStorageExtension(getClientError error) storage.Extension {
	return &mockStorageExtension{getClientError: getClientError}
}

type mockStorageClient struct {
	st     *sync.Map
	closed *atomic.Bool
}

func (m *mockStorageClient) Get(ctx context.Context, s string) ([]byte, error) {
	getOp := storage.GetOperation(s)
	err := m.Batch(ctx, getOp)
	return getOp.Value, err
}

func (m *mockStorageClient) Set(ctx context.Context, s string, bytes []byte) error {
	return m.Batch(ctx, storage.SetOperation(s, bytes))
}

func (m *mockStorageClient) Delete(ctx context.Context, s string) error {
	return m.Batch(ctx, storage.DeleteOperation(s))
}

func (m *mockStorageClient) Close(_ context.Context) error {
	m.closed.Store(true)
	return nil
}

func (m *mockStorageClient) Batch(_ context.Context, ops ...storage.Operation) error {
	if m.isClosed() {
		panic("client already closed")
	}
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

func (m *mockStorageClient) isClosed() bool {
	return m.closed.Load()
}
