// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
)

type partitionKeyType struct{}

// TestQueueBatchConcurrentRequests drives real logs, metrics and traces requests through the
// queue and the batcher from many producer goroutines at once. Unlike the tests built on
// requesttest.FakeRequest, these request types lazily cache their item and byte sizes, so this
// is the only shape that exercises those caches under concurrency. Run it with -race.
func TestQueueBatchConcurrentRequests(t *testing.T) {
	const (
		producers       = 8
		requestsPerProd = 250
		itemsPerReq     = 5
		partitions      = 4
	)

	tests := []struct {
		name       string
		signal     pipeline.Signal
		settings   Settings[request.Request]
		newRequest func(int) request.Request
	}{
		{
			name:       "logs",
			signal:     pipeline.SignalLogs,
			settings:   NewLogsQueueBatchSettings(),
			newRequest: func(n int) request.Request { return newLogsRequest(testdata.GenerateLogs(n)) },
		},
		{
			name:       "metrics",
			signal:     pipeline.SignalMetrics,
			settings:   NewMetricsQueueBatchSettings(),
			newRequest: func(n int) request.Request { return newMetricsRequest(testdata.GenerateMetrics(n)) },
		},
		{
			name:       "traces",
			signal:     pipeline.SignalTraces,
			settings:   NewTracesQueueBatchSettings(),
			newRequest: func(n int) request.Request { return newTracesRequest(testdata.GenerateTraces(n)) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// testdata generators count metrics, not data points, so ask the request itself
			// how many items one payload holds instead of assuming it equals itemsPerReq.
			perRequest := tt.newRequest(itemsPerReq).ItemsCount()
			require.Positive(t, perRequest)

			var gotItems atomic.Int64

			set := AllSettings[request.Request]{
				Signal:    tt.signal,
				ID:        component.NewID(exportertest.NopType),
				Telemetry: componenttest.NewNopTelemetrySettings(),
				Settings:  tt.settings,
			}
			// A partitioner is required to get more than one consumer goroutine: NewQueueBatch
			// pins NumConsumers to 1 when batching is enabled without one. It also lets two
			// consumers land on the same partition, which is the only path where a single
			// request is merged into by two goroutines.
			set.Partitioner = NewPartitioner(func(ctx context.Context, _ request.Request) string {
				key, _ := ctx.Value(partitionKeyType{}).(string)
				return key
			})

			cfg := Config{
				Sizer:           request.SizerTypeItems,
				NumConsumers:    runtime.NumCPU(),
				QueueSize:       1_000_000,
				BlockOnOverflow: true,
				Batch: configoptional.Some(BatchConfig{
					FlushTimeout: 10 * time.Millisecond,
					Sizer:        request.SizerTypeItems,
					// Neither bound is a multiple of perRequest, so every partition hits both
					// mergeTo and split, the two places that mutate the payload and rewrite
					// the size caches.
					MinSize: int64(perRequest)*7 + 2,
					MaxSize: int64(perRequest)*20 + 1,
				}),
			}

			qb, err := NewQueueBatch(set, cfg, func(_ context.Context, req request.Request) error {
				// Read both cached dimensions from the consumer side.
				items := req.ItemsCount()
				assert.Positive(t, req.BytesSize())
				gotItems.Add(int64(items))
				return nil
			})
			require.NoError(t, err)
			require.NoError(t, qb.Start(context.Background(), componenttest.NewNopHost()))

			var wg sync.WaitGroup
			for p := 0; p < producers; p++ {
				wg.Go(func() {
					ctx := context.WithValue(context.Background(), partitionKeyType{}, strconv.Itoa(p%partitions))
					for i := 0; i < requestsPerProd; i++ {
						require.NoError(t, qb.Send(ctx, tt.newRequest(itemsPerReq)))
					}
				})
			}
			wg.Wait()

			// Shutdown flushes every partition's pending batch.
			require.NoError(t, qb.Shutdown(context.Background()))

			assert.Equal(t, int64(producers*requestsPerProd*perRequest), gotItems.Load(),
				"items were lost or double counted while merging and splitting concurrently")
		})
	}
}
