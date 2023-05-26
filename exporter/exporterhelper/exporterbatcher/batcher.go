// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"context"
	"errors"
	"fmt"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// errTooManyBatchers is returned when the MetadataCardinalityLimit has been reached.
var errTooManyBatchers = consumererror.NewPermanent(errors.New("too many batch identifier combinations"))

// batch_processor is a component that accepts spans and metrics, places them
// into batches and sends downstream.
//
// batch_processor implements consumer.Traces and consumer.Metrics
//
// Batches are sent out with any of the following conditions:
// - batch size reaches cfg.SendBatchSize
// - cfg.Timeout is elapsed since the timestamp when the previous batch was sent out.
type batchConsumer struct {
	logger *zap.Logger
	cfg    Config
	next   internal.RequestSender

	// identifyBatchFunc is a function that returns a key
	// identifying the batch to which the item should be added.
	identifyBatchFunc any

	shutdownC  chan struct{}
	goroutines sync.WaitGroup

	telemetry *batchTelemetry

	//  batcher will be either *singletonBatcher or *multiBatcher
	batcher batcher
}

type batcher interface {
	shard(id string) (*shard, error)
	shardsCount() int
}

type batch struct {
	Batch
}

func (b *batch) Count() int {
	if b.Batch != nil {
		return b.Batch.Count()
	}
	return 0
}

func (b *batch) Size() int {
	if b.Batch != nil {
		return b.Batch.Size()
	}
	return 0
}

// shard is a single instance of the batch logic.  When metadata
// keys are in use, one of these is created per distinct combination
// of values.
type shard struct {
	// consumer refers to this consumer, for access to common
	// configuration.
	consumer *batchConsumer

	// exportCtx is a context with the metadata key-values
	// corresponding with this shard set.
	exportCtx context.Context

	// timer informs the shard send a batch.
	timer *time.Timer

	// newItem is used to receive batches from producers.
	newItem chan Batch

	// batch is an in-flight data item containing one of the
	// underlying data types.
	batch batch

	lock sync.Mutex

	minSizeBytes int
	maxSizeBytes int
	minSizeItems int
	maxSizeItems int
}

var _ consumer.Traces = (*tracesBatchConsumer)(nil)
var _ consumer.Metrics = (*metricsBatchConsumer)(nil)
var _ consumer.Logs = (*logsBatchConsumer)(nil)

// newBatchConsumer returns a new batch consumer component.
func newBatchConsumer(set exporter.CreateSettings, cfg Config, identifyBatchFunc any, next internal.RequestSender,
	useOtel bool) (*batchConsumer, error) {
	bp := &batchConsumer{
		logger:            set.Logger,
		cfg:               cfg,
		identifyBatchFunc: identifyBatchFunc,
		next:              next,
		shutdownC:         make(chan struct{}, 1),
	}
	if identifyBatchFunc == nil {
		bp.batcher = &singleShardBatcher{batcher: bp.newShard()}
	} else {
		bp.batcher = &multiShardBatcher{batchConsumer: bp}
	}

	bpt, err := newBatchTelemetry(set, bp.batcher.shardsCount, useOtel)
	if err != nil {
		return nil, fmt.Errorf("error creating batch consumer telemetry: %w", err)
	}
	bp.telemetry = bpt

	return bp, nil
}

// newShard gets or creates a batcher corresponding with attrs.
func (bc *batchConsumer) newShard() *shard {
	b := &shard{
		consumer:     bc,
		newItem:      make(chan Batch, runtime.NumCPU()),
		minSizeBytes: bc.cfg.MinSizeMib * 1024 * 1024,
		maxSizeBytes: bc.cfg.MaxSizeMib * 1024 * 1024,
		minSizeItems: bc.cfg.MinSizeItems,
		maxSizeItems: bc.cfg.MaxSizeItems,
	}
	go b.start()
	return b
}

// Capabilities implements consumer.Traces. batchConsumer itself does not mutate pdata,
// but Batch implementations may do so. Make sure this is reflected in the Capabilities of the exporter using it.
func (bp *batchConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Start is invoked during service startup.
func (bp *batchConsumer) Start(context.Context, component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (bp *batchConsumer) Shutdown(context.Context) error {
	close(bp.shutdownC)

	// Wait until all goroutines are done.
	bp.goroutines.Wait()
	return nil
}

func (s *shard) start() {
	if s.consumer.cfg.Timeout == 0 {
		return
	}

	s.timer = time.NewTimer(s.consumer.cfg.Timeout)

	s.consumer.goroutines.Add(1)
	defer s.consumer.goroutines.Done()

	for {
		select {
		case <-s.consumer.shutdownC:
			return
		case <-s.timer.C:
			if s.batch.Batch != nil {
				s.lock.Lock()
				s.export(triggerTimeout)
				s.lock.Unlock()
			}
			s.timer.Reset(s.consumer.cfg.Timeout)
		}
	}
}

// accept adds an item to the batch. If the batch cannot be accepted due to its big size, it returns false.
func (s *shard) accept(el Batch) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.maxSizeBytes > 0 && el.Size() > s.maxSizeBytes || s.maxSizeItems > 0 && el.Count() > s.maxSizeItems {
		return false
	}

	if s.maxSizeBytes > 0 && s.batch.Size()+el.Size() > s.maxSizeBytes ||
		s.maxSizeItems > 0 && s.batch.Count()+el.Count() > s.maxSizeItems {
		s.export(triggerBatchSize)
	}
	s.appendBatch(el)

	if s.minSizeBytes > 0 && s.batch.Size() > s.minSizeBytes || s.minSizeItems > 0 && s.batch.Count() > s.minSizeItems {
		s.export(triggerBatchSize)
	}
	return true
}

func (s *shard) export(trigger trigger) {
	if s.batch.Batch == nil {
		return
	}

	b := s.batch.Batch
	itemsToSend := b.Count()
	bytesToSend := b.Size()
	s.batch = batch{}
	err := s.consumer.next.Send(b.(internal.Request))
	if err != nil {
		s.consumer.logger.Warn("Failed to send batch to downstream consumer", zap.Error(err))
	} else {
		s.consumer.telemetry.record(trigger, int64(itemsToSend), int64(bytesToSend))
	}
}

func (s *shard) appendBatch(other Batch) {
	if s.batch.Batch == nil {
		s.batch.Batch = other
		return
	}
	s.batch.Batch = s.batch.Batch.Append(other)
}

func (b *shard) hasTimer() bool {
	return b.timer != nil
}

func (b *shard) stopTimer() {
	if b.hasTimer() && !b.timer.Stop() {
		<-b.timer.C
	}
}

// singleShardBatcher is used when metadataKeys is empty, to avoid the
// additional lock and map operations used in multiBatcher.
type singleShardBatcher struct {
	batcher *shard
}

func (sb *singleShardBatcher) shard(_ string) (*shard, error) {
	return sb.batcher, nil
}

func (sb *singleShardBatcher) shardsCount() int {
	return 1
}

// multiBatcher is used when metadataKeys is not empty.
type multiShardBatcher struct {
	*batchConsumer
	batchers sync.Map

	// Guards the size and the storing logic to ensure no more than limit items are stored.
	// If we are willing to allow "some" extra items than the limit this can be removed and size can be made atomic.
	lock sync.Mutex
	size int
}

func (mb *multiShardBatcher) shard(id string) (*shard, error) {
	s, ok := mb.batchers.Load(id)
	if ok {
		return s.(*shard), nil
	}

	mb.lock.Lock()
	defer mb.lock.Unlock()

	if mb.cfg.BatchersLimit != 0 && mb.size >= mb.cfg.BatchersLimit {
		return nil, errTooManyBatchers
	}

	s, loaded := mb.batchers.LoadOrStore(id, mb.newShard())
	if !loaded {
		mb.size++
	}
	return s.(*shard), nil
}

func (mb *multiShardBatcher) shardsCount() int {
	mb.lock.Lock()
	defer mb.lock.Unlock()
	return mb.size
}

type tracesBatchConsumer struct {
	*batchConsumer
	cfg          Config
	batchFactory TracesBatchFactory
}

// NewTracesConsumer creates a new traces batch consumer
func NewTracesConsumer(set exporter.CreateSettings, cfg Config, tbf TracesBatchFactory, next internal.RequestSender,
	useOtel bool) (*tracesBatchConsumer, error) {
	bc, err := newBatchConsumer(set, cfg, tbf.IdentifyTracesBatch, next, useOtel)
	if err != nil {
		return nil, err
	}
	return &tracesBatchConsumer{
		batchConsumer: bc,
		cfg:           cfg,
		batchFactory:  tbf,
	}, nil
}

// ConsumeTraces implements TracesConsumer
func (bp *tracesBatchConsumer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	s, err := bp.shard(ctx, td)
	if err != nil {
		return err
	}

	b, err := bp.batchFactory.BatchFromTraces(td)
	if err != nil {
		return consumererror.NewPermanent(fmt.Errorf("cannot create batch from traces: %w", err))
	}

	if ok := s.accept(b); ok {
		return nil
	}

	if bp.batchFactory.BatchFromResourceSpans == nil {
		return consumererror.NewPermanent(fmt.Errorf("recieved too large traces batch of %d items and %d bytes",
			b.Count(), b.Size()))
	}

	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rs := td.ResourceSpans().At(i)
		b, err = bp.batchFactory.BatchFromResourceSpans(rs)
		if err != nil {
			return consumererror.NewPermanent(fmt.Errorf("cannot create batch from resource spans: %w", err))
		}

		if ok := s.accept(b); ok {
			continue
		}

		if bp.batchFactory.BatchFromScopeSpans == nil {
			return consumererror.NewPermanent(fmt.Errorf(
				"recieved too large resource spans batch of %d items and %d bytes", b.Count(), b.Size()))
		}

		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			b, err = bp.batchFactory.BatchFromScopeSpans(rs, j)
			if err != nil {
				return consumererror.NewPermanent(fmt.Errorf("cannot create batch from scope spans: %w", err))
			}

			if ok := s.accept(b); ok {
				continue
			}

			if bp.batchFactory.BatchFromSpan == nil {
				return consumererror.NewPermanent(fmt.Errorf(
					"recieved too large scope spans batch of %d items and %d bytes", b.Count(), b.Size()))
			}

			for k := 0; k < rs.ScopeSpans().At(j).Spans().Len(); k++ {
				b, err = bp.batchFactory.BatchFromSpan(rs, j, k)
				if err != nil {
					return consumererror.NewPermanent(fmt.Errorf("cannot create batch from span: %w", err))
				}

				if ok := s.accept(b); !ok {
					return consumererror.NewPermanent(fmt.Errorf("recieved too large span of %d bytes", b.Size()))
				}
			}
		}
	}

	return nil
}

func (bp *tracesBatchConsumer) shard(ctx context.Context, td ptrace.Traces) (*shard, error) {
	if bp.batchFactory.IdentifyTracesBatch != nil {
		return bp.batcher.shard("")
	}
	return bp.batcher.shard(bp.batchFactory.IdentifyTracesBatch(ctx, td))
}

type metricsBatchConsumer struct {
	*batchConsumer
	cfg          Config
	batchFactory MetricsBatchFactory
}

// NewMetricsConsumer creates a new batcher for metrics
func NewMetricsConsumer(set exporter.CreateSettings, cfg Config, mbf MetricsBatchFactory, next internal.RequestSender,
	useOtel bool) (*metricsBatchConsumer, error) {
	bc, err := newBatchConsumer(set, cfg, mbf.IdentifyMetricsBatch, next, useOtel)
	if err != nil {
		return nil, err
	}
	return &metricsBatchConsumer{
		batchConsumer: bc,
		cfg:           cfg,
		batchFactory:  mbf,
	}, nil
}

// ConsumeMetrics implements MetricsConsumer
func (bp *metricsBatchConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	s, err := bp.shard(ctx, md)
	if err != nil {
		return err
	}

	b, err := bp.batchFactory.BatchFromMetrics(md)
	if err != nil {
		return consumererror.NewPermanent(fmt.Errorf("cannot create batch from metrics: %w", err))
	}

	if ok := s.accept(b); ok {
		return nil
	}

	if bp.batchFactory.BatchFromResourceMetrics == nil {
		return consumererror.NewPermanent(fmt.Errorf("recieved too large metrics batch of %d items and %d bytes",
			b.Count(), b.Size()))
	}

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		b, err = bp.batchFactory.BatchFromResourceMetrics(rm)
		if err != nil {
			return consumererror.NewPermanent(fmt.Errorf("cannot create batch from resource metrics: %w", err))
		}

		if ok := s.accept(b); ok {
			continue
		}

		if bp.batchFactory.BatchFromScopeMetrics == nil {
			return consumererror.NewPermanent(fmt.Errorf(
				"recieved too large resource metrics batch of %d items and %d bytes", b.Count(), b.Size()))
		}

		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			b, err = bp.batchFactory.BatchFromScopeMetrics(rm, i)
			if err != nil {
				return consumererror.NewPermanent(fmt.Errorf(
					"cannot create batch from scope metrics: %w", err))
			}

			if ok := s.accept(b); !ok {
				return consumererror.NewPermanent(fmt.Errorf("recieved too large instrumentation library metrics "+
					"batch of %d items and %d bytes", b.Count(), b.Size()))
			}
		}
	}

	return nil
}

func (bp *metricsBatchConsumer) shard(ctx context.Context, md pmetric.Metrics) (*shard, error) {
	if bp.batchFactory.IdentifyMetricsBatch != nil {
		return bp.batcher.shard("")
	}
	return bp.batcher.shard(bp.batchFactory.IdentifyMetricsBatch(ctx, md))
}

type logsBatchConsumer struct {
	*batchConsumer
	cfg          Config
	batchFactory LogsBatchFactory
}

// NewLogsConsumer creates a new batcher for logs
func NewLogsConsumer(set exporter.CreateSettings, cfg Config, lbf LogsBatchFactory, next internal.RequestSender,
	useOtel bool) (*logsBatchConsumer, error) {
	bc, err := newBatchConsumer(set, cfg, lbf.IdentifyLogsBatch, next, useOtel)
	if err != nil {
		return nil, err
	}
	return &logsBatchConsumer{
		batchConsumer: bc,
		cfg:           cfg,
		batchFactory:  lbf,
	}, nil
}

// ConsumeLogs implements LogsConsumer
func (bp *logsBatchConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	s, err := bp.shard(ctx, ld)
	if err != nil {
		return err
	}

	b, err := bp.batchFactory.BatchFromLogs(ld)
	if err != nil {
		return consumererror.NewPermanent(fmt.Errorf("cannot create batch from logs: %w", err))
	}

	if ok := s.accept(b); ok {
		return nil
	}

	if bp.batchFactory.BatchFromResourceLogs == nil {
		return consumererror.NewPermanent(fmt.Errorf("recieved too large logs batch of %d items and %d bytes",
			b.Count(), b.Size()))
	}

	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rl := ld.ResourceLogs().At(i)
		b, err = bp.batchFactory.BatchFromResourceLogs(rl)
		if err != nil {
			return consumererror.NewPermanent(fmt.Errorf("cannot create batch from resource logs: %w", err))
		}

		if ok := s.accept(b); ok {
			continue
		}

		if bp.batchFactory.BatchFromScopeLogs == nil {
			return consumererror.NewPermanent(fmt.Errorf(
				"recieved too large resource logs batch of %d items and %d bytes", b.Count(), b.Size()))
		}

		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			b, err = bp.batchFactory.BatchFromScopeLogs(rl, i)
			if err != nil {
				return consumererror.NewPermanent(fmt.Errorf(
					"cannot create batch from scope logs: %w", err))
			}

			if ok := s.accept(b); !ok {
				return consumererror.NewPermanent(fmt.Errorf("recieved too large instrumentation library logs "+
					"batch of %d items and %d bytes", b.Count(), b.Size()))
			}
		}
	}

	return nil
}

func (bp *logsBatchConsumer) shard(ctx context.Context, ld plog.Logs) (*shard, error) {
	if bp.batchFactory.IdentifyLogsBatch != nil {
		return bp.batcher.shard("")
	}
	return bp.batcher.shard(bp.batchFactory.IdentifyLogsBatch(ctx, ld))
}
