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
	"go.opentelemetry.io/collector/processor"
)

// batch_processor is a component that accepts spans, places them into batches and sends downstream.
//
// batch_processor implements consumer.TraceConsumer
//
// Batches are sent out with any of the following conditions:
// - batch size reaches cfg.SendBatchSize
// - cfg.Timeout is elapsed since the timestamp when the previous batch was sent out.
type batchProcessor struct {
	sender consumer.TraceConsumer
	name   string
	logger *zap.Logger

	sendBatchSize uint32
	timeout       time.Duration

	batch   *batch
	timer   *time.Timer
	newItem chan pdata.Traces
	done    chan struct{}
}

var _ consumer.TraceConsumer = (*batchProcessor)(nil)

// newBatchProcessor creates a new batch processor that batch traces by size or with timeout
func newBatchProcessor(params component.ProcessorCreateParams, sender consumer.TraceConsumer, cfg *Config) *batchProcessor {
	p := &batchProcessor{
		name:   cfg.Name(),
		sender: sender,
		logger: params.Logger,

		sendBatchSize: cfg.SendBatchSize,
		timeout:       cfg.Timeout,

		batch:   newBatch(),
		newItem: make(chan pdata.Traces, 1),
		done:    make(chan struct{}),
	}

	go p.startProcessingCycle()

	return p
}

// ConsumeTraces implements batcher as a SpanProcessor and takes the provided spans and adds them to
// batches
func (bp *batchProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	bp.newItem <- td
	return nil
}

func (bp *batchProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (bp *batchProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (bp *batchProcessor) Shutdown(context.Context) error {
	close(bp.done)
	return nil
}

func (bp *batchProcessor) startProcessingCycle() {
	bp.timer = time.NewTimer(bp.timeout)
	for {
		select {
		case td := <-bp.newItem:
			bp.batch.add(td)

			if bp.batch.getSpanCount() >= bp.sendBatchSize {
				bp.timer.Stop()
				bp.sendItems(statBatchSizeTriggerSend)
				bp.resetTimer()
			}
		case <-bp.timer.C:
			if bp.batch.hasData() {
				bp.sendItems(statTimeoutTriggerSend)
			}
			bp.resetTimer()
		case <-bp.done:
			if bp.batch.hasData() {
				// TODO: Set a timeout on sendTraces or
				// make it cancellable using the context that Shutdown gets as a parameter
				bp.sendItems(statTimeoutTriggerSend)
			}
			return
		}
	}
}

func (bp *batchProcessor) resetTimer() {
	bp.timer.Reset(bp.timeout)
}

func (bp *batchProcessor) sendItems(measure *stats.Int64Measure) {
	statsTags := []tag.Mutator{tag.Insert(processor.TagProcessorNameKey, bp.name)}
	_ = stats.RecordWithTags(context.Background(), statsTags, measure.M(1))

	_ = bp.sender.ConsumeTraces(context.Background(), bp.batch.getTraceData())
	bp.batch.reset()
}

type batch struct {
	traceData          pdata.Traces
	resourceSpansCount uint32
	spanCount          uint32
}

func newBatch() *batch {
	b := &batch{}
	b.reset()
	return b
}

// add updates current batch by adding new TraceData object
func (b *batch) add(td pdata.Traces) {
	newResourceSpansCount := td.ResourceSpans().Len()
	if newResourceSpansCount == 0 {
		return
	}

	b.resourceSpansCount = b.resourceSpansCount + uint32(newResourceSpansCount)
	b.spanCount = b.spanCount + uint32(td.SpanCount())
	td.ResourceSpans().MoveAndAppendTo(b.traceData.ResourceSpans())
}

func (b *batch) getTraceData() pdata.Traces {
	return b.traceData
}

func (b *batch) getSpanCount() uint32 {
	return b.spanCount
}

func (b *batch) hasData() bool {
	return b.traceData.ResourceSpans().Len() > 0
}

// resets the current batch structure with zero values
func (b *batch) reset() {

	// TODO: Use b.resourceSpansCount to preset capacity of b.traceData.ResourceSpans
	// once internal data API provides such functionality
	b.traceData = pdata.NewTraces()

	b.spanCount = 0
	b.resourceSpansCount = 0
}
