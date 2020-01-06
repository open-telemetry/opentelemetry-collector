// Copyright 2019, OpenTelemetry Authors
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

package queuedprocessor

import (
	"context"
	"sync"
	"time"

	"github.com/jaegertracing/jaeger/pkg/queue"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumererror"
	"github.com/open-telemetry/opentelemetry-collector/internal/collector/telemetry"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	"github.com/open-telemetry/opentelemetry-collector/processor/batchprocessor"
)

type queuedSpanProcessor struct {
	name                     string
	queue                    *queue.BoundedQueue
	logger                   *zap.Logger
	sender                   consumer.TraceConsumer
	numWorkers               int
	retryOnProcessingFailure bool
	backoffDelay             time.Duration
	stopCh                   chan struct{}
	stopOnce                 sync.Once
}

var _ consumer.TraceConsumer = (*queuedSpanProcessor)(nil)

type queueItem struct {
	queuedTime time.Time
	td         consumerdata.TraceData
	ctx        context.Context
}

// NewQueuedSpanProcessor returns a span processor that maintains a bounded
// in-memory queue of span batches, and sends out span batches using the
// provided sender
func NewQueuedSpanProcessor(sender consumer.TraceConsumer, opts ...Option) processor.TraceProcessor {
	options := Options.apply(opts...)
	sp := newQueuedSpanProcessor(sender, options)

	sp.queue.StartConsumers(sp.numWorkers, func(item interface{}) {
		value := item.(*queueItem)
		sp.processItemFromQueue(value)
	})

	// Start a timer to report the queue length.
	ctx, _ := tag.New(context.Background(), tag.Upsert(processor.TagProcessorNameKey, sp.name))
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
		batcher := batchprocessor.NewBatcher(sp.name, sp.logger, sp, options.batchingOptions...)
		return batcher
	}

	return sp
}

func newQueuedSpanProcessor(sender consumer.TraceConsumer, opts options) *queuedSpanProcessor {
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

// Start is invoked during service startup.
func (sp *queuedSpanProcessor) Start(host component.Host) error {
	return nil
}

// Stop halts the span processor and all its goroutines.
func (sp *queuedSpanProcessor) Stop() {
	sp.stopOnce.Do(func() {
		close(sp.stopCh)
		sp.queue.Stop()
	})
}

// ConsumeTraceData implements the SpanProcessor interface
func (sp *queuedSpanProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	item := &queueItem{
		queuedTime: time.Now(),
		td:         td,
		ctx:        ctx,
	}

	statsTags := processor.StatsTagsForBatch(sp.name, processor.ServiceNameForNode(td.Node), td.SourceFormat)
	numSpans := len(td.Spans)
	stats.RecordWithTags(context.Background(), statsTags, processor.StatReceivedSpanCount.M(int64(numSpans)))

	addedToQueue := sp.queue.Produce(item)
	if !addedToQueue {
		sp.onItemDropped(item, statsTags)
	}
	return nil
}

func (sp *queuedSpanProcessor) GetCapabilities() processor.Capabilities {
	return processor.Capabilities{MutatesConsumedData: false}
}

// Shutdown is invoked during service shutdown.
func (sp *queuedSpanProcessor) Shutdown() error {
	// TODO: flush the queue.
	return nil
}

func (sp *queuedSpanProcessor) processItemFromQueue(item *queueItem) {
	startTime := time.Now()
	err := sp.sender.ConsumeTraceData(item.ctx, item.td)
	if err == nil {
		// Record latency metrics and return
		sendLatencyMs := int64(time.Since(startTime) / time.Millisecond)
		inQueueLatencyMs := int64(time.Since(item.queuedTime) / time.Millisecond)
		statsTags := processor.StatsTagsForBatch(sp.name, processor.ServiceNameForNode(item.td.Node), item.td.SourceFormat)
		stats.RecordWithTags(context.Background(),
			statsTags,
			statSuccessSendOps.M(1),
			statSendLatencyMs.M(sendLatencyMs),
			statInQueueLatencyMs.M(inQueueLatencyMs))

		return
	}

	// There was an error
	statsTags := processor.StatsTagsForBatch(sp.name, processor.ServiceNameForNode(item.td.Node), item.td.SourceFormat)

	// Immediately drop data on permanent errors. In this context permanent
	// errors indicate some kind of bad data.
	if consumererror.IsPermanent(err) {
		numSpans := len(item.td.Spans)
		sp.logger.Warn(
			"Unrecoverable bad data error",
			zap.String("processor", sp.name),
			zap.Int("#spans", numSpans),
			zap.String("spanFormat", item.td.SourceFormat),
			zap.Error(err))

		stats.RecordWithTags(
			context.Background(),
			statsTags,
			processor.StatBadBatchDroppedSpanCount.M(int64(numSpans)))

		return
	}

	stats.RecordWithTags(context.Background(), statsTags, statFailedSendOps.M(1))
	batchSize := len(item.td.Spans)
	sp.logger.Warn("Sender failed", zap.String("processor", sp.name), zap.Error(err), zap.String("spanFormat", item.td.SourceFormat))
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
			zap.Duration("backoff_delay", sp.backoffDelay))
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
		zap.String("spanSource", item.td.SourceFormat))
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

	processorTagKeys := []tag.Key{processor.TagProcessorNameKey}

	queueLengthView := &view.View{
		Name:        statQueueLength.Name(),
		Measure:     statQueueLength,
		Description: "Current number of batches in the queued exporter",
		TagKeys:     processorTagKeys,
		Aggregation: view.LastValue(),
	}
	countSuccessSendView := &view.View{
		Name:        statSuccessSendOps.Name(),
		Measure:     statSuccessSendOps,
		Description: "The number of successful send operations performed by queued exporter",
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}
	countFailuresSendView := &view.View{
		Name:        statFailedSendOps.Name(),
		Measure:     statFailedSendOps,
		Description: "The number of failed send operations performed by queued exporter",
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	latencyDistributionAggregation := view.Distribution(10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 10000, 20000, 30000, 50000)

	sendLatencyView := &view.View{
		Name:        statSendLatencyMs.Name(),
		Measure:     statSendLatencyMs,
		Description: "The latency of the successful send operations.",
		TagKeys:     processorTagKeys,
		Aggregation: latencyDistributionAggregation,
	}
	inQueueLatencyView := &view.View{
		Name:        statInQueueLatencyMs.Name(),
		Measure:     statInQueueLatencyMs,
		Description: "The \"in queue\" latency of the successful send operations.",
		TagKeys:     processorTagKeys,
		Aggregation: latencyDistributionAggregation,
	}

	return []*view.View{queueLengthView, countSuccessSendView, countFailuresSendView, sendLatencyView, inQueueLatencyView}
}
