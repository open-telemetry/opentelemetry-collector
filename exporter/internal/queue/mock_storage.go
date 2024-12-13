// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/experimental/storage"
)

type mockStorageExtension struct {
	component.StartFunc
	component.ShutdownFunc
	st             sync.Map
	getClientError error
	executionDelay time.Duration
}

func (m *mockStorageExtension) GetClient(context.Context, component.Kind, component.ID, string) (storage.Client, error) {
	if m.getClientError != nil {
		return nil, m.getClientError
	}
	return &mockStorageClient{st: &m.st, closed: &atomic.Bool{}, executionDelay: m.executionDelay}, nil
}

func NewMockStorageExtension(getClientError error) storage.Extension {
	return NewMockStorageExtensionWithDelay(getClientError, 0)
}

func NewMockStorageExtensionWithDelay(getClientError error, executionDelay time.Duration) storage.Extension {
	return &mockStorageExtension{
		getClientError: getClientError,
		executionDelay: executionDelay,
	}
}

type mockStorageClient struct {
	st             *sync.Map
	closed         *atomic.Bool
	executionDelay time.Duration // simulate real storage client delay
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

func (m *mockStorageClient) Close(context.Context) error {
	m.closed.Store(true)
	return nil
}

func (m *mockStorageClient) Batch(_ context.Context, ops ...storage.Operation) error {
	if m.isClosed() {
		panic("client already closed")
	}
	if m.executionDelay != 0 {
		time.Sleep(m.executionDelay)
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
