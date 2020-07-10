// Copyright The OpenTelemetry Authors
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
	"fmt"
	"sync"
	"time"

	"github.com/jaegertracing/jaeger/pkg/queue"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/processor"
)

type queuedSpanProcessor struct {
	name                     string
	queue                    *queue.BoundedQueue
	logger                   *zap.Logger
	sender                   consumer.TraceConsumer
	numWorkers               int
	retryOnProcessingFailure bool
	backoffDelay             time.Duration
	statsTags                []tag.Mutator
	stopCh                   chan struct{}
	stopOnce                 sync.Once
}

var _ consumer.TraceConsumer = (*queuedSpanProcessor)(nil)

type queueItem struct {
	queuedTime     time.Time
	td             pdata.Traces
	spanCountStats *processor.SpanCountStats
	ctx            context.Context
}

func newQueuedSpanProcessor(
	params component.ProcessorCreateParams,
	sender consumer.TraceConsumer,
	cfg *Config,
) *queuedSpanProcessor {
	return &queuedSpanProcessor{
		name:                     cfg.Name(),
		queue:                    queue.NewBoundedQueue(cfg.QueueSize, func(item interface{}) {}),
		logger:                   params.Logger,
		numWorkers:               cfg.NumWorkers,
		sender:                   sender,
		statsTags:                []tag.Mutator{tag.Insert(processor.TagProcessorNameKey, cfg.Name())},
		retryOnProcessingFailure: cfg.RetryOnFailure,
		backoffDelay:             cfg.BackoffDelay,
		stopCh:                   make(chan struct{}),
	}
}

// Start is invoked during service startup.
func (sp *queuedSpanProcessor) Start(_ context.Context, _ component.Host) error {
	// emit 0's so that the metric is present and reported, rather than absent
	ctx := obsreport.ProcessorContext(context.Background(), sp.name)
	stats.Record(
		ctx,
		processor.StatTraceBatchesDroppedCount.M(int64(0)),
		processor.StatDroppedSpanCount.M(int64(0)))

	sp.queue.StartConsumers(sp.numWorkers, func(item interface{}) {
		value := item.(*queueItem)
		sp.processItemFromQueue(value)
	})

	// Start a timer to report the queue length.
	ticker := time.NewTicker(1 * time.Second)
	go func() {
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
	}()

	return nil
}

// Stop halts the span processor and all its goroutines.
func (sp *queuedSpanProcessor) Stop() {
	sp.stopOnce.Do(func() {
		close(sp.stopCh)
		sp.queue.Stop()
	})
}

// ConsumeTraces implements the SpanProcessor interface
func (sp *queuedSpanProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	ctx = obsreport.ProcessorContext(ctx, sp.name)
	item := &queueItem{
		queuedTime:     time.Now(),
		td:             td,
		spanCountStats: processor.NewSpanCountStats(td, sp.name),
		ctx:            ctx,
	}

	processor.RecordsSpanCountMetrics(ctx, item.spanCountStats, processor.StatReceivedSpanCount)

	addedToQueue := sp.queue.Produce(item)
	if !addedToQueue {
		// TODO: in principle this may not end in data loss because this can be
		// in the same call stack as the receiver, ie.: the call from the receiver
		// to here is synchronous. This means that actually it could be proper to
		// record this as "refused" instead of "dropped".
		sp.onItemDropped(item, fmt.Errorf("failed to add to the queue: %w", error(nil)))
	} else {
		obsreport.ProcessorTraceDataAccepted(ctx, item.spanCountStats.GetAllSpansCount())
	}
	return nil
}

func (sp *queuedSpanProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

// Shutdown is invoked during service shutdown.
func (sp *queuedSpanProcessor) Shutdown(context.Context) error {
	// TODO: flush the queue.
	return nil
}

func (sp *queuedSpanProcessor) processItemFromQueue(item *queueItem) {
	startTime := time.Now()
	err := sp.sender.ConsumeTraces(item.ctx, item.td)
	if err == nil {
		// Record latency metrics and return
		sendLatencyMs := int64(time.Since(startTime) / time.Millisecond)
		inQueueLatencyMs := int64(time.Since(item.queuedTime) / time.Millisecond)
		_ = stats.RecordWithTags(item.ctx,
			sp.statsTags,
			statSuccessSendOps.M(1),
			statSendLatencyMs.M(sendLatencyMs),
			statInQueueLatencyMs.M(inQueueLatencyMs))

		return
	}

	// There was an error
	_ = stats.RecordWithTags(item.ctx, sp.statsTags, statFailedSendOps.M(1))

	// Immediately drop data on permanent errors.
	if consumererror.IsPermanent(err) {
		// throw away the batch
		sp.onItemDropped(item, err)
		return
	}

	// Immediately drop data on no retires configured.
	if !sp.retryOnProcessingFailure {
		// throw away the batch
		sp.onItemDropped(item, fmt.Errorf("no retry processing %w", err))
		return
	}

	// TODO: (@pjanotti) do not put it back on the end of the queue, retry with it directly.
	// This will have the benefit of keeping the batch closer to related ones in time.
	if !sp.queue.Produce(item) {
		sp.onItemDropped(item, fmt.Errorf("failed to re-enqueue: %w", err))
		return
	}

	sp.logger.Warn("Failed to process batch, re-enqueued", zap.Int("batch-size", item.spanCountStats.GetAllSpansCount()))

	// back-off for configured delay, but get interrupted when shutting down
	if sp.backoffDelay > 0 {
		sp.logger.Warn("Backing off before next attempt", zap.Duration("backoff_delay", sp.backoffDelay))
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

func (sp *queuedSpanProcessor) onItemDropped(item *queueItem, err error) {
	_ = stats.RecordWithTags(item.ctx,
		[]tag.Mutator{tag.Insert(processor.TagProcessorNameKey, sp.name)},
		processor.StatTraceBatchesDroppedCount.M(int64(1)))
	processor.RecordsSpanCountMetrics(item.ctx, item.spanCountStats, processor.StatDroppedSpanCount)

	obsreport.ProcessorTraceDataDropped(item.ctx, item.spanCountStats.GetAllSpansCount())

	sp.logger.Error("Failed to process batch, discarding", zap.Int("batch-size", item.spanCountStats.GetAllSpansCount()), zap.Error(err))
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
		Description: "Current number of batches in the queue",
		TagKeys:     processorTagKeys,
		Aggregation: view.LastValue(),
	}
	countSuccessSendView := &view.View{
		Name:        statSuccessSendOps.Name(),
		Measure:     statSuccessSendOps,
		Description: "The number of successful send operations performed by queued_retry processor",
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}
	countFailuresSendView := &view.View{
		Name:        statFailedSendOps.Name(),
		Measure:     statFailedSendOps,
		Description: "The number of failed send operations performed by queued_retry processor",
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

	legacyViews := []*view.View{queueLengthView, countSuccessSendView, countFailuresSendView, sendLatencyView, inQueueLatencyView}

	return obsreport.ProcessorMetricViews(typeStr, legacyViews)
}
