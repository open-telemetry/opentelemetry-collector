// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor/internal/metadata"
)

// errTooManyBatchers is returned when the MetadataCardinalityLimit has been reached.
var errTooManyBatchers = consumererror.NewPermanent(errors.New("too many batcher metadata-value combinations"))

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
	timeout          time.Duration
	sendBatchSize    int
	sendBatchMaxSize int

	// batchFunc is a factory for new batch objects corresponding
	// with the appropriate signal.
	batchFunc func() batch

	// metadataKeys is the configured list of metadata keys.  When
	// empty, the `singleton` batcher is used.  When non-empty,
	// each distinct combination of metadata keys and values
	// triggers a new batcher, counted in `goroutines`.
	metadataKeys []string

	// metadataLimit is the limiting size of the batchers map.
	metadataLimit int

	shutdownC  chan struct{}
	goroutines sync.WaitGroup

	telemetry *batchProcessorTelemetry

	// batcher will be either *singletonBatcher or *multiBatcher
	batcher batcher

	// tracer is the configured tracer
	tracer trace.Tracer
}

// batcher is describes a *singletonBatcher or *multiBatcher.
type batcher interface {
	// start initializes background resources used by this batcher.
	start(ctx context.Context) error

	// consume incorporates a new item of data into the pending batch.
	consume(ctx context.Context, data any) error

	// currentMetadataCardinality returns the number of shards.
	currentMetadataCardinality() int
}

// shard is a single instance of the batch logic.  When metadata
// keys are in use, one of these is created per distinct combination
// of values.
type shard struct {
	// processor refers to this processor, for access to common
	// configuration.
	processor *batchProcessor

	// exportCtx is a context with the metadata key-values
	// corresponding with this shard set.
	exportCtx context.Context

	// timer informs the shard send a batch.
	timer *time.Timer

	// newItem is used to receive data items from producers.
	newItem chan dataItem

	// batch is an in-flight data item containing one of the
	// underlying data types.
	batch batch

	// pending describes the contributors to the current batch.
	pending []pendingItem
}

// pendingItem is stored parallel to a pending batch and records how
// many items the waiter submitted, used to match trace contexts with
// batches.  It is also used inside sendItems() to represent a
// partially complete batch.
type pendingItem struct {
	parentCtx context.Context
	numItems  int
}

// dataItem is exchanged between the waiter and the batching process
// includes the pendingItem and its data.
type dataItem struct {
	data      any
	parentCtx context.Context
}

// batch is an interface generalizing the individual signal types.
type batch interface {
	// export the current batch
	export(ctx context.Context, req any) error

	// splitBatch returns a full request built from pending items.
	splitBatch(ctx context.Context, sendBatchMaxSize int) (sentBatchSize int, req any)

	// itemCount returns the size of the current batch
	itemCount() int

	// add item to the current batch
	add(item any) int

	// sizeBytes counts the OTLP encoding size of the batch
	sizeBytes(item any) int
}

var _ consumer.Traces = (*batchProcessor)(nil)
var _ consumer.Metrics = (*batchProcessor)(nil)
var _ consumer.Logs = (*batchProcessor)(nil)

// newBatchProcessor returns a new batch processor component.
func newBatchProcessor(set processor.Settings, cfg *Config, batchFunc func() batch) (*batchProcessor, error) {
	// use lower-case, to be consistent with http/2 headers.
	mks := make([]string, len(cfg.MetadataKeys))
	for i, k := range cfg.MetadataKeys {
		mks[i] = strings.ToLower(k)
	}
	sort.Strings(mks)
	bp := &batchProcessor{
		logger: set.Logger,

		sendBatchSize:    int(cfg.SendBatchSize),
		sendBatchMaxSize: int(cfg.SendBatchMaxSize),
		timeout:          cfg.Timeout,
		batchFunc:        batchFunc,
		shutdownC:        make(chan struct{}, 1),
		metadataKeys:     mks,
		metadataLimit:    int(cfg.MetadataCardinalityLimit),
		tracer:           metadata.Tracer(set.TelemetrySettings),
	}
	if len(bp.metadataKeys) == 0 {
		bp.batcher = &singleShardBatcher{
			processor: bp,
			single:    nil, // created in start
		}
	} else {
		bp.batcher = &multiShardBatcher{
			processor: bp,
		}
	}

	bpt, err := newBatchProcessorTelemetry(set, bp.batcher.currentMetadataCardinality)
	if err != nil {
		return nil, fmt.Errorf("error creating batch processor telemetry: %w", err)
	}
	bp.telemetry = bpt

	return bp, nil
}

// newShard gets or creates a batcher corresponding with attrs.
func (bp *batchProcessor) newShard(md map[string][]string) *shard {
	exportCtx := client.NewContext(context.Background(), client.Info{
		Metadata: client.NewMetadata(md),
	})
	b := &shard{
		processor: bp,
		newItem:   make(chan dataItem, runtime.NumCPU()),
		exportCtx: exportCtx,
		batch:     bp.batchFunc(),
	}
	return b
}

func (bp *batchProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// Start is invoked during service startup.
func (bp *batchProcessor) Start(ctx context.Context, _ component.Host) error {
	return bp.batcher.start(ctx)
}

// Shutdown is invoked during service shutdown.
func (bp *batchProcessor) Shutdown(context.Context) error {
	close(bp.shutdownC)

	// Wait until all goroutines are done.
	bp.goroutines.Wait()
	return nil
}

func (b *shard) start() {
	b.processor.goroutines.Add(1)
	go b.startLoop()
}

func (b *shard) startLoop() {
	defer b.processor.goroutines.Done()

	// timerCh ensures we only block when there is a
	// timer, since <- from a nil channel is blocking.
	var timerCh <-chan time.Time
	if b.processor.timeout != 0 && b.processor.sendBatchSize != 0 {
		b.timer = time.NewTimer(b.processor.timeout)
		timerCh = b.timer.C
	}
	for {
		select {
		case <-b.processor.shutdownC:
		DONE:
			for {
				select {
				case item := <-b.newItem:
					b.processItem(item)
				default:
					break DONE
				}
			}
			// This is the close of the channel
			if b.batch.itemCount() > 0 {
				// TODO: Set a timeout on sendTraces or
				// make it cancellable using the context that Shutdown gets as a parameter
				b.sendItems(triggerTimeout)
			}
			return
		case item := <-b.newItem:
			if item.data == nil {
				continue
			}
			// Important invariant. processItem() must return with the pending
			// number of items less than the minimum batch size.
			b.processItem(item)
		case <-timerCh:
			if b.batch.itemCount() > 0 {
				b.sendItems(triggerTimeout)
			}
			b.resetTimer()
		}
	}
}

func (b *shard) processItem(item dataItem) {
	totalItems := b.batch.add(item.data)

	b.pending = append(b.pending, pendingItem{
		parentCtx: item.parentCtx,
		numItems:  totalItems,
	})

	// The call to flushItems() is necessary to maintain the invariant that
	// after this call returns, the pending data is less than a full batch.
	b.flushItems()
}

func (b *shard) flushItems() {
	sent := false
	for b.batch.itemCount() > 0 && (!b.hasTimer() || b.batch.itemCount() >= b.processor.sendBatchSize) {
		sent = true
		b.sendItems(triggerBatchSize)
	}

	if sent {
		b.stopTimer()
		b.resetTimer()
	}
}

func (b *shard) hasTimer() bool {
	return b.timer != nil
}

func (b *shard) stopTimer() {
	if b.hasTimer() && !b.timer.Stop() {
		<-b.timer.C
	}
}

func (b *shard) resetTimer() {
	if b.hasTimer() {
		b.timer.Reset(b.processor.timeout)
	}
}

func (b *shard) sendItems(trigger trigger) {
	// Note because of the invariant stated for processItems, we know the
	// number of current waiters exceeds the batch by at most one entry.
	// Therefore, we can enumerate the possibilities.
	//
	// 1. len(b.pending) == 1 where the item count of element[0] is <= batch size
	// 2. len(b.pending) == 1 where the item count of element[0] is > batch size
	// 3. len(b.pending) == N where N>1 and the item count in elements[0:N-1] is < batch size
	//
	// Importantly, in all cases, the batch will include a portion
	// of data from all contexts in the pending slice.  In case 2
	// there is always a single residual pendingItem.  In case 3 there
	// may or may not be a single residential pendingItem.

	pendingSize := b.batch.itemCount()
	sent, req := b.batch.splitBatch(b.exportCtx, b.processor.sendBatchMaxSize)

	var rootCtx context.Context

	// If the portion being sent belongs to the first item in the pending list,
	// then there is a single context we can use.
	if sent <= b.pending[0].numItems {
		rootCtx = b.pending[0].parentCtx
	} else {
		rootCtx = b.exportCtx
	}
	ctx, parent := b.processor.tracer.Start(rootCtx, "batch_processor/export")
	defer parent.End()

	// Note: linking spans in both directions.  This could be
	// inferred by the trace backend, but this adds helpful
	// information in cases where sampling may break links.  See
	// https://github.com/open-telemetry/opentelemetry-specification/issues/1877
	// Note that there is a possibility that the span has already
	// ended (if EarlyReturn or context canceled), in which case
	// this becomes a no-op.
	parentSpanContext := parent.SpanContext()
	for _, pending := range b.pending {
		span := trace.SpanFromContext(pending.parentCtx)
		span.AddLink(trace.Link{SpanContext: parentSpanContext})
		parent.AddLink(trace.Link{SpanContext: span.SpanContext()})
	}

	err := b.batch.export(ctx, req)

	// Remember the last pending parent context, we may need it.
	finalParent := b.pending[len(b.pending)-1].parentCtx
	// Clear the array to permit GC of finished contexts while
	// re-using the slice instead of a new allocation.
	for i := range b.pending {
		b.pending[i] = pendingItem{}
	}
	// There is either one or zero items left in the pending slice,
	// according to the invariant discussed above.
	if pendingSize == sent {
		// The batch is fully sent, pending slice is clear.
		b.pending = b.pending[:0]
	} else {
		// The batch is fully sent, pending slice keeps one item.
		b.pending = b.pending[:1]
		b.pending[0].parentCtx = finalParent
		b.pending[0].numItems = pendingSize - sent
	}

	if err != nil {
		b.processor.logger.Warn("Sender failed", zap.Error(err))
		// TODO: it seems incorrect not to call telemetry.record()
		// for errors.  Yes?
		return
	}

	var bytes int
	if b.processor.telemetry.detailed {
		bytes = b.batch.sizeBytes(req)
	}
	b.processor.telemetry.record(trigger, int64(sent), int64(bytes))
}

func (b *shard) consumeBatch(ctx context.Context, data any) error {
	var itemCount int
	switch telem := data.(type) {
	case ptrace.Traces:
		itemCount = telem.SpanCount()
	case pmetric.Metrics:
		itemCount = telem.DataPointCount()
	case plog.Logs:
		itemCount = telem.LogRecordCount()
	}

	if itemCount == 0 {
		return nil
	}

	item := dataItem{
		data:      data,
		parentCtx: ctx,
	}

	b.newItem <- item
	return nil
}

// singleShardBatcher is used when metadataKeys is empty, to avoid the
// additional lock and map operations used in multiBatcher.
type singleShardBatcher struct {
	processor *batchProcessor
	single    *shard
}

func (sb *singleShardBatcher) start(context.Context) error {
	sb.single = sb.processor.newShard(nil)
	sb.single.start()
	return nil
}

func (sb *singleShardBatcher) consume(ctx context.Context, data any) error {
	return sb.single.consumeBatch(ctx, data)
}

func (sb *singleShardBatcher) currentMetadataCardinality() int {
	return 1
}

// multiBatcher is used when metadataKeys is not empty.
type multiShardBatcher struct {
	processor *batchProcessor
	batchers  sync.Map

	// Guards the size and the storing logic to ensure no more than limit items are stored.
	// If we are willing to allow "some" extra items than the limit this can be removed and size can be made atomic.
	lock sync.Mutex
	size int
}

func (mb *multiShardBatcher) start(context.Context) error {
	return nil
}

func (mb *multiShardBatcher) consume(ctx context.Context, data any) error {
	// Get each metadata key value, form the corresponding
	// attribute set for use as a map lookup key.
	info := client.FromContext(ctx)
	md := map[string][]string{}
	var attrs []attribute.KeyValue
	for _, k := range mb.processor.metadataKeys {
		// Lookup the value in the incoming metadata, copy it
		// into the outgoing metadata, and create a unique
		// value for the attributeSet.
		vs := info.Metadata.Get(k)
		md[k] = vs
		if len(vs) == 1 {
			attrs = append(attrs, attribute.String(k, vs[0]))
		} else {
			attrs = append(attrs, attribute.StringSlice(k, vs))
		}
	}
	aset := attribute.NewSet(attrs...)

	b, ok := mb.batchers.Load(aset)
	if !ok {
		mb.lock.Lock()
		if mb.processor.metadataLimit != 0 && mb.size >= mb.processor.metadataLimit {
			mb.lock.Unlock()
			return errTooManyBatchers
		}

		// aset.ToSlice() returns the sorted, deduplicated,
		// and name-lowercased list of attributes.
		var loaded bool
		b, loaded = mb.batchers.LoadOrStore(aset, mb.processor.newShard(md))
		if !loaded {
			// Start the goroutine only if we added the object to the map, otherwise is already started.
			b.(*shard).start()
			mb.size++
		}
		mb.lock.Unlock()
	}
	return b.(*shard).consumeBatch(ctx, data)
}

func (mb *multiShardBatcher) currentMetadataCardinality() int {
	mb.lock.Lock()
	defer mb.lock.Unlock()
	return mb.size
}

// ConsumeTraces implements processor.Traces
func (bp *batchProcessor) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return bp.batcher.consume(ctx, td)
}

// ConsumeMetrics implements processor.Metrics
func (bp *batchProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return bp.batcher.consume(ctx, md)
}

// ConsumeLogs implements processor.Logs
func (bp *batchProcessor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	return bp.batcher.consume(ctx, ld)
}

// newBatchTraces creates a new batch processor that batches traces by size or with timeout
func newBatchTracesProcessor(set processor.Settings, next consumer.Traces, cfg *Config) (*batchProcessor, error) {
	return newBatchProcessor(set, cfg, func() batch { return newBatchTraces(next) })
}

// newBatchMetricsProcessor creates a new batch processor that batches metrics by size or with timeout
func newBatchMetricsProcessor(set processor.Settings, next consumer.Metrics, cfg *Config) (*batchProcessor, error) {
	return newBatchProcessor(set, cfg, func() batch { return newBatchMetrics(next) })
}

// newBatchLogsProcessor creates a new batch processor that batches logs by size or with timeout
func newBatchLogsProcessor(set processor.Settings, next consumer.Logs, cfg *Config) (*batchProcessor, error) {
	return newBatchProcessor(set, cfg, func() batch { return newBatchLogs(next) })
}

type batchTraces struct {
	nextConsumer consumer.Traces
	traceData    ptrace.Traces
	spanCount    int
	sizer        ptrace.Sizer
}

func newBatchTraces(nextConsumer consumer.Traces) *batchTraces {
	return &batchTraces{nextConsumer: nextConsumer, traceData: ptrace.NewTraces(), sizer: &ptrace.ProtoMarshaler{}}
}

// add updates current batchTraces by adding new TraceData object
func (bt *batchTraces) add(item any) int {
	td := item.(ptrace.Traces)
	newSpanCount := td.SpanCount()
	if newSpanCount != 0 {
		bt.spanCount += newSpanCount
		td.ResourceSpans().MoveAndAppendTo(bt.traceData.ResourceSpans())
	}
	return newSpanCount
}

func (bt *batchTraces) sizeBytes(data any) int {
	return bt.sizer.TracesSize(data.(ptrace.Traces))
}

func (bt *batchTraces) export(ctx context.Context, req any) error {
	td := req.(ptrace.Traces)
	return bt.nextConsumer.ConsumeTraces(ctx, td)
}

func (bt *batchTraces) splitBatch(_ context.Context, sendBatchMaxSize int) (int, any) {
	var req ptrace.Traces
	var sent int
	if sendBatchMaxSize > 0 && bt.itemCount() > sendBatchMaxSize {
		req = splitTraces(sendBatchMaxSize, bt.traceData)
		bt.spanCount -= sendBatchMaxSize
		sent = sendBatchMaxSize
	} else {
		req = bt.traceData
		sent = bt.spanCount
		bt.traceData = ptrace.NewTraces()
		bt.spanCount = 0
	}
	return sent, req
}

func (bt *batchTraces) itemCount() int {
	return bt.spanCount
}

type batchMetrics struct {
	nextConsumer   consumer.Metrics
	metricData     pmetric.Metrics
	dataPointCount int
	sizer          pmetric.Sizer
}

func newBatchMetrics(nextConsumer consumer.Metrics) *batchMetrics {
	return &batchMetrics{nextConsumer: nextConsumer, metricData: pmetric.NewMetrics(), sizer: &pmetric.ProtoMarshaler{}}
}

func (bm *batchMetrics) sizeBytes(data any) int {
	return bm.sizer.MetricsSize(data.(pmetric.Metrics))
}

func (bm *batchMetrics) export(ctx context.Context, req any) error {
	md := req.(pmetric.Metrics)
	return bm.nextConsumer.ConsumeMetrics(ctx, md)
}

func (bm *batchMetrics) splitBatch(_ context.Context, sendBatchMaxSize int) (int, any) {
	var req pmetric.Metrics
	var sent int
	if sendBatchMaxSize > 0 && bm.dataPointCount > sendBatchMaxSize {
		req = splitMetrics(sendBatchMaxSize, bm.metricData)
		bm.dataPointCount -= sendBatchMaxSize
		sent = sendBatchMaxSize
	} else {
		req = bm.metricData
		sent = bm.dataPointCount
		bm.metricData = pmetric.NewMetrics()
		bm.dataPointCount = 0
	}

	return sent, req
}

func (bm *batchMetrics) itemCount() int {
	return bm.dataPointCount
}

func (bm *batchMetrics) add(item any) int {
	md := item.(pmetric.Metrics)

	newDataPointCount := md.DataPointCount()
	if newDataPointCount != 0 {
		bm.dataPointCount += newDataPointCount
		md.ResourceMetrics().MoveAndAppendTo(bm.metricData.ResourceMetrics())
	}
	return newDataPointCount
}

type batchLogs struct {
	nextConsumer consumer.Logs
	logData      plog.Logs
	logCount     int
	sizer        plog.Sizer
}

func newBatchLogs(nextConsumer consumer.Logs) *batchLogs {
	return &batchLogs{nextConsumer: nextConsumer, logData: plog.NewLogs(), sizer: &plog.ProtoMarshaler{}}
}

func (bl *batchLogs) sizeBytes(data any) int {
	return bl.sizer.LogsSize(data.(plog.Logs))
}

func (bl *batchLogs) export(ctx context.Context, req any) error {
	ld := req.(plog.Logs)
	return bl.nextConsumer.ConsumeLogs(ctx, ld)
}

func (bl *batchLogs) splitBatch(_ context.Context, sendBatchMaxSize int) (int, any) {
	var req plog.Logs
	var sent int

	if sendBatchMaxSize > 0 && bl.logCount > sendBatchMaxSize {
		req = splitLogs(sendBatchMaxSize, bl.logData)
		bl.logCount -= sendBatchMaxSize
		sent = sendBatchMaxSize
	} else {
		req = bl.logData
		sent = bl.logCount
		bl.logData = plog.NewLogs()
		bl.logCount = 0
	}
	return sent, req
}

func (bl *batchLogs) itemCount() int {
	return bl.logCount
}

func (bl *batchLogs) add(item any) int {
	ld := item.(plog.Logs)

	newLogsCount := ld.LogRecordCount()
	if newLogsCount != 0 {
		bl.logCount += newLogsCount
		ld.ResourceLogs().MoveAndAppendTo(bl.logData.ResourceLogs())
	}
	return newLogsCount
}
