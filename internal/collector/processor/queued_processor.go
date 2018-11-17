// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processor

import (
	"sync"
	"time"

	"github.com/jaegertracing/jaeger/pkg/queue"
	"go.uber.org/zap"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
)

type queuedSpanProcessor struct {
	queue                    *queue.BoundedQueue
	logger                   *zap.Logger
	sender                   SpanProcessor
	numWorkers               int
	retryOnProcessingFailure bool
	backoffDelay             time.Duration
	stopCh                   chan struct{}
	stopOnce                 sync.Once
}

var _ SpanProcessor = (*queuedSpanProcessor)(nil)

type queueItem struct {
	queuedTime time.Time
	batch      *agenttracepb.ExportTraceServiceRequest
	spanFormat string
}

// NewQueuedSpanProcessor returns a span processor that maintains a bounded
// in-memory queue of span batches, and sends out span batches using the
// provided sender
func NewQueuedSpanProcessor(sender SpanProcessor, opts ...Option) SpanProcessor {
	sp := newQueuedSpanProcessor(sender, opts...)

	sp.queue.StartConsumers(sp.numWorkers, func(item interface{}) {
		value := item.(*queueItem)
		sp.processItemFromQueue(value)
	})

	return sp
}

func newQueuedSpanProcessor(sender SpanProcessor, opts ...Option) *queuedSpanProcessor {
	options := Options.apply(opts...)
	boundedQueue := queue.NewBoundedQueue(options.queueSize, func(item interface{}) {})
	return &queuedSpanProcessor{
		queue:                    boundedQueue,
		logger:                   options.logger,
		numWorkers:               options.numWorkers,
		sender:                   sender,
		retryOnProcessingFailure: options.retryOnProcessingFailure,
		backoffDelay:             options.backoffDelay,
		stopCh:                   make(chan struct{}),
	}
}

// Stop halts the span processor and all its goroutines.
func (sp *queuedSpanProcessor) Stop() {
	sp.stopOnce.Do(func() {
		close(sp.stopCh)
		sp.queue.Stop()
	})
}

// ProcessSpans implements the SpanProcessor interface
func (sp *queuedSpanProcessor) ProcessSpans(batch *agenttracepb.ExportTraceServiceRequest, spanFormat string) (failures uint64, err error) {
	allAdded := sp.enqueueSpanBatch(batch, spanFormat)
	if !allAdded {
		failures = uint64(len(batch.Spans))
	}
	return
}

func (sp *queuedSpanProcessor) enqueueSpanBatch(batch *agenttracepb.ExportTraceServiceRequest, spanFormat string) bool {
	item := &queueItem{
		queuedTime: time.Now(),
		batch:      batch,
		spanFormat: spanFormat,
	}
	addedToQueue := sp.queue.Produce(item)
	if !addedToQueue {
		sp.onItemDropped(item)
	}
	return addedToQueue
}

func (sp *queuedSpanProcessor) processItemFromQueue(item *queueItem) {
	// TODO: @(pjanotti) metrics: startTime := time.Now()
	// TODO:
	_, err := sp.sender.ProcessSpans(item.batch, item.spanFormat)
	if err != nil {
		batchSize := len(item.batch.Spans)
		if !sp.retryOnProcessingFailure {
			// throw away the batch
			sp.logger.Error("Failed to process batch, discarding", zap.Int("batch-size", batchSize))
			sp.onItemDropped(item)
		} else {
			// TODO: (@pjanotti) do not put it back on the end of the queue, retry with it directly.
			// This will have the benefit of keeping the batch closer to related ones in time.
			if !sp.queue.Produce(item) {
				sp.logger.Error("Failed to process batch and failed to re-enqueue", zap.Int("batch-size", batchSize))
				sp.onItemDropped(item)
			} else {
				sp.logger.Warn("Failed to process batch, re-enqueued", zap.Int("batch-size", batchSize))
			}
		}
		// back-off for configured delay, but get interrupted when shutting down
		if sp.backoffDelay > 0 {
			sp.logger.Warn("Backing off before next attempt",
				zap.Duration("backoff-delay", sp.backoffDelay))
			select {
			case <-sp.stopCh:
				sp.logger.Info("Interrupted due to shutdown")
				break
			case <-time.After(sp.backoffDelay):
				sp.logger.Info("Resume processing")
				break
			}
		}
	}
}

func (sp *queuedSpanProcessor) onItemDropped(item *queueItem) {
	sp.logger.Warn("Span batch dropped",
		zap.Int("#spans", len(item.batch.Spans)),
		zap.String("spanSource", item.spanFormat))
}
