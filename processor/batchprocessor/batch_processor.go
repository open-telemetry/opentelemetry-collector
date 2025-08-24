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
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
	"go.opentelemetry.io/collector/processor"
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
type batchProcessor[T any] struct {
	logger           *zap.Logger
	timeout          time.Duration
	sendBatchSize    int
	sendBatchMaxSize int

	// batchFunc is a factory for new batch objects corresponding
	// with the appropriate signal.
	batchFunc func() batch[T]

	shutdownC  chan struct{}
	goroutines sync.WaitGroup

	telemetry *batchProcessorTelemetry

	// batcher will be either *singletonBatcher or *multiBatcher
	batcher batcher[T]
}

// batcher is describes a *singletonBatcher or *multiBatcher.
type batcher[T any] interface {
	// start initializes background resources used by this batcher.
	start(ctx context.Context) error

	// consume incorporates a new item of data into the pending batch.
	consume(ctx context.Context, data T) error

	// currentMetadataCardinality returns the number of shards.
	currentMetadataCardinality() int
}

// shard is a single instance of the batch logic.  When metadata
// keys are in use, one of these is created per distinct combination
// of values.
type shard[T any] struct {
	// processor refers to this processor, for access to common
	// configuration.
	processor *batchProcessor[T]

	// exportCtx is a context with the metadata key-values
	// corresponding with this shard set.
	exportCtx context.Context

	// timer informs the shard send a batch.
	timer *time.Timer

	// newItem is used to receive data items from producers.
	newItem chan T

	// batch is an in-flight data item containing one of the
	// underlying data types.
	batch batch[T]
}

// batch is an interface generalizing the individual signal types.
type batch[T any] interface {
	// export the current batch
	export(ctx context.Context, req T) error

	// split returns a full request built from pending items.
	split(sendBatchMaxSize int) (sentBatchSize int, req T)

	// itemCount returns the size of the current batch
	itemCount() int

	// add item to the current batch
	add(item T)

	// sizeBytes counts the OTLP encoding size of the batch
	sizeBytes(item T) int
}

// newBatchProcessor returns a new batch processor component.
func newBatchProcessor[T any](set processor.Settings, cfg *Config, batchFunc func() batch[T]) (*batchProcessor[T], error) {
	// use lower-case, to be consistent with http/2 headers.
	mks := make([]string, len(cfg.MetadataKeys))
	for i, k := range cfg.MetadataKeys {
		mks[i] = strings.ToLower(k)
	}
	sort.Strings(mks)
	bp := &batchProcessor[T]{
		logger: set.Logger,

		sendBatchSize:    int(cfg.SendBatchSize),
		sendBatchMaxSize: int(cfg.SendBatchMaxSize),
		timeout:          cfg.Timeout,
		batchFunc:        batchFunc,
		shutdownC:        make(chan struct{}, 1),
	}
	if len(mks) == 0 {
		bp.batcher = &singleShardBatcher[T]{
			processor: bp,
			single:    bp.newShard(nil),
		}
	} else {
		bp.batcher = &multiShardBatcher[T]{
			metadataKeys:  mks,
			metadataLimit: int(cfg.MetadataCardinalityLimit),
			processor:     bp,
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
func (bp *batchProcessor[T]) newShard(md map[string][]string) *shard[T] {
	exportCtx := client.NewContext(context.Background(), client.Info{
		Metadata: client.NewMetadata(md),
	})
	b := &shard[T]{
		processor: bp,
		newItem:   make(chan T, runtime.NumCPU()),
		exportCtx: exportCtx,
		batch:     bp.batchFunc(),
	}
	return b
}

func (bp *batchProcessor[T]) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// Start is invoked during service startup.
func (bp *batchProcessor[T]) Start(ctx context.Context, _ component.Host) error {
	return bp.batcher.start(ctx)
}

// Shutdown is invoked during service shutdown.
func (bp *batchProcessor[T]) Shutdown(context.Context) error {
	close(bp.shutdownC)

	// Wait until all goroutines are done.
	bp.goroutines.Wait()

	return nil
}

func (b *shard[T]) start() {
	b.processor.goroutines.Add(1)
	go b.startLoop()
}

func (b *shard[T]) startLoop() {
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
			b.processItem(item)
		case <-timerCh:
			if b.batch.itemCount() > 0 {
				b.sendItems(triggerTimeout)
			}
			b.resetTimer()
		}
	}
}

func (b *shard[T]) processItem(item T) {
	b.batch.add(item)
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

func (b *shard[T]) hasTimer() bool {
	return b.timer != nil
}

func (b *shard[T]) stopTimer() {
	if b.hasTimer() && !b.timer.Stop() {
		<-b.timer.C
	}
}

func (b *shard[T]) resetTimer() {
	if b.hasTimer() {
		b.timer.Reset(b.processor.timeout)
	}
}

func (b *shard[T]) sendItems(trigger trigger) {
	sent, req := b.batch.split(b.processor.sendBatchMaxSize)

	bpt := b.processor.telemetry
	var bytes int
	// Check if the instrument is enabled to calculate the size of the batch in bytes.
	// See https://pkg.go.dev/go.opentelemetry.io/otel/sdk/metric/internal/x#readme-instrument-enabled
	batchSendSizeBytes := bpt.telemetryBuilder.ProcessorBatchBatchSendSizeBytes
	instr, ok := batchSendSizeBytes.(interface{ Enabled(context.Context) bool })
	if !ok || instr.Enabled(bpt.exportCtx) {
		bytes = b.batch.sizeBytes(req)
	}

	err := b.batch.export(b.exportCtx, req)
	if err != nil {
		b.processor.logger.Warn("Sender failed", zap.Error(err))
		return
	}
	bpt.record(trigger, int64(sent), int64(bytes))
}

// singleShardBatcher is used when metadataKeys is empty, to avoid the
// additional lock and map operations used in multiBatcher.
type singleShardBatcher[T any] struct {
	processor *batchProcessor[T]
	single    *shard[T]
}

func (sb *singleShardBatcher[T]) start(context.Context) error {
	sb.single.start()
	return nil
}

func (sb *singleShardBatcher[T]) consume(_ context.Context, data T) error {
	sb.single.newItem <- data
	return nil
}

func (sb *singleShardBatcher[T]) currentMetadataCardinality() int {
	return 1
}

// multiShardBatcher is used when metadataKeys is not empty.
type multiShardBatcher[T any] struct {
	// metadataKeys is the configured list of metadata keys.  When
	// empty, the `singleton` batcher is used.  When non-empty,
	// each distinct combination of metadata keys and values
	// triggers a new batcher, counted in `goroutines`.
	metadataKeys []string

	// metadataLimit is the limiting size of the batchers map.
	metadataLimit int

	processor *batchProcessor[T]
	batchers  sync.Map

	// Guards the size and the storing logic to ensure no more than limit items are stored.
	// If we are willing to allow "some" extra items than the limit this can be removed and size can be made atomic.
	lock sync.Mutex
	size int
}

func (mb *multiShardBatcher[T]) start(context.Context) error {
	return nil
}

func (mb *multiShardBatcher[T]) consume(ctx context.Context, data T) error {
	// Get each metadata key value, form the corresponding
	// attribute set for use as a map lookup key.
	info := client.FromContext(ctx)
	md := map[string][]string{}
	var attrs []attribute.KeyValue
	for _, k := range mb.metadataKeys {
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
		if mb.metadataLimit != 0 && mb.size >= mb.metadataLimit {
			mb.lock.Unlock()
			return errTooManyBatchers
		}

		// aset.ToSlice() returns the sorted, deduplicated,
		// and name-lowercased list of attributes.
		var loaded bool
		b, loaded = mb.batchers.LoadOrStore(aset, mb.processor.newShard(md))
		if !loaded {
			// Start the goroutine only if we added the object to the map, otherwise is already started.
			b.(*shard[T]).start()
			mb.size++
		}
		mb.lock.Unlock()
	}
	b.(*shard[T]).newItem <- data
	return nil
}

func (mb *multiShardBatcher[T]) currentMetadataCardinality() int {
	mb.lock.Lock()
	defer mb.lock.Unlock()
	return mb.size
}

type tracesBatchProcessor struct {
	*batchProcessor[ptrace.Traces]
}

// newTracesBatchProcessor creates a new batch processor that batches traces by size or with timeout
func newTracesBatchProcessor(set processor.Settings, next consumer.Traces, cfg *Config) (processor.Traces, error) {
	bp, err := newBatchProcessor(set, cfg, func() batch[ptrace.Traces] { return newBatchTraces(next) })
	if err != nil {
		return nil, err
	}
	return &tracesBatchProcessor{batchProcessor: bp}, nil
}

func (t *tracesBatchProcessor) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	pref.RefTraces(td)
	return t.batcher.consume(ctx, td)
}

type metricsBatchProcessor struct {
	*batchProcessor[pmetric.Metrics]
}

// newMetricsBatchProcessor creates a new batch processor that batches metrics by size or with timeout
func newMetricsBatchProcessor(set processor.Settings, next consumer.Metrics, cfg *Config) (processor.Metrics, error) {
	bp, err := newBatchProcessor(set, cfg, func() batch[pmetric.Metrics] { return newMetricsBatch(next) })
	if err != nil {
		return nil, err
	}
	return &metricsBatchProcessor{batchProcessor: bp}, nil
}

// ConsumeMetrics implements processor.Metrics
func (m *metricsBatchProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	pref.RefMetrics(md)
	return m.batcher.consume(ctx, md)
}

type logsBatchProcessor struct {
	*batchProcessor[plog.Logs]
}

// newLogsBatchProcessor creates a new batch processor that batches logs by size or with timeout
func newLogsBatchProcessor(set processor.Settings, next consumer.Logs, cfg *Config) (processor.Logs, error) {
	bp, err := newBatchProcessor(set, cfg, func() batch[plog.Logs] { return newBatchLogs(next) })
	if err != nil {
		return nil, err
	}
	return &logsBatchProcessor{batchProcessor: bp}, nil
}

// ConsumeLogs implements processor.Logs
func (l *logsBatchProcessor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	pref.RefLogs(ld)
	return l.batcher.consume(ctx, ld)
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
func (bt *batchTraces) add(td ptrace.Traces) {
	defer pref.UnrefTraces(td)
	newSpanCount := td.SpanCount()
	if newSpanCount == 0 {
		return
	}

	bt.spanCount += newSpanCount
	td.ResourceSpans().MoveAndAppendTo(bt.traceData.ResourceSpans())
}

func (bt *batchTraces) sizeBytes(td ptrace.Traces) int {
	return bt.sizer.TracesSize(td)
}

func (bt *batchTraces) export(ctx context.Context, td ptrace.Traces) error {
	return bt.nextConsumer.ConsumeTraces(ctx, td)
}

func (bt *batchTraces) split(sendBatchMaxSize int) (int, ptrace.Traces) {
	var td ptrace.Traces
	var sent int
	if sendBatchMaxSize > 0 && bt.itemCount() > sendBatchMaxSize {
		td = splitTraces(sendBatchMaxSize, bt.traceData)
		bt.spanCount -= sendBatchMaxSize
		sent = sendBatchMaxSize
	} else {
		td = bt.traceData
		sent = bt.spanCount
		bt.traceData = ptrace.NewTraces()
		bt.spanCount = 0
	}
	return sent, td
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

func newMetricsBatch(nextConsumer consumer.Metrics) *batchMetrics {
	return &batchMetrics{nextConsumer: nextConsumer, metricData: pmetric.NewMetrics(), sizer: &pmetric.ProtoMarshaler{}}
}

func (bm *batchMetrics) sizeBytes(md pmetric.Metrics) int {
	return bm.sizer.MetricsSize(md)
}

func (bm *batchMetrics) export(ctx context.Context, md pmetric.Metrics) error {
	return bm.nextConsumer.ConsumeMetrics(ctx, md)
}

func (bm *batchMetrics) split(sendBatchMaxSize int) (int, pmetric.Metrics) {
	var md pmetric.Metrics
	var sent int
	if sendBatchMaxSize > 0 && bm.dataPointCount > sendBatchMaxSize {
		md = splitMetrics(sendBatchMaxSize, bm.metricData)
		bm.dataPointCount -= sendBatchMaxSize
		sent = sendBatchMaxSize
	} else {
		md = bm.metricData
		sent = bm.dataPointCount
		bm.metricData = pmetric.NewMetrics()
		bm.dataPointCount = 0
	}

	return sent, md
}

func (bm *batchMetrics) itemCount() int {
	return bm.dataPointCount
}

func (bm *batchMetrics) add(md pmetric.Metrics) {
	defer pref.UnrefMetrics(md)
	newDataPointCount := md.DataPointCount()
	if newDataPointCount == 0 {
		return
	}
	bm.dataPointCount += newDataPointCount
	md.ResourceMetrics().MoveAndAppendTo(bm.metricData.ResourceMetrics())
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

func (bl *batchLogs) sizeBytes(ld plog.Logs) int {
	return bl.sizer.LogsSize(ld)
}

func (bl *batchLogs) export(ctx context.Context, ld plog.Logs) error {
	return bl.nextConsumer.ConsumeLogs(ctx, ld)
}

func (bl *batchLogs) split(sendBatchMaxSize int) (int, plog.Logs) {
	var ld plog.Logs
	var sent int

	if sendBatchMaxSize > 0 && bl.logCount > sendBatchMaxSize {
		ld = splitLogs(sendBatchMaxSize, bl.logData)
		bl.logCount -= sendBatchMaxSize
		sent = sendBatchMaxSize
	} else {
		ld = bl.logData
		sent = bl.logCount
		bl.logData = plog.NewLogs()
		bl.logCount = 0
	}
	return sent, ld
}

func (bl *batchLogs) itemCount() int {
	return bl.logCount
}

func (bl *batchLogs) add(ld plog.Logs) {
	defer pref.UnrefLogs(ld)
	newLogsCount := ld.LogRecordCount()
	if newLogsCount == 0 {
		return
	}
	bl.logCount += newLogsCount
	ld.ResourceLogs().MoveAndAppendTo(bl.logData.ResourceLogs())
}
