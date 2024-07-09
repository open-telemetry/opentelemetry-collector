// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
)

func TestValidateConcurrencySettings(t *testing.T) {
	cCfg := NewDefaultConcurrencySettings()
	assert.NoError(t, cCfg.Validate())

	cCfg.Concurrency = 0
	assert.EqualError(t, cCfg.Validate(), "number of concurrent senders must be positive")

	cCfg.Concurrency = -1
	assert.EqualError(t, cCfg.Validate(), "number of concurrent senders must be positive")
}

func TestConcurrencySender(t *testing.T) {
	batcherCfg := exporterbatcher.NewDefaultConfig()
	batcherCfg.MinSizeItems = 10
	batcherCfg.FlushTimeout = 100 * time.Millisecond

	cfg := ConcurrencySettings{
		Concurrency: 10,
	}

	tests := []struct {
		name          string
		batcherOption Option
		concurrencyOption Option
		numRequests int
		numItems int
		hasError bool
		errString string
	}{
		{
			name:          "success",
			batcherOption: WithBatcher(batcherCfg, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc)),
			concurrencyOption: WithConcurrency(cfg),
			numRequests: cfg.Concurrency + 1,
			numItems: 100,
		},
		{
			name:          "canceled_context",
			batcherOption: WithBatcher(batcherCfg, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc)),
			concurrencyOption: WithConcurrency(cfg),
			numRequests: cfg.Concurrency + 1,
			numItems: 100,
			hasError: true,
			errString: "context canceled",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			be, err := newBaseExporter(defaultSettings, defaultDataType, newBlockingSender, tt.batcherOption, tt.concurrencyOption)
			require.NotNil(t, be)
			require.NoError(t, err)
			sink := newFakeRequestSink()

			require.NoError(t, be.Start(ctx, componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(ctx))
			})

			var errs error
			var reqWG sync.WaitGroup
			for i := 0; i < tt.numRequests; i++ {
				reqWG.Add(1)
				go func() {
					errs = multierr.Append(errs, be.send(ctx, &fakeRequest{items: tt.numItems, sink: sink}))
					reqWG.Done()
				}()

			}

			assert.Eventually(t, func() bool {
				items := uint64(cfg.Concurrency) * uint64(100)
				return sink.requestsCount.Load() == uint64(cfg.Concurrency) && sink.itemsCount.Load() == items
			}, 100*time.Millisecond, 10*time.Millisecond)

			if tt.hasError {
				// senders blocking on acquiring the semaphore will see a context canceled error.
				cancel()
			}

			// unblock sender to allow blocked requests to acquire the semaphore.
			bs := be.obsrepSender.(*blockingSender)
			bs.unblock()
			reqWG.Wait()

			if tt.hasError {
				assert.ErrorContains(t, errs, tt.errString)
			} else {
				assert.NoError(t, errs)
			}
		})
	}
}

func TestConcurrencySender_RequestError(t *testing.T) {
	batcherCfg := exporterbatcher.NewDefaultConfig()
	batcherCfg.MinSizeItems = 10
	batcherCfg.FlushTimeout = 100 * time.Millisecond
	numRequests := 10
	errStr := "export test error"

	cfg := ConcurrencySettings{
		Concurrency: 10,
	}
	batcherOption := WithBatcher(batcherCfg, WithRequestBatchFuncs(fakeBatchMergeFunc, fakeBatchMergeSplitFunc))
	concurrencyOption := WithConcurrency(cfg)

	ctx := context.Background()
	be, err := newBaseExporter(defaultSettings, defaultDataType, newNoopObsrepSender, batcherOption, concurrencyOption)
	require.NotNil(t, be)
	require.NoError(t, err)
	sink := newFakeRequestSink()

	require.NoError(t, be.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(ctx))
	})

	var errs error
	var reqWG sync.WaitGroup
	req := &fakeRequest{items: 100, sink: sink, exportErr: errors.New(errStr)}
	for i := 0; i < numRequests; i++ {
		reqWG.Add(1)
		go func() {
			errs = multierr.Append(errs, be.send(ctx, req))
			reqWG.Done()
		}()

	}

	reqWG.Wait()

	assert.ErrorContains(t, errs, errStr)
}


type blockingSender struct {
	baseRequestSender
	blockCh        chan struct{} 
}

func newBlockingSender(*ObsReport) requestSender {
	return &blockingSender{
		blockCh:        make(chan struct{}), 
	}
}

func (bs *blockingSender) send(ctx context.Context, req ...Request) error {
	err := bs.nextSender.send(ctx, req...)
	<-bs.blockCh

	return err
}

func (bs *blockingSender) unblock() {
	close(bs.blockCh)
}