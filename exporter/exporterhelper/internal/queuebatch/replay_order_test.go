// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
)

type replayTestEncoding struct{}

func (replayTestEncoding) Marshal(_ context.Context, req request.Request) ([]byte, error) {
	return []byte(strconv.Itoa(req.ItemsCount())), nil
}

func (replayTestEncoding) Unmarshal(body []byte) (context.Context, request.Request, error) {
	items, err := strconv.Atoi(string(body))
	return context.Background(), &requesttest.FakeRequest{Items: items}, err
}

func TestQueueBatchReplayOrderIsOptIn(t *testing.T) {
	for _, ordered := range []bool{false, true} {
		t.Run(strconv.FormatBool(ordered), func(t *testing.T) {
			ctx := context.Background()
			storageID := component.MustNewID("storage")
			host := hosttest.NewHost(map[component.ID]component.Component{storageID: storagetest.NewMockStorageExtension(nil)})
			cfg := newTestConfig()
			cfg.NumConsumers = 1
			cfg.Batch = configoptional.None[BatchConfig]()
			cfg.StorageID = &storageID
			set := newFakeRequestSettings()
			set.Encoding = replayTestEncoding{}
			set.ReplayInOrder = ordered
			dispatched := make(chan int, 2)
			receive := func() int {
				t.Helper()
				select {
				case value := <-dispatched:
					return value
				case <-time.After(time.Second):
					t.Fatal("timed out waiting for replay")
					return 0
				}
			}
			unblock := make(chan struct{})
			original, err := NewQueueBatch(set, cfg, func(_ context.Context, req request.Request) error {
				dispatched <- req.ItemsCount()
				<-unblock
				return experr.NewShutdownErr(context.Canceled)
			})
			require.NoError(t, err)
			require.NoError(t, original.Start(ctx, host))
			t.Cleanup(func() {
				close(unblock)
				require.NoError(t, original.Shutdown(ctx))
			})
			require.NoError(t, original.Send(ctx, &requesttest.FakeRequest{Items: 1}))
			require.Equal(t, 1, receive())
			require.NoError(t, original.Send(ctx, &requesttest.FakeRequest{Items: 2}))

			// The mock storage permits reopening the same durable state while
			// the original consumer remains blocked, simulating a crash.
			restarted, err := NewQueueBatch(set, cfg, func(_ context.Context, req request.Request) error {
				dispatched <- req.ItemsCount()
				return nil
			})
			require.NoError(t, err)
			require.NoError(t, restarted.Start(ctx, host))
			t.Cleanup(func() { require.NoError(t, restarted.Shutdown(ctx)) })
			want := []int{2, 1}
			if ordered {
				want = []int{1, 2}
			}
			require.Equal(t, want, []int{receive(), receive()})
		})
	}
}
