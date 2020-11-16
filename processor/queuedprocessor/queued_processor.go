// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package queuedprocessor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jaegertracing/jaeger/pkg/queue"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/processor"
)

type queuedProcessor struct {
	name                     string
	queue                    *queue.BoundedQueue
	logger                   *zap.Logger
	traceNext                consumer.TracesConsumer
	metricNext               consumer.MetricsConsumer
	numWorkers               int
	retryOnProcessingFailure bool
	backoffDelay             time.Duration
	stopCh                   chan struct{}
	stopOnce                 sync.Once
	obsrep                   *obsreport.ProcessorObsReport
}

var _ consumer.TracesConsumer = (*queuedProcessor)(nil)
var errorRefused = errors.New("failed to add to the queue")

type queueItem interface {
	context() context.Context
	queuedTime() time.Time
	export(sp *queuedProcessor) error
	onAccepted()
	// Returns a new queue item that contains the items left to be exported.
	onPartialError(partialErr consumererror.PartialError) queueItem
	onRefused(logger *zap.Logger, err error)
	onDropped(logger *zap.Logger, err error)
}

type baseQueueItem struct {
	ctx    context.Context
	qt     time.Time
	obsrep *obsreport.ProcessorObsReport
}

func (item *baseQueueItem) context() context.Context {
	return item.ctx
}

func (item *baseQueueItem) queuedTime() time.Time {
	return item.qt
}

type traceQueueItem struct {
	baseQueueItem
	td             pdata.Traces
	spanCountStats *processor.SpanCountStats
}

func newTraceQueueItem(ctx context.Context, td pdata.Traces, obsrep *obsreport.ProcessorObsReport) queueItem {
	return &traceQueueItem{
		baseQueueItem:  baseQueueItem{ctx: ctx, qt: time.Now(), obsrep: obsrep},
		td:             td,
		spanCountStats: processor.NewSpanCountStats(td),
	}
}

func (item *traceQueueItem) onAccepted() {
	processor.RecordsSpanCountMetrics(item.ctx, item.spanCountStats, processor.StatReceivedSpanCount)
	item.obsrep.TracesAccepted(item.ctx, item.spanCountStats.GetAllSpansCount())
}

func (item *traceQueueItem) onPartialError(partialErr consumererror.PartialError) queueItem {
	return newTraceQueueItem(item.ctx, partialErr.GetTraces(), item.obsrep)
}

func (item *traceQueueItem) onRefused(logger *zap.Logger, err error) {
	// Count the StatReceivedSpanCount even if items were refused.
	processor.RecordsSpanCountMetrics(item.ctx, item.spanCountStats, processor.StatReceivedSpanCount)

	item.obsrep.TracesRefused(item.ctx, item.spanCountStats.GetAllSpansCount())

	// TODO: in principle this may not end in data loss because this can be
	// in the same call stack as the receiver, ie.: the call from the receiver
	// to here is synchronous. This means that actually it could be proper to
	// record this as "refused" instead of "dropped".
	stats.Record(item.ctx, processor.StatTraceBatchesDroppedCount.M(int64(1)))
	processor.RecordsSpanCountMetrics(item.ctx, item.spanCountStats, processor.StatDroppedSpanCount)

	logger.Error("Failed to process batch, refused", zap.Int("#spans", item.spanCountStats.GetAllSpansCount()), zap.Error(err))
}

func (item *traceQueueItem) onDropped(logger *zap.Logger, err error) {
	item.obsrep.TracesDropped(item.ctx, item.spanCountStats.GetAllSpansCount())

	stats.Record(item.ctx, processor.StatTraceBatchesDroppedCount.M(int64(1)))
	processor.RecordsSpanCountMetrics(item.ctx, item.spanCountStats, processor.StatDroppedSpanCount)
	logger.Error("Failed to process batch, discarding", zap.Int("#spans", item.spanCountStats.GetAllSpansCount()), zap.Error(err))
}

func (item *traceQueueItem) export(sp *queuedProcessor) error {
	return sp.traceNext.ConsumeTraces(item.ctx, item.td)
}

type metricsQueueItem struct {
	baseQueueItem
	md        pdata.Metrics
	numPoints int
}

func newMetricsQueueItem(ctx context.Context, md pdata.Metrics, obsrep *obsreport.ProcessorObsReport) queueItem {
	_, numPoints := md.MetricAndDataPointCount()
	return &metricsQueueItem{
		baseQueueItem: baseQueueItem{ctx: ctx, qt: time.Now(), obsrep: obsrep},
		md:            md,
		numPoints:     numPoints,
	}
}

func (item *metricsQueueItem) onAccepted() {
	item.obsrep.MetricsAccepted(item.ctx, item.numPoints)
}

func (item *metricsQueueItem) onPartialError(consumererror.PartialError) queueItem {
	// TODO: implement this.
	return item
}

func (item *metricsQueueItem) onRefused(logger *zap.Logger, err error) {
	item.obsrep.MetricsRefused(item.ctx, item.numPoints)

	logger.Error("Failed to process batch, refused", zap.Int("#points", item.numPoints), zap.Error(err))
}

func (item *metricsQueueItem) onDropped(logger *zap.Logger, err error) {
	stats.Record(item.ctx, processor.StatTraceBatchesDroppedCount.M(int64(1)))
	item.obsrep.MetricsDropped(item.ctx, item.numPoints)

	logger.Error("Failed to process batch, discarding", zap.Int("#points", item.numPoints), zap.Error(err))
}

func (item *metricsQueueItem) export(sp *queuedProcessor) error {
	return sp.metricNext.ConsumeMetrics(item.ctx, item.md)
}

func newQueuedTracesProcessor(
	params component.ProcessorCreateParams,
	nextConsumer consumer.TracesConsumer,
	cfg *Config,
) *queuedProcessor {
	return &queuedProcessor{
		name:                     cfg.Name(),
		queue:                    queue.NewBoundedQueue(cfg.QueueSize, func(item interface{}) {}),
		logger:                   params.Logger,
		numWorkers:               cfg.NumWorkers,
		traceNext:                nextConsumer,
		metricNext:               nil,
		retryOnProcessingFailure: cfg.RetryOnFailure,
		backoffDelay:             cfg.BackoffDelay,
		stopCh:                   make(chan struct{}),
		obsrep:                   obsreport.NewProcessorObsReport(configtelemetry.GetMetricsLevelFlagValue(), cfg.Name()),
	}
}

func newQueuedMetricsProcessor(
	params component.ProcessorCreateParams,
	nextConsumer consumer.MetricsConsumer,
	cfg *Config,
) *queuedProcessor {
	return &queuedProcessor{
		name:                     cfg.Name(),
		queue:                    queue.NewBoundedQueue(cfg.QueueSize, func(item interface{}) {}),
		logger:                   params.Logger,
		numWorkers:               cfg.NumWorkers,
		traceNext:                nil,
		metricNext:               nextConsumer,
		retryOnProcessingFailure: cfg.RetryOnFailure,
		backoffDelay:             cfg.BackoffDelay,
		stopCh:                   make(chan struct{}),
		obsrep:                   obsreport.NewProcessorObsReport(configtelemetry.GetMetricsLevelFlagValue(), cfg.Name()),
	}
}

// Start is invoked during service startup.
func (sp *queuedProcessor) Start(ctx context.Context, _ component.Host) error {
	// emit 0's so that the metric is present and reported, rather than absent
	statsTags := []tag.Mutator{tag.Insert(processor.TagProcessorNameKey, sp.name)}
	_ = stats.RecordWithTags(
		ctx,
		statsTags,
		processor.StatTraceBatchesDroppedCount.M(int64(0)),
		processor.StatDroppedSpanCount.M(int64(0)))

	sp.queue.StartConsumers(sp.numWorkers, func(item interface{}) {
		value := item.(queueItem)
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
				_ = stats.RecordWithTags(
					context.Background(),
					statsTags,
					statQueueLength.M(int64(sp.queue.Size())))
			}
		}
	}()

	return nil
}

// ConsumeTraces implements the TracesProcessor interface
func (sp *queuedProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	item := newTraceQueueItem(ctx, td, sp.obsrep)

	addedToQueue := sp.queue.Produce(item)
	if !addedToQueue {
		item.onRefused(sp.logger, errorRefused)
		return errorRefused
	}

	item.onAccepted()
	return nil
}

// ConsumeMetrics implements the MetricsProcessor interface
func (sp *queuedProcessor) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	item := newMetricsQueueItem(ctx, md, sp.obsrep)

	addedToQueue := sp.queue.Produce(item)
	if !addedToQueue {
		item.onRefused(sp.logger, errorRefused)
		return errorRefused
	}

	item.onAccepted()
	return nil
}

func (sp *queuedProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

// Shutdown is invoked during service shutdown.
func (sp *queuedProcessor) Shutdown(context.Context) error {
	err := componenterror.ErrAlreadyStopped
	sp.stopOnce.Do(func() {
		err = nil
		close(sp.stopCh)
		sp.queue.Stop()
	})
	return err
}

func (sp *queuedProcessor) processItemFromQueue(item queueItem) {
	startTime := time.Now()
	err := item.export(sp)
	if err == nil {
		// Record latency metrics and return
		sendLatencyMs := int64(time.Since(startTime) / time.Millisecond)
		inQueueLatencyMs := int64(time.Since(item.queuedTime()) / time.Millisecond)
		stats.Record(item.context(),
			statSuccessSendOps.M(1),
			statSendLatencyMs.M(sendLatencyMs),
			statInQueueLatencyMs.M(inQueueLatencyMs))

		return
	}

	// There was an error
	stats.Record(item.context(), statFailedSendOps.M(1))

	// Immediately drop data on permanent errors.
	if consumererror.IsPermanent(err) {
		// throw away the batch
		item.onDropped(sp.logger, err)
		return
	}

	// If partial error, update data and stats with non exported data.
	if partialErr, isPartial := err.(consumererror.PartialError); isPartial {
		item = item.onPartialError(partialErr)
	}

	// Immediately drop data on no retries configured.
	if !sp.retryOnProcessingFailure {
		// throw away the batch
		item.onDropped(sp.logger, fmt.Errorf("no retry processing %w", err))
		return
	}

	// TODO: (@pjanotti) do not put it back on the end of the queue, retry with it directly.
	// This will have the benefit of keeping the batch closer to related ones in time.
	if !sp.queue.Produce(item) {
		item.onDropped(sp.logger, fmt.Errorf("failed to re-enqueue: %w", err))
		return
	}

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
