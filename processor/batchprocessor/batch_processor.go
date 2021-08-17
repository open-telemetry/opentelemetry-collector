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
	"sync"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/otlp"
	"go.opentelemetry.io/collector/model/pdata"
)

// batch_processor is a component that accepts spans and metrics, places them
// into batches and sends downstream.
//
// batch_processor implements consumer.Traces and consumer.Metrics
//
// Batches are sent out with any of the following conditions:
// - batch size reaches cfg.SendBatchSize
// - cfg.Timeout is elapsed since the timestamp when the previous batch was sent out.
type batchProcessor struct {
	logger           *zap.Logger
	exportCtx        context.Context
	timer            *time.Timer
	timeout          time.Duration
	sendBatchSize    int
	sendBatchMaxSize int

	newItem chan interface{}
	batch   batch

	shutdownC  chan struct{}
	goroutines sync.WaitGroup

	telemetryLevel configtelemetry.Level
}

type batch interface {
	// export the current batch
	export(ctx context.Context, sendBatchMaxSize int) error

	// itemCount returns the size of the current batch
	itemCount() int

	// size returns the size in bytes of the current batch
	size() int

	// add item to the current batch
	add(item interface{})
}

var _ consumer.Traces = (*batchProcessor)(nil)
var _ consumer.Metrics = (*batchProcessor)(nil)
var _ consumer.Logs = (*batchProcessor)(nil)

func newBatchProcessor(set component.ProcessorCreateSettings, cfg *Config, batch batch, telemetryLevel configtelemetry.Level) (*batchProcessor, error) {
	exportCtx, err := tag.New(context.Background(), tag.Insert(processorTagKey, cfg.ID().String()))
	if err != nil {
		return nil, err
	}
	return &batchProcessor{
		logger:         set.Logger,
		exportCtx:      exportCtx,
		telemetryLevel: telemetryLevel,

		sendBatchSize:    int(cfg.SendBatchSize),
		sendBatchMaxSize: int(cfg.SendBatchMaxSize),
		timeout:          cfg.Timeout,
		newItem:          make(chan interface{}, runtime.NumCPU()),
		batch:            batch,
		shutdownC:        make(chan struct{}, 1),
	}, nil
}

func (bp *batchProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// Start is invoked during service startup.
func (bp *batchProcessor) Start(context.Context, component.Host) error {
	bp.goroutines.Add(1)
	go bp.startProcessingCycle()
	return nil
}

// Shutdown is invoked during service shutdown.
func (bp *batchProcessor) Shutdown(context.Context) error {
	close(bp.shutdownC)

	// Wait until all goroutines are done.
	bp.goroutines.Wait()
	return nil
}

func (bp *batchProcessor) startProcessingCycle() {
	defer bp.goroutines.Done()
	bp.timer = time.NewTimer(bp.timeout)
	for {
		select {
		case <-bp.shutdownC:
		DONE:
			for {
				select {
				case item := <-bp.newItem:
					bp.processItem(item)
				default:
					break DONE
				}
			}
			// This is the close of the channel
			if bp.batch.itemCount() > 0 {
				// TODO: Set a timeout on sendTraces or
				// make it cancellable using the context that Shutdown gets as a parameter
				bp.sendItems(statTimeoutTriggerSend)
			}
			return
		case item := <-bp.newItem:
			if item == nil {
				continue
			}
			bp.processItem(item)
		case <-bp.timer.C:
			if bp.batch.itemCount() > 0 {
				bp.sendItems(statTimeoutTriggerSend)
			}
			bp.resetTimer()
		}
	}
}

func (bp *batchProcessor) processItem(item interface{}) {
	bp.batch.add(item)
	sent := false
	for bp.batch.itemCount() >= bp.sendBatchSize {
		sent = true
		bp.sendItems(statBatchSizeTriggerSend)
	}

	if sent {
		bp.stopTimer()
		bp.resetTimer()
	}
}

func (bp *batchProcessor) stopTimer() {
	if !bp.timer.Stop() {
		<-bp.timer.C
	}
}

func (bp *batchProcessor) resetTimer() {
	bp.timer.Reset(bp.timeout)
}

func (bp *batchProcessor) sendItems(triggerMeasure *stats.Int64Measure) {
	// Add that it came form the trace pipeline?
	stats.Record(bp.exportCtx, triggerMeasure.M(1), statBatchSendSize.M(int64(bp.batch.itemCount())))

	if bp.telemetryLevel == configtelemetry.LevelDetailed {
		stats.Record(bp.exportCtx, statBatchSendSizeBytes.M(int64(bp.batch.size())))
	}

	if err := bp.batch.export(bp.exportCtx, bp.sendBatchMaxSize); err != nil {
		bp.logger.Warn("Sender failed", zap.Error(err))
	}
}

// ConsumeTraces implements TracesProcessor
func (bp *batchProcessor) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	bp.newItem <- td
	return nil
}

// ConsumeMetrics implements MetricsProcessor
func (bp *batchProcessor) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	// First thing is convert into a different internal format
	bp.newItem <- md
	return nil
}

// ConsumeLogs implements LogsProcessor
func (bp *batchProcessor) ConsumeLogs(_ context.Context, ld pdata.Logs) error {
	bp.newItem <- ld
	return nil
}

// newBatchTracesProcessor creates a new batch processor that batches traces by size or with timeout
func newBatchTracesProcessor(set component.ProcessorCreateSettings, next consumer.Traces, cfg *Config, telemetryLevel configtelemetry.Level) (*batchProcessor, error) {
	return newBatchProcessor(set, cfg, newBatchTraces(next), telemetryLevel)
}

// newBatchMetricsProcessor creates a new batch processor that batches metrics by size or with timeout
func newBatchMetricsProcessor(set component.ProcessorCreateSettings, next consumer.Metrics, cfg *Config, telemetryLevel configtelemetry.Level) (*batchProcessor, error) {
	return newBatchProcessor(set, cfg, newBatchMetrics(next), telemetryLevel)
}

// newBatchLogsProcessor creates a new batch processor that batches logs by size or with timeout
func newBatchLogsProcessor(set component.ProcessorCreateSettings, next consumer.Logs, cfg *Config, telemetryLevel configtelemetry.Level) (*batchProcessor, error) {
	return newBatchProcessor(set, cfg, newBatchLogs(next), telemetryLevel)
}

type batchTraces struct {
	nextConsumer consumer.Traces
	traceData    pdata.Traces
	spanCount    int
	sizer        pdata.TracesSizer
}

func newBatchTraces(nextConsumer consumer.Traces) *batchTraces {
	return &batchTraces{nextConsumer: nextConsumer, traceData: pdata.NewTraces(), sizer: otlp.NewProtobufTracesMarshaler().(pdata.TracesSizer)}
}

// add updates current batchTraces by adding new TraceData object
func (bt *batchTraces) add(item interface{}) {
	td := item.(pdata.Traces)
	newSpanCount := td.SpanCount()
	if newSpanCount == 0 {
		return
	}

	bt.spanCount += newSpanCount
	td.ResourceSpans().MoveAndAppendTo(bt.traceData.ResourceSpans())
}

func (bt *batchTraces) export(ctx context.Context, sendBatchMaxSize int) error {
	var req pdata.Traces
	if sendBatchMaxSize > 0 && bt.itemCount() > sendBatchMaxSize {
		req = splitTraces(sendBatchMaxSize, bt.traceData)
		bt.spanCount -= sendBatchMaxSize
	} else {
		req = bt.traceData
		bt.traceData = pdata.NewTraces()
		bt.spanCount = 0
	}
	return bt.nextConsumer.ConsumeTraces(ctx, req)
}

func (bt *batchTraces) itemCount() int {
	return bt.spanCount
}

func (bt *batchTraces) size() int {
	return bt.sizer.TracesSize(bt.traceData)
}

type batchMetrics struct {
	nextConsumer   consumer.Metrics
	metricData     pdata.Metrics
	dataPointCount int
	sizer          pdata.MetricsSizer
}

func newBatchMetrics(nextConsumer consumer.Metrics) *batchMetrics {
	return &batchMetrics{nextConsumer: nextConsumer, metricData: pdata.NewMetrics(), sizer: otlp.NewProtobufMetricsMarshaler().(pdata.MetricsSizer)}
}

func (bm *batchMetrics) export(ctx context.Context, sendBatchMaxSize int) error {
	var req pdata.Metrics
	if sendBatchMaxSize > 0 && bm.dataPointCount > sendBatchMaxSize {
		req = splitMetrics(sendBatchMaxSize, bm.metricData)
		bm.dataPointCount -= sendBatchMaxSize
	} else {
		req = bm.metricData
		bm.metricData = pdata.NewMetrics()
		bm.dataPointCount = 0
	}
	return bm.nextConsumer.ConsumeMetrics(ctx, req)
}

func (bm *batchMetrics) itemCount() int {
	return bm.dataPointCount
}

func (bm *batchMetrics) size() int {
	return bm.sizer.MetricsSize(bm.metricData)
}

func (bm *batchMetrics) add(item interface{}) {
	md := item.(pdata.Metrics)

	newDataPointCount := md.DataPointCount()
	if newDataPointCount == 0 {
		return
	}
	bm.dataPointCount += newDataPointCount
	md.ResourceMetrics().MoveAndAppendTo(bm.metricData.ResourceMetrics())
}

type batchLogs struct {
	nextConsumer consumer.Logs
	logData      pdata.Logs
	logCount     int
	sizer        pdata.LogsSizer
}

func newBatchLogs(nextConsumer consumer.Logs) *batchLogs {
	return &batchLogs{nextConsumer: nextConsumer, logData: pdata.NewLogs(), sizer: otlp.NewProtobufLogsMarshaler().(pdata.LogsSizer)}
}

func (bl *batchLogs) export(ctx context.Context, sendBatchMaxSize int) error {
	var req pdata.Logs
	if sendBatchMaxSize > 0 && bl.logCount > sendBatchMaxSize {
		req = splitLogs(sendBatchMaxSize, bl.logData)
		bl.logCount -= sendBatchMaxSize
	} else {
		req = bl.logData
		bl.logData = pdata.NewLogs()
		bl.logCount = 0
	}
	return bl.nextConsumer.ConsumeLogs(ctx, req)
}

func (bl *batchLogs) itemCount() int {
	return bl.logCount
}

func (bl *batchLogs) size() int {
	return bl.sizer.LogsSize(bl.logData)
}

func (bl *batchLogs) add(item interface{}) {
	ld := item.(pdata.Logs)

	newLogsCount := ld.LogRecordCount()
	if newLogsCount == 0 {
		return
	}
	bl.logCount += newLogsCount
	ld.ResourceLogs().MoveAndAppendTo(bl.logData.ResourceLogs())
}
