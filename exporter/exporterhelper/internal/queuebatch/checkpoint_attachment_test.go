// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
)

type checkpointAwareRequest struct {
	deferredRequest
	store request.QueueCheckpointStore
}

func (r *checkpointAwareRequest) SetQueueCheckpointStore(store request.QueueCheckpointStore) {
	r.store = store
}

func TestAsyncQueueAttachesCheckpointStoreOnlyWithPersistence(t *testing.T) {
	for _, durable := range []bool{false, true} {
		name := "memory"
		if durable {
			name = "persistent"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			cfg := newTestConfig()
			cfg.Batch = configoptional.None[BatchConfig]()
			storageID := component.MustNewID("storage")
			if durable {
				cfg.StorageID = &storageID
			}
			r := &checkpointAwareRequest{deferredRequest: deferredRequest{FakeRequest: requesttest.FakeRequest{Items: 1}}}
			set := newFakeRequestSettings()
			set.Encoding = newFakeEncoding(r)
			attached := make(chan request.QueueCheckpointStore, 1)
			qb, err := NewAsyncQueueBatch(set, cfg, func(_ context.Context, req request.Request) error {
				attached <- req.(*checkpointAwareRequest).store
				req.(*checkpointAwareRequest).completion(nil)
				return nil
			})
			require.NoError(t, err)
			host := hosttest.NewHost(map[component.ID]component.Component{storageID: storagetest.NewMockStorageExtension(nil)})
			require.NoError(t, qb.Start(ctx, host))
			t.Cleanup(func() { require.NoError(t, qb.Shutdown(ctx)) })
			require.NoError(t, qb.Send(ctx, r))
			select {
			case store := <-attached:
				if durable {
					require.NotNil(t, store)
				} else {
					require.Nil(t, store, "memory ACKs must not serialize durable tail snapshots")
				}
			case <-time.After(time.Second):
				t.Fatal("request never reached the sender")
			}
		})
	}
}
