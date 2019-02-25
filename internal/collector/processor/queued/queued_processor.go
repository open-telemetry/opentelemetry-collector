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

package queued

import (
	"context"
	"sync"
	"time"

	"github.com/jaegertracing/jaeger/pkg/queue"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/internal/collector/processor"
	"github.com/census-instrumentation/opencensus-service/internal/collector/processor/nodebatcher"
	"github.com/census-instrumentation/opencensus-service/internal/collector/telemetry"
)

type queuedSpanProcessor struct {
	name                     string
	queue                    *queue.BoundedQueue
	logger                   *zap.Logger
	sender                   processor.SpanProcessor
	numWorkers               int
	retryOnProcessingFailure bool
	backoffDelay             time.Duration
	stopCh                   chan struct{}
	stopOnce                 sync.Once
}

var _ processor.SpanProcessor = (*queuedSpanProcessor)(nil)

type queueItem struct {
	queuedTime time.Time
	td         data.TraceData
	spanFormat string
}

// NewQueuedSpanProcessor returns a span processor that maintains a bounded
// in-memory queue of span batches, and sends out span batches using the
// provided sender
func NewQueuedSpanProcessor(sender processor.SpanProcessor, opts ...Option) processor.SpanProcessor {
	options := Options.apply(opts...)
	sp := newQueuedSpanProcessor(sender, options)

	sp.queue.StartConsumers(sp.numWorkers, func(item interface{}) {
		value := item.(*queueItem)
		sp.processItemFromQueue(value)
	})

	// Start a timer to report the queue length.
	ctx, _ := tag.New(context.Background(), tag.Upsert(processor.TagExporterNameKey, sp.name))
	ticker := time.NewTicker(1 * time.Second)
	go func(ctx context.Context) {
		defer ticker.Stop()
		for {
			select {
			case <-sp.stopCh:
				return
			case <-ticker.C:
				length := int64(sp.queue.Size())
				stats.Record(ctx, statQueueLength.M(length))
			}
		}
	}(ctx)

	if options.batchingEnabled {
		sp.logger.Info("Using queued processor with batching.")
		batcher := nodebatcher.NewBatcher(sp.name, sp.logger, sp, options.batchingOptions...)
		return batcher
	}

	return sp
}

func newQueuedSpanProcessor(sender processor.SpanProcessor, opts options) *queuedSpanProcessor {
	boundedQueue := queue.NewBoundedQueue(opts.queueSize, func(item interface{}) {})
	return &queuedSpanProcessor{
		name:                     opts.name,
		queue:                    boundedQueue,
		logger:                   opts.logger,
		numWorkers:               opts.numWorkers,
		sender:                   sender,
		retryOnProcessingFailure: opts.retryOnProcessingFailure,
		backoffDelay:             opts.backoffDelay,
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
func (sp *queuedSpanProcessor) ProcessSpans(td data.TraceData, spanFormat string) (failures uint64, err error) {
	allAdded := sp.enqueueSpanBatch(td, spanFormat)
	if !allAdded {
		failures = uint64(len(td.Spans))
	}
	return
}

func (sp *queuedSpanProcessor) enqueueSpanBatch(td data.TraceData, spanFormat string) bool {
	item := &queueItem{
		queuedTime: time.Now(),
		td:         td,
		spanFormat: spanFormat,
	}

	statsTags := processor.StatsTagsForBatch(sp.name, processor.ServiceNameForNode(td.Node), spanFormat)
	numSpans := len(td.Spans)
	stats.RecordWithTags(context.Background(), statsTags, processor.StatReceivedSpanCount.M(int64(numSpans)))

	addedToQueue := sp.queue.Produce(item)
	if !addedToQueue {
		sp.onItemDropped(item, statsTags)
	}
	return addedToQueue
}

func (sp *queuedSpanProcessor) processItemFromQueue(item *queueItem) {
	startTime := time.Now()
	_, err := sp.sender.ProcessSpans(item.td, item.spanFormat)
	if err == nil {
		// Record latency metrics and return
		sendLatencyMs := int64(time.Since(startTime) / time.Millisecond)
		inQueueLatencyMs := int64(time.Since(item.queuedTime) / time.Millisecond)
		statsTags := processor.StatsTagsForBatch(sp.name, processor.ServiceNameForNode(item.td.Node), item.spanFormat)
		stats.RecordWithTags(context.Background(),
			statsTags,
			statSuccessSendOps.M(1),
			statSendLatencyMs.M(sendLatencyMs),
			statInQueueLatencyMs.M(inQueueLatencyMs))

		return
	}

	// There was an error
	statsTags := processor.StatsTagsForBatch(sp.name, processor.ServiceNameForNode(item.td.Node), item.spanFormat)
	stats.RecordWithTags(context.Background(), statsTags, statFailedSendOps.M(1))
	batchSize := len(item.td.Spans)
	sp.logger.Warn("Sender failed", zap.String("processor", sp.name), zap.Error(err), zap.String("spanFormat", item.spanFormat))
	if !sp.retryOnProcessingFailure {
		// throw away the batch
		sp.logger.Error("Failed to process batch, discarding", zap.String("processor", sp.name), zap.Int("batch-size", batchSize))
		sp.onItemDropped(item, statsTags)
	} else {
		// TODO: (@pjanotti) do not put it back on the end of the queue, retry with it directly.
		// This will have the benefit of keeping the batch closer to related ones in time.
		if !sp.queue.Produce(item) {
			sp.logger.Error("Failed to process batch and failed to re-enqueue", zap.String("processor", sp.name), zap.Int("batch-size", batchSize))
			sp.onItemDropped(item, statsTags)
		} else {
			sp.logger.Warn("Failed to process batch, re-enqueued", zap.String("processor", sp.name), zap.Int("batch-size", batchSize))
		}
	}

	// back-off for configured delay, but get interrupted when shutting down
	if sp.backoffDelay > 0 {
		sp.logger.Warn("Backing off before next attempt",
			zap.String("processor", sp.name),
			zap.Duration("backoff-delay", sp.backoffDelay))
		select {
		case <-sp.stopCh:
			sp.logger.Info("Interrupted due to shutdown", zap.String("processor", sp.name))
			break
		case <-time.After(sp.backoffDelay):
			sp.logger.Info("Resume processing", zap.String("processor", sp.name))
			break
		}
	}
}

func (sp *queuedSpanProcessor) onItemDropped(item *queueItem, statsTags []tag.Mutator) {
	numSpans := len(item.td.Spans)
	stats.RecordWithTags(context.Background(), statsTags, processor.StatDroppedSpanCount.M(int64(numSpans)))

	sp.logger.Warn("Span batch dropped",
		zap.String("processor", sp.name),
		zap.Int("#spans", len(item.td.Spans)),
		zap.String("spanSource", item.spanFormat))
}

// Variables related to metrics specific to queued processor.
var (
	statInQueueLatencyMs = stats.Int64("queue_latency", "Latency (in milliseconds) that a batch stayed in queue", stats.UnitMilliseconds)
	statSendLatencyMs    = stats.Int64("send_latency", "Latency (in milliseconds) to send a batch", stats.UnitMilliseconds)

	statSuccessSendOps = stats.Int64("success_send", "Number of successful send operations", stats.UnitDimensionless)
	statFailedSendOps  = stats.Int64("fail_send", "Number of failed send operations", stats.UnitDimensionless)

	statQueueLength = stats.Int64("queue_length", "Current length of the queue (in batches)", stats.UnitDimensionless)
)

// MetricViews return the metrics views according to given telemetry level.
func MetricViews(level telemetry.Level) []*view.View {
	if level == telemetry.None {
		return nil
	}

	tagKeys := processor.MetricTagKeys(level)
	if tagKeys == nil {
		return nil
	}

	exporterTagKeys := []tag.Key{processor.TagExporterNameKey}

	queueLengthView := &view.View{
		Name:        statQueueLength.Name(),
		Measure:     statQueueLength,
		Description: "Current number of batches in the queued exporter",
		TagKeys:     exporterTagKeys,
		Aggregation: view.LastValue(),
	}
	countSuccessSendView := &view.View{
		Name:        statSuccessSendOps.Name(),
		Measure:     statSuccessSendOps,
		Description: "The number of successful send operations performed by queued exporter",
		TagKeys:     tagKeys,
		Aggregation: view.Count(),
	}
	countFailuresSendView := &view.View{
		Name:        statFailedSendOps.Name(),
		Measure:     statFailedSendOps,
		Description: "The number of failed send operations performed by queued exporter",
		TagKeys:     tagKeys,
		Aggregation: view.Count(),
	}

	latencyDistributionAggregation := view.Distribution(10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 10000, 20000, 30000, 50000)

	sendLatencyView := &view.View{
		Name:        statSendLatencyMs.Name(),
		Measure:     statSendLatencyMs,
		Description: "The latency of the successful send operations.",
		TagKeys:     exporterTagKeys,
		Aggregation: latencyDistributionAggregation,
	}
	inQueueLatencyView := &view.View{
		Name:        statInQueueLatencyMs.Name(),
		Measure:     statInQueueLatencyMs,
		Description: "The \"in queue\" latency of the successful send operations.",
		TagKeys:     exporterTagKeys,
		Aggregation: latencyDistributionAggregation,
	}

	return []*view.View{queueLengthView, countSuccessSendView, countFailuresSendView, sendLatencyView, inQueueLatencyView}
}
