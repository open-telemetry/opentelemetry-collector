// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
)

// MergeBatcherConfig defines a basic configuration for batching requests based on a timeout and a minimum number of
// items.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type MergeBatcherConfig struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`

	// Timeout sets the time after which a batch will be sent regardless of its size.
	// When this is set to zero, data will be sent immediately.
	// This is a recommended option, as it will ensure that the data is sent in a timely manner.
	Timeout time.Duration `mapstructure:"timeout"` // Is there a better name to avoid confusion with the consumerSender timeout?

	// MinSizeItems is the number of items (spans, data points or log records for OTLP) at which the batch should be
	// sent regardless of the timeout. There is no guarantee that the batch size always greater than this value.
	// This option requires the Request to implement RequestItemsCounter interface. Otherwise, it will be ignored.
	MinSizeItems int `mapstructure:"min_size_items"`
}

func (c MergeBatcherConfig) Validate() error {
	if c.MinSizeItems < 0 {
		return errors.New("min_size_items must be greater than or equal to zero")
	}
	if c.Timeout <= 0 {
		return errors.New("timeout must be greater than zero")
	}
	return nil
}

func NewDefaultMergeBatcherConfig() MergeBatcherConfig {
	return MergeBatcherConfig{
		Enabled:      true,
		Timeout:      200 * time.Millisecond,
		MinSizeItems: 8192,
	}
}

// SplitBatcherConfig defines batching configuration for merging or splitting requests based on a timeout and
// minimum and maximum number of items.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type SplitBatcherConfig struct {
	// MaxSizeItems is the maximum number of the batch items, i.e. spans, data points or log records for OTLP,
	// but can be anything else for other formats. If the batch size exceeds this value,
	// it will be broken up into smaller batches if possible.
	// Setting this value to zero disables the maximum size limit.
	MaxSizeItems int `mapstructure:"max_size_items"`
}

func (c SplitBatcherConfig) Validate() error {
	if c.MaxSizeItems < 0 {
		return errors.New("max_size_items must be greater than or equal to zero")
	}
	return nil
}

// MergeSplitBatcherConfig defines batching configuration for merging or splitting requests based on a timeout and
// minimum and maximum number of items.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type MergeSplitBatcherConfig struct {
	MergeBatcherConfig `mapstructure:",squash"`
	SplitBatcherConfig `mapstructure:",squash"`
}

func (c MergeSplitBatcherConfig) Validate() error {
	if c.MaxSizeItems < c.MinSizeItems {
		return errors.New("max_size_items must be greater than or equal to min_size_items")
	}
	return nil
}

// BatchConfigBatchersLimit defines batching configuration part for setting a maximum number of batchers.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type BatchConfigBatchersLimit struct {
	// BatchersLimit is the maximum number of batchers that can be used for batching.
	// Requests producing batch identifiers that exceed this limit will be dropped.
	// If this value is zero, then there is no limit on the number of batchers.
	BatchersLimit int `mapstructure:"batchers_limit"`
}

// BatchMergeFunc is a function that merges two requests into a single request.
// Context will be propagated from the first request. If you want to separate batches based on the context,
// use WithRequestBatchIdentifier option. Context will be propagated from the first request.
// Do not mutate the requests passed to the function if error can be returned after mutation.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type BatchMergeFunc func(context.Context, Request, Request) (Request, error)

// BatchMergeSplitFunc is a function that merge and/or splits a request into multiple requests based on the provided
// limit of maximum number of items. All the returned requests MUST have a number of items that does not exceed the
// maximum number of items. Size of the last returned request MUST be less or equal than the size of any other returned
// request. The original request MUST not be mutated if error is returned. The length of the returned slice MUST not
// be 0. The optionalReq argument can be nil, make sure to check it before using. maxItems argument is guaranteed to be
// greater than 0. Context will be propagated from the original request.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type BatchMergeSplitFunc func(ctx context.Context, optionalReq Request, req Request, maxItems int) ([]Request, error)

type BatcherOption func(*batchSender)

func WithSplitBatcher(cfg SplitBatcherConfig, msf BatchMergeSplitFunc) BatcherOption {
	return func(b *batchSender) {
		if cfg.MaxSizeItems != 0 {
			b.splitCfg = cfg
			b.mergeSplitFunc = msf
		}
	}
}

// batchSender is a component that accepts places requests into batches before passing them to the downstream senders.
//
// batch_processor implements consumer.Traces and consumer.Metrics
//
// Batches are sent out with any of the following conditions:
// - batch size reaches cfg.SendBatchSize
// - cfg.Timeout is elapsed since the timestamp when the previous batch was sent out.
type batchSender struct {
	baseRequestSender
	mergeCfg       MergeBatcherConfig
	splitCfg       SplitBatcherConfig
	mergeFunc      BatchMergeFunc
	mergeSplitFunc BatchMergeSplitFunc

	// concurrencyLimit is the maximum number of goroutines that can be created by the batcher.
	// If this number is reached and all the goroutines are busy, the batch will be sent right away.
	// Populated from the number of queue consumers if queue is enabled.
	concurrencyLimit uint64
	activeRequests   atomic.Uint64

	resetTimerCh chan struct{}

	mu          sync.Mutex
	activeBatch *batch

	logger *zap.Logger

	shutdownCh chan struct{}
	stopped    *atomic.Bool
}

// newBatchSender returns a new batch consumer component.
func newBatchSender(cfg MergeBatcherConfig, set exporter.CreateSettings, mf BatchMergeFunc, opts ...BatcherOption) requestSender {
	bs := &batchSender{
		activeBatch:  newEmptyBatch(),
		mergeCfg:     cfg,
		mergeFunc:    mf,
		logger:       set.Logger,
		shutdownCh:   make(chan struct{}),
		stopped:      &atomic.Bool{},
		resetTimerCh: make(chan struct{}),
	}

	for _, op := range opts {
		op(bs)
	}
	return bs
}

func (bs *batchSender) Start(_ context.Context, _ component.Host) error {
	timer := time.NewTimer(bs.mergeCfg.Timeout)
	go func() {
		for {
			select {
			case <-bs.shutdownCh:
				bs.mu.Lock()
				if bs.activeBatch.request != nil {
					bs.exportActiveBatch()
				}
				bs.mu.Unlock()
				return
			case <-timer.C:
				bs.mu.Lock()
				if bs.activeBatch.request != nil {
					bs.exportActiveBatch()
				}
				bs.mu.Unlock()
				timer.Reset(bs.mergeCfg.Timeout)
			case <-bs.resetTimerCh:
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(bs.mergeCfg.Timeout)
			}
		}
	}()

	return nil
}

type batch struct {
	ctx     context.Context
	request Request
	done    chan struct{}
	err     error
}

func newEmptyBatch() *batch {
	return &batch{
		ctx:  context.Background(),
		done: make(chan struct{}),
	}
}

// exportActiveBatch exports the active batch and creates a new one.
// Caller must hold the lock.
func (bs *batchSender) exportActiveBatch() {
	go func(b *batch) {
		b.err = b.request.Export(b.ctx)
		close(b.done)
	}(bs.activeBatch)
	bs.activeBatch = newEmptyBatch()
}

// isActiveBatchReady returns true if the active batch is ready to be exported.
// The batch is ready if it has reached the minimum size or the concurrency limit is reached.
// Caller must hold the lock.
func (bs *batchSender) isActiveBatchReady() bool {
	return bs.activeBatch.request.ItemsCount() >= bs.mergeCfg.MinSizeItems || bs.activeRequests.Load() >= bs.concurrencyLimit
}

func (bs *batchSender) send(ctx context.Context, req Request) error {
	// Stopped batch sender should act as pass-through to allow the queue to be drained.
	if bs.stopped.Load() {
		return bs.nextSender.send(ctx, req)
	}

	bs.activeRequests.Add(1)
	defer bs.activeRequests.Add(^uint64(0))

	if bs.mergeSplitFunc != nil {
		return bs.sendMergeSplitBatch(ctx, req)
	}
	return bs.sendMergeBatch(ctx, req)
}

func (bs *batchSender) sendMergeSplitBatch(ctx context.Context, req Request) error {
	bs.mu.Lock()

	reqs, err := bs.mergeSplitFunc(ctx, bs.activeBatch.request, req, bs.splitCfg.MaxSizeItems)
	if err != nil || len(reqs) == 0 {
		bs.mu.Unlock()
		return err
	}
	if len(reqs) == 1 || bs.activeBatch.request != nil {
		bs.activeBatch.request = reqs[0]
		bs.activeBatch.ctx = ctx
		batch := bs.activeBatch
		var sent bool
		if bs.isActiveBatchReady() || len(reqs) > 1 {
			bs.exportActiveBatch()
			sent = true
		}
		bs.mu.Unlock()
		if sent {
			bs.resetTimerCh <- struct{}{}
		}
		<-batch.done
		if batch.err != nil {
			return batch.err
		}
		reqs = reqs[1:]
	} else {
		bs.mu.Unlock()
	}

	// Intentionally do not put the last request in the active batch to not block it.
	// TODO: Consider including the partial request in the error to avoid double publishing.
	for _, r := range reqs {
		if err := r.Export(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (bs *batchSender) sendMergeBatch(ctx context.Context, req Request) error {
	bs.mu.Lock()

	if bs.activeBatch.request != nil {
		var err error
		req, err = bs.mergeFunc(ctx, bs.activeBatch.request, req)
		if err != nil {
			bs.mu.Unlock()
			return err
		}
	}
	bs.activeBatch.request = req
	bs.activeBatch.ctx = ctx
	batch := bs.activeBatch
	var sent bool
	if bs.isActiveBatchReady() {
		bs.exportActiveBatch()
		sent = true
	}
	bs.mu.Unlock()
	if sent {
		bs.resetTimerCh <- struct{}{}
	}
	<-batch.done
	return batch.err
}

func (bs *batchSender) Shutdown(context.Context) error {
	bs.stopped.Store(true)
	close(bs.shutdownCh)
	return nil
}
