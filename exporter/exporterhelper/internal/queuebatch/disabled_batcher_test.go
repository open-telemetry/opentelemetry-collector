// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

func TestDisabledBatcher(t *testing.T) {
	tests := []struct {
		name       string
		maxWorkers int
	}{
		{
			name:       "one_worker",
			maxWorkers: 1,
		},
		{
			name:       "three_workers",
			maxWorkers: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := requesttest.NewSink()
			var exportCalls atomic.Int64
			var attemptedItemCount atomic.Int64
			exportFunc := func(ctx context.Context, req request.Request) error {
				err := sink.Export(ctx, req)
				attemptedItemCount.Add(int64(req.ItemsCount()))
				exportCalls.Add(1)
				return err
			}
			ba := newDisabledBatcher(exportFunc)

			q, err := queue.NewQueue(queue.Settings[request.Request]{
				Capacity:        1000,
				BlockOnOverflow: true,
				NumConsumers:    tt.maxWorkers,
				Telemetry:       componenttest.NewNopTelemetrySettings(),
			}, ba.Consume)
			require.NoError(t, err)

			require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, q.Shutdown(context.Background()))
				require.NoError(t, ba.Shutdown(context.Background()))
			})

			var offers int
			var offeredItemCount int
			offerItems := func(items int) {
				offers++
				offeredItemCount += items
				require.NoError(t, q.Offer(context.Background(), &requesttest.FakeRequest{Items: items}))
			}

			// Create batches of 8 items per num_consumers, and arrange for one to fail.
			sink.SetExportErr(errors.New("transient error"))
			for range tt.maxWorkers {
				offerItems(8)
			}
			offerItems(17)
			offerItems(13)
			offerItems(35)
			offerItems(2)

			assert.EventuallyWithT(t, func(t *assert.CollectT) {
				assert.Equal(t, int64(offers), exportCalls.Load())
				assert.Equal(t, int64(offeredItemCount), attemptedItemCount.Load())
			}, time.Second, 10*time.Millisecond)

			// One of the 8-item requests failed,
			assert.Equal(t, offers-1, sink.RequestsCount())
			assert.Equal(t, offeredItemCount-8, sink.ItemsCount())
		})
	}
}
