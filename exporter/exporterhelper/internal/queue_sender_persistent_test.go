// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

// Requests that are in flight when the collector shuts down must stay in the persistent queue if the
// destination is unavailable, even when the exporter doesn't enable retry_on_failure.
func TestPersistentQueueRetainsInFlightRequestsOnShutdown(t *testing.T) {
	tests := []struct {
		name string
		// error returned by the exporter while the destination is unavailable.
		exportErr error
		// whether retry_on_failure is enabled.
		retryEnabled bool
		// number of items expected to be redelivered after the restart.
		wantRedelivered int64
	}{
		{
			name:            "retryable_error_retained",
			exportErr:       errors.New("connection refused"),
			wantRedelivered: numPersistentTestRequests,
		},
		{
			name:            "retryable_error_retained_with_retry_on_failure",
			exportErr:       errors.New("connection refused"),
			retryEnabled:    true,
			wantRedelivered: numPersistentTestRequests,
		},
		{
			// Permanent errors always fail, so retaining them would keep them in the storage forever.
			name:            "permanent_error_dropped",
			exportErr:       consumererror.NewPermanent(errors.New("invalid data")),
			wantRedelivered: numPersistentTestRequests - persistentTestNumConsumers,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := storagetest.NewMockStorageExtension(nil)
			host := hosttest.NewHost(map[component.ID]component.Component{{}: ext})

			// Fill the queue while the destination is unavailable and hold every consumer inside the
			// export call, so all of them are in flight when the shutdown starts.
			var inFlight atomic.Int64
			unblock := make(chan struct{})
			var unblockOnce sync.Once
			failing := newPersistentTestExporter(t, tt.retryEnabled, func(context.Context, request.Request) error {
				inFlight.Add(1)
				<-unblock
				return tt.exportErr
			})
			require.NoError(t, failing.Start(context.Background(), host))

			for range numPersistentTestRequests {
				require.NoError(t, failing.Send(context.Background(), &requesttest.FakeRequest{Items: 1}))
			}
			require.Eventually(t, func() bool { return inFlight.Load() == persistentTestNumConsumers },
				time.Second, 10*time.Millisecond)

			// Release the consumers shortly after the shutdown started, so they all fail during the
			// shutdown. Releasing them late is harmless because Shutdown blocks until they return.
			go func() {
				time.Sleep(100 * time.Millisecond)
				unblockOnce.Do(func() { close(unblock) })
			}()
			require.NoError(t, failing.Shutdown(context.Background()))

			// Restart against the same storage with a healthy destination.
			var redelivered atomic.Int64
			healthy := newPersistentTestExporter(t, tt.retryEnabled, func(_ context.Context, req request.Request) error {
				redelivered.Add(int64(req.ItemsCount()))
				return nil
			})
			require.NoError(t, healthy.Start(context.Background(), host))
			t.Cleanup(func() { require.NoError(t, healthy.Shutdown(context.Background())) })

			assert.Eventually(t, func() bool { return redelivered.Load() == tt.wantRedelivered },
				time.Second, 10*time.Millisecond, "redelivered %d items, want %d", redelivered.Load(), tt.wantRedelivered)
		})
	}
}

const (
	numPersistentTestRequests  = 100
	persistentTestNumConsumers = 10
)

func newPersistentTestExporter(tb testing.TB, retryEnabled bool, pusher func(context.Context, request.Request) error) *BaseExporter {
	storageID := component.ID{}
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = persistentTestNumConsumers
	qCfg.StorageID = &storageID
	// Disable batching to keep the number of in-flight requests predictable.
	qCfg.Batch = configoptional.None[queuebatch.BatchConfig]()

	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = retryEnabled

	be, err := NewBaseExporter(
		exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalLogs, pusher,
		WithRetry(rCfg),
		WithQueueBatchSettings(queuebatch.Settings[request.Request]{Encoding: itemsCountEncoding{}}),
		WithQueue(configoptional.Some(qCfg)),
	)
	require.NoError(tb, err)
	return be
}

// itemsCountEncoding round-trips the number of items of a request, which is all the tests above need.
type itemsCountEncoding struct{}

func (itemsCountEncoding) Marshal(_ context.Context, req request.Request) ([]byte, error) {
	return []byte{byte(req.ItemsCount())}, nil
}

func (itemsCountEncoding) Unmarshal(buf []byte) (context.Context, request.Request, error) {
	if len(buf) != 1 {
		return context.Background(), nil, errors.New("invalid encoded request")
	}
	return context.Background(), &requesttest.FakeRequest{Items: int(buf[0])}, nil
}
