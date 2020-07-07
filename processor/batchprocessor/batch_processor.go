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

package batchprocessor

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/processor"
)

// batch_processor is a component that accepts spans and metrics, places them
// into batches and sends downstream.
//
// batch_processor implements consumer.TraceConsumer and consumer.MetricsConsumer
//
// Batches are sent out with any of the following conditions:
// - batch size reaches cfg.SendBatchSize
// - cfg.Timeout is elapsed since the timestamp when the previous batch was sent out.
type batchProcessor struct {
	name   string
	logger *zap.Logger

	sendBatchSize uint32
	timeout       time.Duration

	timer *time.Timer
	done  chan struct{}
}

type batchTraceProcessor struct {
	batchProcessor

	traceConsumer consumer.TraceConsumer

	batchTraces  *batchTraces
	newTraceItem chan pdata.Traces
}

type batchMetricProcessor struct {
	batchProcessor

	metricsConsumer consumer.MetricsConsumer

	batchMetrics  *batchMetrics
	newMetricItem chan pdata.Metrics
}

var _ consumer.TraceConsumer = (*batchTraceProcessor)(nil)
var _ consumer.MetricsConsumer = (*batchMetricProcessor)(nil)

// newBatchTracesProcessor creates a new batch processor that batches traces by size or with timeout
func newBatchTracesProcessor(params component.ProcessorCreateParams, trace consumer.TraceConsumer, cfg *Config) *batchTraceProcessor {
	p := &batchTraceProcessor{
		batchProcessor: batchProcessor{
			name:   cfg.Name(),
			logger: params.Logger,

			sendBatchSize: cfg.SendBatchSize,
			timeout:       cfg.Timeout,
			done:          make(chan struct{}),
		},
		traceConsumer: trace,

		batchTraces:  newBatchTraces(),
		newTraceItem: make(chan pdata.Traces, 1),
	}

	go p.startProcessingCycle()

	return p
}

// newBatchMetricsProcessor creates a new batch processor that batches metrics by size or with timeout
func newBatchMetricsProcessor(params component.ProcessorCreateParams, metrics consumer.MetricsConsumer, cfg *Config) *batchMetricProcessor {
	p := &batchMetricProcessor{
		batchProcessor: batchProcessor{
			name:   cfg.Name(),
			logger: params.Logger,

			sendBatchSize: cfg.SendBatchSize,
			timeout:       cfg.Timeout,
			done:          make(chan struct{}),
		},
		metricsConsumer: metrics,

		batchMetrics:  newBatchMetrics(),
		newMetricItem: make(chan pdata.Metrics, 1),
	}

	go p.startProcessingCycle()

	return p
}

// ConsumeTraces implements batcher as a SpanProcessor and takes the provided spans and adds them to
// batches.
func (bp *batchTraceProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	bp.newTraceItem <- td
	return nil
}

func (bp *batchTraceProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (bp *batchTraceProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (bp *batchTraceProcessor) Shutdown(context.Context) error {
	close(bp.done)
	return nil
}

func (bp *batchTraceProcessor) startProcessingCycle() {
	bp.timer = time.NewTimer(bp.timeout)
	for {
		select {
		case td := <-bp.newTraceItem:
			bp.batchTraces.add(td)

			if bp.batchTraces.getSpanCount() >= bp.sendBatchSize {
				bp.timer.Stop()
				bp.sendItems(statBatchSizeTriggerSend)
				bp.resetTimer()
			}
		case <-bp.timer.C:
			if bp.batchTraces.hasData() {
				bp.sendItems(statTimeoutTriggerSend)
			}
			bp.resetTimer()
		case <-bp.done:
			if bp.batchTraces.hasData() {
				// TODO: Set a timeout on sendTraces or
				// make it cancellable using the context that Shutdown gets as a parameter
				bp.sendItems(statTimeoutTriggerSend)
			}
			return
		}
	}
}

func (bp *batchTraceProcessor) resetTimer() {
	bp.timer.Reset(bp.timeout)
}

func (bp *batchTraceProcessor) sendItems(measure *stats.Int64Measure) {
	// Add that it came form the trace pipeline?
	statsTags := []tag.Mutator{tag.Insert(processor.TagProcessorNameKey, bp.name)}
	_ = stats.RecordWithTags(context.Background(), statsTags, measure.M(1))
	_ = stats.RecordWithTags(context.Background(), statsTags, statBatchSendSize.M(int64(bp.batchTraces.getSpanCount())))

	if err := bp.traceConsumer.ConsumeTraces(context.Background(), bp.batchTraces.getTraceData()); err != nil {
		bp.logger.Warn("Sender failed", zap.Error(err))
	}
	bp.batchTraces.reset()
}

func (bp *batchMetricProcessor) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	// First thing is convert into a different internal format
	bp.newMetricItem <- md
	return nil
}

func (bp *batchMetricProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (bp *batchMetricProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (bp *batchMetricProcessor) Shutdown(context.Context) error {
	close(bp.done)
	return nil
}

func (bp *batchMetricProcessor) startProcessingCycle() {
	bp.timer = time.NewTimer(bp.timeout)
	for {
		select {
		case md := <-bp.newMetricItem:
			bp.batchMetrics.add(md)
			if bp.batchMetrics.getItemCount() >= bp.sendBatchSize {
				bp.timer.Stop()
				bp.sendItems(statBatchSizeTriggerSend)
				bp.resetTimer()
			}
		case <-bp.timer.C:
			if bp.batchMetrics.hasData() {
				bp.sendItems(statTimeoutTriggerSend)
			}
			bp.resetTimer()
		case <-bp.done:
			if bp.batchMetrics.hasData() {
				bp.sendItems(statTimeoutTriggerSend)
			}
			return
		}
	}
}

func (bp *batchMetricProcessor) resetTimer() {
	bp.timer.Reset(bp.timeout)
}
func (bp *batchMetricProcessor) sendItems(measure *stats.Int64Measure) {
	// Add that it came from the metrics pipeline
	statsTags := []tag.Mutator{tag.Insert(processor.TagProcessorNameKey, bp.name)}
	_ = stats.RecordWithTags(context.Background(), statsTags, measure.M(1))
	_ = stats.RecordWithTags(context.Background(), statsTags, statBatchSendSize.M(int64(bp.batchMetrics.metricData.MetricCount())))

	_ = bp.metricsConsumer.ConsumeMetrics(context.Background(), bp.batchMetrics.getData())
	bp.batchMetrics.reset()
}

type batchTraces struct {
	traceData          pdata.Traces
	resourceSpansCount uint32
	spanCount          uint32
}

func newBatchTraces() *batchTraces {
	b := &batchTraces{}
	b.reset()
	return b
}

// add updates current batchTraces by adding new TraceData object
func (b *batchTraces) add(td pdata.Traces) {
	newResourceSpansCount := td.ResourceSpans().Len()
	if newResourceSpansCount == 0 {
		return
	}

	b.resourceSpansCount = b.resourceSpansCount + uint32(newResourceSpansCount)
	b.spanCount = b.spanCount + uint32(td.SpanCount())
	td.ResourceSpans().MoveAndAppendTo(b.traceData.ResourceSpans())
}

func (b *batchTraces) getTraceData() pdata.Traces {
	return b.traceData
}

func (b *batchTraces) getSpanCount() uint32 {
	return b.spanCount
}

func (b *batchTraces) hasData() bool {
	return b.traceData.ResourceSpans().Len() > 0
}

// resets the current batchTraces structure with zero values
func (b *batchTraces) reset() {
	// TODO: Use b.resourceSpansCount to preset capacity of b.traceData.ResourceSpans
	// once internal data API provides such functionality
	b.traceData = pdata.NewTraces()

	b.spanCount = 0
	b.resourceSpansCount = 0
}

type batchMetrics struct {
	metricData    data.MetricData
	resourceCount uint32
	itemCount     uint32
}

func newBatchMetrics() *batchMetrics {
	b := &batchMetrics{}
	b.reset()
	return b
}

func (bm *batchMetrics) getData() pdata.Metrics {
	return pdatautil.MetricsFromInternalMetrics(bm.metricData)
}

func (bm *batchMetrics) getItemCount() uint32 {
	return bm.itemCount
}

func (bm *batchMetrics) hasData() bool {
	return bm.metricData.ResourceMetrics().Len() > 0
}

// resets the current batchMetrics structure with zero/empty values.
func (bm *batchMetrics) reset() {
	bm.metricData = data.NewMetricData()
	bm.itemCount = 0
	bm.resourceCount = 0
}

func (bm *batchMetrics) add(pm pdata.Metrics) {
	md := pdatautil.MetricsToInternalMetrics(pm)

	newResourceCount := md.ResourceMetrics().Len()
	if newResourceCount == 0 {
		return
	}
	bm.resourceCount = bm.resourceCount + uint32(newResourceCount)
	bm.itemCount = bm.itemCount + uint32(md.MetricCount())
	md.ResourceMetrics().MoveAndAppendTo(bm.metricData.ResourceMetrics())
}
