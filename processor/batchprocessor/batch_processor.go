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

package batchprocessor

import (
	"context"
	"runtime"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/processor"
)

// batch_processor is a component that accepts spans and metrics, places them
// into batches and sends downstream.
//
// batch_processor implements consumer.TracesConsumer and consumer.MetricsConsumer
//
// Batches are sent out with any of the following conditions:
// - batch size reaches cfg.SendBatchSize
// - cfg.Timeout is elapsed since the timestamp when the previous batch was sent out.
type batchProcessor struct {
	name           string
	logger         *zap.Logger
	telemetryLevel configtelemetry.Level

	sendBatchSize    uint32
	timeout          time.Duration
	sendBatchMaxSize uint32

	timer   *time.Timer
	done    chan struct{}
	newItem chan interface{}
	batch   batch

	ctx    context.Context
	cancel context.CancelFunc
}

type batch interface {
	// export the current batch
	export(ctx context.Context) error

	// itemCount returns the size of the current batch
	itemCount() uint32

	// size returns the size in bytes of the current batch
	size() int

	// reset the current batch structure with zero/empty values.
	reset()

	// add item to the current batch
	add(item interface{})
}

type itemWithContext struct {
	ctx  context.Context
	item interface{}
}

var _ consumer.TracesConsumer = (*batchProcessor)(nil)
var _ consumer.MetricsConsumer = (*batchProcessor)(nil)
var _ consumer.LogsConsumer = (*batchProcessor)(nil)

func newBatchProcessor(params component.ProcessorCreateParams, cfg *Config, batch batch, telemetryLevel configtelemetry.Level) *batchProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &batchProcessor{
		name:           cfg.Name(),
		logger:         params.Logger,
		telemetryLevel: telemetryLevel,

		sendBatchSize:    cfg.SendBatchSize,
		sendBatchMaxSize: cfg.SendBatchMaxSize,
		timeout:          cfg.Timeout,
		done:             make(chan struct{}, 1),
		newItem:          make(chan interface{}, runtime.NumCPU()),
		batch:            batch,
		ctx:              ctx,
		cancel:           cancel,
	}
}

func (bp *batchProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (bp *batchProcessor) Start(context.Context, component.Host) error {
	go bp.startProcessingCycle()
	return nil
}

// Shutdown is invoked during service shutdown.
func (bp *batchProcessor) Shutdown(context.Context) error {
	bp.cancel()
	<-bp.done
	return nil
}

func getTokenFromContext(ctx context.Context) string {
	c, ok := client.FromContext(ctx)
	if ok {
		return c.Token
	}
	return ""
}

func (bp *batchProcessor) startProcessingCycle() {
	currentContext := context.Background()
	currentToken := ""
	bp.timer = time.NewTimer(bp.timeout)
	for {
		select {
		case <-bp.ctx.Done():
		DONE:
			for {
				select {
				case itemAndCtxFromChannel := <-bp.newItem:
					itemAndContext := itemAndCtxFromChannel.(itemWithContext)
					currentToken, currentContext = bp.processItemIfTokenUnchanged(currentContext, itemAndContext, currentToken)
				default:
					break DONE
				}
			}
			// This is the close of the channel
			if bp.batch.itemCount() > 0 {
				// TODO: Set a timeout on sendTraces or
				// make it cancellable using the context that Shutdown gets as a parameter
				bp.sendItems(currentContext, statTimeoutTriggerSend)
			}
			close(bp.done)
			return
		case itemAndCtxFromChannel := <-bp.newItem:
			itemAndContext := itemAndCtxFromChannel.(itemWithContext)
			if itemAndContext.item == nil {
				continue
			}
			currentToken, currentContext = bp.processItemIfTokenUnchanged(currentContext, itemAndContext, currentToken)
		case <-bp.timer.C:
			if bp.batch.itemCount() > 0 {
				bp.sendItems(currentContext, statTimeoutTriggerSend)
			}
			bp.resetTimer()
		}
	}
}

func (bp *batchProcessor) processItemIfTokenUnchanged(currentContext context.Context, itemAndContext itemWithContext, currentToken string) (string, context.Context) {
	newToken := getTokenFromContext(itemAndContext.ctx)
	if currentToken != newToken && currentToken != "" {
		bp.timer.Stop()
		bp.sendItems(currentContext, statBatchSizeTriggerSend)
		bp.resetTimer()
	}
	currentToken = newToken
	currentContext = itemAndContext.ctx
	bp.processItem(itemAndContext)
	return currentToken, currentContext
}

func (bp *batchProcessor) processItem(itemAndContext itemWithContext) {
	if bp.sendBatchMaxSize > 0 {
		if td, ok := itemAndContext.item.(pdata.Traces); ok {
			itemCount := bp.batch.itemCount()
			if itemCount+uint32(td.SpanCount()) > bp.sendBatchMaxSize {
				tdRemainSize := splitTrace(int(bp.sendBatchSize-itemCount), td)
				itemAndContext.item = tdRemainSize
				go func() {
					bp.newItem <- itemWithContext{itemAndContext.ctx, td}
				}()
			}
		}
		if td, ok := itemAndContext.item.(pdata.Metrics); ok {
			itemCount := bp.batch.itemCount()
			if itemCount+uint32(td.MetricCount()) > bp.sendBatchMaxSize {
				tdRemainSize := splitMetrics(int(bp.sendBatchSize-itemCount), td)
				itemAndContext.item = tdRemainSize
				go func() {
					bp.newItem <- td
				}()
			}
		}
	}

	bp.batch.add(itemAndContext.item)
	if bp.batch.itemCount() >= bp.sendBatchSize {
		bp.timer.Stop()
		bp.sendItems(itemAndContext.ctx, statBatchSizeTriggerSend)
		bp.resetTimer()
	}
}

func (bp *batchProcessor) resetTimer() {
	bp.timer.Reset(bp.timeout)
}

func (bp *batchProcessor) sendItems(ctx context.Context, measure *stats.Int64Measure) {
	// Add that it came form the trace pipeline?
	statsTags := []tag.Mutator{tag.Insert(processor.TagProcessorNameKey, bp.name)}
	_ = stats.RecordWithTags(ctx, statsTags, measure.M(1), statBatchSendSize.M(int64(bp.batch.itemCount())))

	if bp.telemetryLevel == configtelemetry.LevelDetailed {
		_ = stats.RecordWithTags(ctx, statsTags, statBatchSendSizeBytes.M(int64(bp.batch.size())))
	}

	if err := bp.batch.export(ctx); err != nil {
		bp.logger.Warn("Sender failed", zap.Error(err))
	}
	bp.batch.reset()
}

// ConsumeTraces implements TracesProcessor
func (bp *batchProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	ctx.Value("Token")
	bp.newItem <- itemWithContext{ctx, td}
	return nil
}

// ConsumeTraces implements MetricsProcessor
func (bp *batchProcessor) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	// First thing is convert into a different internal format
	bp.newItem <- itemWithContext{ctx, md}
	return nil
}

// ConsumeLogs implements LogsProcessor
func (bp *batchProcessor) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	bp.newItem <- itemWithContext{ctx, ld}
	return nil
}

// newBatchTracesProcessor creates a new batch processor that batches traces by size or with timeout
func newBatchTracesProcessor(params component.ProcessorCreateParams, trace consumer.TracesConsumer, cfg *Config, telemetryLevel configtelemetry.Level) *batchProcessor {
	return newBatchProcessor(params, cfg, newBatchTraces(trace), telemetryLevel)
}

// newBatchMetricsProcessor creates a new batch processor that batches metrics by size or with timeout
func newBatchMetricsProcessor(params component.ProcessorCreateParams, metrics consumer.MetricsConsumer, cfg *Config, telemetryLevel configtelemetry.Level) *batchProcessor {
	return newBatchProcessor(params, cfg, newBatchMetrics(metrics), telemetryLevel)
}

// newBatchLogsProcessor creates a new batch processor that batches logs by size or with timeout
func newBatchLogsProcessor(params component.ProcessorCreateParams, logs consumer.LogsConsumer, cfg *Config, telemetryLevel configtelemetry.Level) *batchProcessor {
	return newBatchProcessor(params, cfg, newBatchLogs(logs), telemetryLevel)
}

type batchTraces struct {
	nextConsumer consumer.TracesConsumer
	traceData    pdata.Traces
	spanCount    uint32
}

func newBatchTraces(nextConsumer consumer.TracesConsumer) *batchTraces {
	b := &batchTraces{nextConsumer: nextConsumer}
	b.reset()
	return b
}

// add updates current batchTraces by adding new TraceData object
func (bt *batchTraces) add(item interface{}) {
	td := item.(pdata.Traces)
	newSpanCount := td.SpanCount()
	if newSpanCount == 0 {
		return
	}

	bt.spanCount += uint32(newSpanCount)
	td.ResourceSpans().MoveAndAppendTo(bt.traceData.ResourceSpans())
}

func (bt *batchTraces) export(ctx context.Context) error {
	return bt.nextConsumer.ConsumeTraces(ctx, bt.traceData)
}

func (bt *batchTraces) itemCount() uint32 {
	return bt.spanCount
}

func (bt *batchTraces) size() int {
	return bt.traceData.Size()
}

// resets the current batchTraces structure with zero values
func (bt *batchTraces) reset() {
	bt.traceData = pdata.NewTraces()
	bt.spanCount = 0
}

type batchMetrics struct {
	nextConsumer consumer.MetricsConsumer
	metricData   pdata.Metrics
	metricCount  uint32
}

func newBatchMetrics(nextConsumer consumer.MetricsConsumer) *batchMetrics {
	b := &batchMetrics{nextConsumer: nextConsumer}
	b.reset()
	return b
}

func (bm *batchMetrics) export(ctx context.Context) error {
	return bm.nextConsumer.ConsumeMetrics(ctx, bm.metricData)
}

func (bm *batchMetrics) itemCount() uint32 {
	return bm.metricCount
}

func (bm *batchMetrics) size() int {
	return bm.metricData.Size()
}

// resets the current batchMetrics structure with zero/empty values.
func (bm *batchMetrics) reset() {
	bm.metricData = pdata.NewMetrics()
	bm.metricCount = 0
}

func (bm *batchMetrics) add(item interface{}) {
	md := item.(pdata.Metrics)

	newMetricsCount := md.MetricCount()
	if newMetricsCount == 0 {
		return
	}
	bm.metricCount += uint32(newMetricsCount)
	md.ResourceMetrics().MoveAndAppendTo(bm.metricData.ResourceMetrics())
}

type batchLogs struct {
	nextConsumer consumer.LogsConsumer
	logData      pdata.Logs
	logCount     uint32
}

func newBatchLogs(nextConsumer consumer.LogsConsumer) *batchLogs {
	b := &batchLogs{nextConsumer: nextConsumer}
	b.reset()
	return b
}

func (bm *batchLogs) export(ctx context.Context) error {
	return bm.nextConsumer.ConsumeLogs(ctx, bm.logData)
}

func (bm *batchLogs) itemCount() uint32 {
	return bm.logCount
}

func (bm *batchLogs) size() int {
	return bm.logData.SizeBytes()
}

// resets the current batchLogs structure with zero/empty values.
func (bm *batchLogs) reset() {
	bm.logData = pdata.NewLogs()
	bm.logCount = 0
}

func (bm *batchLogs) add(item interface{}) {
	ld := item.(pdata.Logs)

	newLogsCount := ld.LogRecordCount()
	if newLogsCount == 0 {
		return
	}
	bm.logCount += uint32(newLogsCount)
	ld.ResourceLogs().MoveAndAppendTo(bm.logData.ResourceLogs())
}
