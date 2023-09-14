// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
)

// BaseBatchConfig defines a basic configuration for batching requests based on a timeout and a minimum number of
// items.
// All additional batching configurations should be embedded along with this struct.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type BaseBatchConfig struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`

	// Timeout sets the time after which a batch will be sent regardless of its size.
	// When this is set to zero, batched data will be sent immediately.
	// This is a recommended option, as it will ensure that the data is sent in a timely manner.
	Timeout time.Duration `mapstructure:"timeout"` // Is there a better name to avoid confusion with the consumerSender timeout?

	// MinSizeItems is the number of items (spans, data points or log records for OTLP) at which the batch should be
	// sent regardless of the timeout. There is no guarantee that the batch size always greater than this value.
	// This option requires the Request to implement RequestItemsCounter interface. Otherwise, it will be ignored.
	MinSizeItems int `mapstructure:"min_size_items"`
}

func NewDefaultBaseBatchConfig() BaseBatchConfig {
	return BaseBatchConfig{
		Enabled:      true,
		Timeout:      200 * time.Millisecond,
		MinSizeItems: 8192,
	}
}

// BatchConfigMaxItems defines batching configuration part for setting a maximum number of items.
// Should not be used directly, use as part of a struct that embeds it.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type BatchConfigMaxItems struct {
	// MaxSizeItems is the maximum number of the batch items, i.e. spans, data points or log records for OTLP,
	// but can be anything else for other formats. If the batch size exceeds this value,
	// it will be broken up into smaller batches if possible.
	MaxSizeItems int `mapstructure:"max_size_items"`
}

// BatchConfigBatchersLimit defines batching configuration part for setting a maximum number of batchers.
type BatchConfigBatchersLimit struct {
	// BatchersLimit is the maximum number of batchers that can be used for batching.
	// Requests producing batch identifiers that exceed this limit will be dropped.
	// If this value is zero, then there is no limit on the number of batchers.
	BatchersLimit int `mapstructure:"batchers_limit"`
}

// BatchMergeFunc is a function that merges two requests into a single request.
// Context will be propagated from the first request. If you want to separate batches based on the context,
// use WithRequestBatchIdentifier option.
type BatchMergeFunc func(context.Context, Request, Request) (Request, error)

// BatchMergeOrSplitFunc is a function that merges two requests into a single request or splits a request into multiple
// requests if the merged request exceeds the maximum number of items.
// If maxItems is zero, then the function must always merge the requests into a single request.
// All the returned requests must have a number of items that does not exceed the maximum number of items.
// Size of the last returned request must not be more than the size of any other returned request.
// If the requirements above are not met, users can expect the req1 to always contain fewer items than maxItems.
// Users must expect that the first request can be nil. In this case, the function must split the second request.
type BatchMergeOrSplitFunc func(ctx context.Context, req1 Request, req2 Request, maxItems int) ([]Request, error)

// IdentifyBatchFunc returns an identifier for a Request batch. This function can be used to separate particular
// Requests into different batches. Batches with different identifiers will not be merged together.
// Provided context can be used to extract information from the context and use it as a part of the identifier as well.
// This function is optional. If not provided, all Requests will be batched together.
type IdentifyBatchFunc func(ctx context.Context, r Request) string

type batcherConfig struct {
	enabled       bool
	timeout       time.Duration
	minSizeItems  int
	maxSizeItems  int
	batchersLimit int
}

// Batcher is a helper struct that can be used to batch requests into different batches.
// This function is optional. If not provided, all Requests will be batched together.
type Batcher struct {
	cfg              batcherConfig
	mergeFunc        func(context.Context, *request, *request) (*request, error)
	mergeOrSplitFunc func(ctx context.Context, req1 *request, req2 *request, maxItems int) ([]*request, error)
	batchIdentifier  IdentifyBatchFunc
}

func NewMergeBatcher(cfg BaseBatchConfig, mf BatchMergeFunc, opts ...BatcherOption) *Batcher {
	b := &Batcher{
		cfg: batcherConfig{
			enabled:      cfg.Enabled,
			timeout:      cfg.Timeout,
			minSizeItems: cfg.MinSizeItems,
		},
		// Internal merge function that merges two requests into a single one.
		// The first request is allowed to be nil. The second request cannot be nil.
		mergeFunc: func(ctx context.Context, req1 *request, req2 *request) (*request, error) {
			if req1 == nil {
				return req2, nil
			}
			r, err := mf(ctx, req1.Request, req2.Request)
			if err != nil {
				return nil, err
			}
			return &request{
				baseRequest: baseRequest{ctx: req1.Context()},
				Request:     r,
			}, nil
		},
	}
	for _, op := range opts {
		op(b)
	}
	return b
}

func NewMergeOrSplitBatcher(cfg BaseBatchConfig, bcmi BatchConfigMaxItems, msf BatchMergeOrSplitFunc, opts ...BatcherOption) *Batcher {
	mergeOrSplitFunc := func(ctx context.Context, req1 *request, req2 *request, maxItems int) ([]*request, error) {
		var r1 Request
		if req1 != nil {
			r1 = req1.Request
		}
		r2 := req2.Request
		r, err := msf(ctx, r1, r2, maxItems)
		if err != nil {
			return nil, err
		}
		reqs := make([]*request, 0, len(r))
		for _, req := range r {
			reqs = append(reqs, &request{
				baseRequest: baseRequest{ctx: req2.Context()},
				Request:     req,
			})
		}
		return reqs, nil
	}
	b := &Batcher{
		cfg: batcherConfig{
			enabled:      cfg.Enabled,
			timeout:      cfg.Timeout,
			minSizeItems: cfg.MinSizeItems,
			maxSizeItems: bcmi.MaxSizeItems,
		},
		mergeOrSplitFunc: mergeOrSplitFunc,
	}
	for _, op := range opts {
		op(b)
	}
	return b
}

type BatcherOption func(*Batcher)

func WithRequestBatchIdentifier(f IdentifyBatchFunc, bl BatchConfigBatchersLimit) BatcherOption {
	return func(b *Batcher) {
		b.batchIdentifier = f
		b.cfg.batchersLimit = bl.BatchersLimit
	}
}

// errTooManyBatchers is returned when the MetadataCardinalityLimit has been reached.
var errTooManyBatchers = consumererror.NewPermanent(errors.New("too many batch identifier combinations"))

// batchSender is a component that accepts places requests into batches before passing them to the downstream senders.
//
// batch_processor implements consumer.Traces and consumer.Metrics
//
// Batches are sent out with any of the following conditions:
// - batch size reaches cfg.SendBatchSize
// - cfg.Timeout is elapsed since the timestamp when the previous batch was sent out.
type batchSender struct {
	baseRequestSender
	cfg              batcherConfig
	mergeFunc        func(context.Context, *request, *request) (*request, error)
	mergeOrSplitFunc func(ctx context.Context, req1 *request, req2 *request, maxItems int) ([]*request, error)

	logger *zap.Logger

	// batchIdentifier is a function that returns a key
	// identifying the batch to which the item should be added.
	batchIdentifier IdentifyBatchFunc

	shutdownCh chan struct{}
	goroutines sync.WaitGroup

	//  batcher will be either *singletonBatcher or *multiBatcher
	batcher batcher
}

// newBatchSender returns a new batch consumer component.
func newBatchSender(set exporter.CreateSettings, b *Batcher) *batchSender {
	bs := &batchSender{
		cfg:              b.cfg,
		mergeFunc:        b.mergeFunc,
		mergeOrSplitFunc: b.mergeOrSplitFunc,
		logger:           set.Logger,
		shutdownCh:       make(chan struct{}),
		batchIdentifier:  b.batchIdentifier,
	}
	if b.batchIdentifier == nil {
		bs.batcher = &singleShardBatcher{batcher: bs.newShard()}
	} else {
		bs.batcher = &multiShardBatcher{batchSender: bs}
	}
	return bs
}

// newShard gets or creates a batcher corresponding with attrs.
func (bs *batchSender) newShard() *shard {
	b := &shard{
		batchSender: bs,
		newRequest:  make(chan *request, runtime.NumCPU()),
	}
	bs.goroutines.Add(1)
	go b.start()
	return b
}

// Shutdown is invoked during service shutdown.
func (bs *batchSender) shutdown() {
	close(bs.shutdownCh)

	// Wait until all goroutines are done.
	bs.goroutines.Wait()
}

func (bs *batchSender) send(req internal.Request) error {
	s, err := bs.batcher.shard(bs.identifyBatch(req.(*request)))
	if err != nil {
		return err
	}

	// For now the batcher can work asyncronously only. Potentially we can
	// add a sync mode later.
	s.newRequest <- req.(*request)
	return nil
}

func (bs *batchSender) identifyBatch(req *request) string {
	if bs.batchIdentifier != nil {
		return bs.batchIdentifier(req.Context(), req.Request)
	}
	return ""
}

type batcher interface {
	shard(id string) (*shard, error)
}

// shard is a single instance of the batch logic.  When metadata
// keys are in use, one of these is created per distinct combination
// of values.
type shard struct {
	// batchSender for access to common fields and configuration.
	*batchSender

	// exportCtx is a context with the metadata key-values
	// corresponding with this shard set.
	exportCtx context.Context

	// timer informs the shard send a batch.
	timer *time.Timer

	// newRequest is used to receive batches from producers.
	newRequest chan *request

	// batch is an in-flight data item containing one of the
	// underlying data types.
	batch *request
}

func (s *shard) start() {
	defer s.batchSender.goroutines.Done()

	if s.batchSender.cfg.timeout == 0 {
		return
	}

	s.timer = time.NewTimer(s.batchSender.cfg.timeout)

	for {
		select {
		case <-s.batchSender.shutdownCh:
		DONE:
			for {
				select {
				case req := <-s.newRequest:
					s.processRequest(req)
				default:
					break DONE
				}
			}
			// This is the close of the channel
			if s.batch != nil && s.batch.Count() > 0 {
				s.export(s.batch)
			}
			return
		case req := <-s.newRequest:
			s.processRequest(req)
		case <-s.timer.C:
			if s.batch != nil && s.batch.Count() > 0 {
				s.export(s.batch)
			}
			s.resetTimer()
		}
	}
}

func (s *shard) processRequest(req *request) {
	var sent bool
	if s.batchSender.mergeOrSplitFunc != nil {
		sent = s.processRequestMergeOrSplit(req)
	} else {
		s.processRequestMerge(req)
	}

	if s.batch != nil && s.batch.Count() >= s.batchSender.cfg.minSizeItems {
		s.export(s.batch)
		s.batch = nil
		sent = true
	}

	if sent {
		s.stopTimer()
		s.resetTimer()
	}
}

func (s *shard) processRequestMergeOrSplit(req *request) bool {
	reqs, err := s.batchSender.mergeOrSplitFunc(s.exportCtx, s.batch, req, s.batchSender.cfg.maxSizeItems)
	if err != nil {
		s.batchSender.logger.Error("Failed to merge or split the request", zap.Error(err))
		return false
	}
	sent := false
	for _, r := range reqs[:len(reqs)-1] {
		s.export(r)
		sent = true
	}
	s.batch = reqs[len(reqs)-1]
	return sent
}

func (s *shard) processRequestMerge(req *request) {
	req, err := s.batchSender.mergeFunc(s.exportCtx, s.batch, req)
	if err != nil {
		s.batchSender.logger.Error("Failed to merge or split the request", zap.Error(err))
		return
	}
	s.batch = req
}

func (s *shard) export(req *request) {
	err := s.batchSender.nextSender.send(req)
	// TODO: Replace with metrics and logging.
	if err != nil {
		s.batchSender.logger.Error("Failed to send batch", zap.Error(err))
	}
}

func (s *shard) hasTimer() bool {
	return s.timer != nil
}

func (s *shard) stopTimer() {
	if s.hasTimer() && !s.timer.Stop() {
		<-s.timer.C
	}
}

func (s *shard) resetTimer() {
	if s.hasTimer() {
		s.timer.Reset(s.batchSender.cfg.timeout)
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

// multiBatcher is used when metadataKeys is not empty.
type multiShardBatcher struct {
	*batchSender
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

	if mb.batchSender.cfg.batchersLimit != 0 && mb.size >= mb.batchSender.cfg.batchersLimit {
		return nil, errTooManyBatchers
	}

	s, loaded := mb.batchers.LoadOrStore(id, mb.newShard())
	if !loaded {
		mb.size++
	}
	return s.(*shard), nil
}
