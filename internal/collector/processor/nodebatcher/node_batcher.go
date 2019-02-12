// Copyright 2019, OpenCensus Authors
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

package nodebatcher

import (
	"context"
	"crypto/md5"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/proto"
	"go.opencensus.io/stats"
	"go.uber.org/zap"

	"github.com/census-instrumentation/opencensus-service/internal/collector/processor"
)

const (
	initialBatchCapacity     = uint32(1024)
	nodeStatusDead           = uint32(1)
	batchStatusClosed        = uint32(1)
	tickerPendingNodesBuffer = 16

	defaultRemoveAfterCycles = uint32(10)
	defaultSendBatchSize     = uint32(8192)
	defaultNumTickers        = 4
	defaultTickTime          = 1 * time.Second
	defaultTimeout           = 1 * time.Second
)

// batcher is a component that accepts spans, and places them into batches grouped by node and resource.
//
// batcher implements processor.SpanProcessor
//
// batcher is a composition of four main pieces. First is its buckets map which maps nodes to buckets.
// Second is the nodeBatcher which keeps a batch associated with a single node, and sends it downstream.
// Third is a bucketTicker that ticks every so often and closes any open and not recently sent batches.
// Fourth is a batch which accepts spans and return when it should be closed due to size.
//
// When we no longer have to batch by node, the following changes should be made:
//   1) batcher should be removed and nodeBatcher should be promoted to batcher
//   2) bucketTicker should be simplified significantly and replaced with a single ticker, since
//      tracking by node is no longer needed.
type batcher struct {
	buckets sync.Map
	sender  processor.SpanProcessor
	tickers []*bucketTicker
	name    string
	logger  *zap.Logger

	removeAfterCycles uint32
	sendBatchSize     uint32
	numTickers        int
	tickTime          time.Duration
	timeout           time.Duration

	bucketMu sync.RWMutex
}

var _ processor.SpanProcessor = (*batcher)(nil)

// NewBatcher creates a new batcher that batches spans by node and resource
func NewBatcher(name string, logger *zap.Logger, sender processor.SpanProcessor, opts ...Option) processor.SpanProcessor {
	// Init with defaults
	b := &batcher{
		name:   name,
		sender: sender,
		logger: logger,

		removeAfterCycles: defaultRemoveAfterCycles,
		sendBatchSize:     defaultSendBatchSize,
		numTickers:        defaultNumTickers,
		tickTime:          defaultTickTime,
		timeout:           defaultTimeout,
	}

	// Override with options
	for _, opt := range opts {
		opt(b)
	}

	// start tickers after options loaded in
	b.tickers = newStartedBucketTickersForBatch(b)
	return b
}

// ProcessSpans implements batcher as a SpanProcessor and takes the provided spans and adds them to
// batches
func (b *batcher) ProcessSpans(request *agenttracepb.ExportTraceServiceRequest, spanFormat string) (uint64, error) {
	bucketID := b.genBucketID(request.Node, request.Resource, spanFormat)
	bucket := b.getOrAddBucket(bucketID, request.Node, request.Resource, spanFormat)
	bucket.add(request.Spans)
	return 0, nil
}

func (b *batcher) genBucketID(node *commonpb.Node, resource *resourcepb.Resource, spanFormat string) string {
	h := md5.New()
	if node != nil {
		nodeKey, err := proto.Marshal(node)
		if err != nil {
			b.logger.Error("Error marshalling node to batcher mapkey.", zap.Error(err))
		} else {
			h.Write(nodeKey)
		}
	}
	if resource != nil {
		resourceKey, err := proto.Marshal(resource) // TODO: remove once resource is in span
		if err != nil {
			b.logger.Error("Error marshalling resource to batcher mapkey.", zap.Error(err))
		} else {
			h.Write(resourceKey)
		}
	}
	return fmt.Sprintf("%x", h.Sum([]byte(spanFormat)))
}

func (b *batcher) getBucket(bucketID string) *nodeBatcher {
	bucket, ok := b.buckets.Load(bucketID)
	if ok {
		return bucket.(*nodeBatcher)
	}
	return nil
}

func (b *batcher) getOrAddBucket(
	bucketID string, node *commonpb.Node, resource *resourcepb.Resource, spanFormat string,
) *nodeBatcher {
	bucket, alreadyStored := b.buckets.LoadOrStore(
		bucketID,
		newNodeBucket(b, node, resource, spanFormat, b.sendBatchSize, b.timeout, b.logger),
	)
	// Add this bucket to a random ticker
	if !alreadyStored {
		stats.Record(context.Background(), statNodesAddedToBatches.M(1))
		b.tickers[rand.Intn(len(b.tickers))].add(bucketID)
	}
	return bucket.(*nodeBatcher)
}

func (b *batcher) removeBucket(bucketID string) {
	stats.Record(context.Background(), statNodesRemovedFromBatches.M(1))
	b.buckets.Delete(bucketID)
}

type nodeBatcher struct {
	// Please keep this field as the first element to ensure alignment for atomics on 32-bit systems
	lastSent int64

	timeout       time.Duration
	sendBatchSize uint32
	currBatch     *batch
	node          *commonpb.Node
	resource      *resourcepb.Resource // TODO(skaris) remove when resource is added to span
	spanFormat    string
	logger        *zap.Logger

	dead            uint32
	cyclesUntouched uint32
	parent          *batcher
}

func newNodeBucket(
	parent *batcher,
	node *commonpb.Node,
	resource *resourcepb.Resource,
	spanFormat string,
	sendBatchSize uint32,
	timeout time.Duration,
	logger *zap.Logger,
) *nodeBatcher {
	nb := &nodeBatcher{
		timeout:       timeout,
		sendBatchSize: sendBatchSize,
		currBatch:     newBatch(initialBatchCapacity, sendBatchSize),
		node:          node,
		resource:      resource,
		spanFormat:    spanFormat,
		lastSent:      int64(0),
		logger:        logger,

		dead:            uint32(0),
		cyclesUntouched: uint32(0),
		parent:          parent,
	}

	return nb
}

func (nb *nodeBatcher) add(spans []*tracepb.Span) {
	// Reset cycles untouched to 0
	atomic.StoreUint32(&nb.cyclesUntouched, 0)
	// Add will always add these spans to the current batch, if ok is false it means that
	// we need to cut
	var b *batch
	closed := true
	cutBatch := false
	for closed {
		// atomic.LoadPointer only takes unsafe.Pointer interfaces. We do not use unsafe
		// to skirt around the golang type system.
		b = (*batch)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&nb.currBatch))))
		cutBatch, closed = b.add(spans)
	}
	if atomic.LoadUint32(&nb.dead) == nodeStatusDead || cutBatch {
		statsTags := processor.StatsTagsForBatch(
			nb.parent.name, processor.ServiceNameForNode(nb.node), nb.spanFormat,
		)
		if cutBatch {
			stats.RecordWithTags(context.Background(), statsTags, statBatchSizeTriggerSend.M(1))
		} else {
			stats.RecordWithTags(context.Background(), statsTags, statBatchOnDeadNode.M(1))
		}
		// Shoot off event so that we don't block this add on other adds
		go nb.cutBatch(b)
	}
}

// cutBatch cuts the provided batch, and sets a new batch on this nodeBatcher
func (nb *nodeBatcher) cutBatch(b *batch) {
	// atomic.CompareAndSwapPointer only takes unsafe.Pointer interfaces. We do not use unsafe
	// to skirt around the golang type system.
	currBatchPtr := (*unsafe.Pointer)(unsafe.Pointer(&nb.currBatch))
	swapped := atomic.CompareAndSwapPointer(
		currBatchPtr,
		unsafe.Pointer(b),
		unsafe.Pointer(newBatch(b.getCurrentCap(), nb.sendBatchSize)),
	)
	// Since we are doing an atomic compare and swap, this batch will only be sent once.
	if swapped {
		b.closeBatch()
		atomic.StoreInt64(&nb.lastSent, time.Now().UnixNano())
		nb.sendBatch(b)
	}
}

func (nb *nodeBatcher) sendBatch(batch *batch) {
	spans := batch.getSpans()
	statsTags := processor.StatsTagsForBatch(
		nb.parent.name, processor.ServiceNameForNode(nb.node), nb.spanFormat,
	)
	stats.RecordWithTags(context.Background(), statsTags, statBatchSize.M(int64(len(spans))))

	if len(spans) == 0 {
		return
	}
	request := &agenttracepb.ExportTraceServiceRequest{
		Node:     nb.node,
		Resource: nb.resource,
		Spans:    spans,
	}
	_, err := nb.parent.sender.ProcessSpans(request, nb.spanFormat)
	// Assumed that the next processor always handles a batch, and doesn't error
	if err != nil {
		nb.logger.Error(
			"Failed to process batch, discarding",
			zap.String("processor", "batcher"),
			zap.Uint32("batch-size", batch.getCurrentItemCount()),
		)
	}
}

type bucketTicker struct {
	ticker       *time.Ticker
	nodes        map[string]bool
	parent       *batcher
	pendingNodes chan string
	stopCn       chan struct{}
	once         sync.Once
}

func newStartedBucketTickersForBatch(b *batcher) []*bucketTicker {
	tickers := make([]*bucketTicker, 0, b.numTickers)
	for ignored := 0; ignored < b.numTickers; ignored++ {
		ticker := newBucketTicker(b, b.tickTime)
		go ticker.start()
		tickers = append(tickers, ticker)
	}
	return tickers
}

func newBucketTicker(parent *batcher, tickTime time.Duration) *bucketTicker {
	return &bucketTicker{
		ticker:       time.NewTicker(tickTime),
		nodes:        make(map[string]bool),
		parent:       parent,
		pendingNodes: make(chan string, tickerPendingNodesBuffer),
		stopCn:       make(chan struct{}),
	}
}

func (bt *bucketTicker) add(bucketID string) {
	bt.pendingNodes <- bucketID
}

func (bt *bucketTicker) start() {
	bt.once.Do(func() {
		for {
			select {
			case <-bt.ticker.C:
				for nbKey := range bt.nodes {
					nb := bt.parent.getBucket(nbKey)
					// Need to check nil here incase the node was deleted from the parent batcher, but
					// not deleted from bt.nodes
					if nb == nil {
						// re-delete just in case
						delete(bt.nodes, nbKey)
						continue
					}
					// atomic.LoadPointer only takes unsafe.Pointer interfaces. We do not use unsafe
					// to skirt around the golang type system.
					ptr := (*unsafe.Pointer)(unsafe.Pointer(&nb.currBatch))
					b := (*batch)(atomic.LoadPointer(ptr))
					if atomic.LoadInt64(&nb.lastSent)+nb.timeout.Nanoseconds() > time.Now().UnixNano() {
						// In this case, we sent more recently than the timeout, so do nothing
						continue
					} else if b.getCurrentItemCount() > 0 {
						// If the batch is non-empty, go ahead and cut it
						statsTags := processor.StatsTagsForBatch(
							nb.parent.name, processor.ServiceNameForNode(nb.node), nb.spanFormat,
						)
						stats.RecordWithTags(context.Background(), statsTags, statTimeoutTriggerSend.M(1))
						nb.cutBatch(b)
					} else {
						cyclesUntouched := atomic.AddUint32(&nb.cyclesUntouched, 1)
						if cyclesUntouched > nb.parent.removeAfterCycles {
							nb.parent.removeBucket(nbKey)
							delete(bt.nodes, nbKey)
							// At this point, add() should no longer be callable from the parent,
							// so mark this nodeBatcher as dead (for adds that have already been called and are racing)
							// and try to cut a final batch (just in case something was added in between
							// the above if and marking this nodeBatcher as dead)
							atomic.StoreUint32(&nb.dead, nodeStatusDead)
							// Reload batch in case `add` loaded a new batch that was then added to.
							currBatchPtr := (*unsafe.Pointer)(unsafe.Pointer(&nb.currBatch))
							b = (*batch)(atomic.LoadPointer(currBatchPtr))
							if b.getCurrentItemCount() > 0 {
								nb.cutBatch(b)
							}
						}
					}
				}
			case newBucketKey := <-bt.pendingNodes:
				bt.nodes[newBucketKey] = true
			case <-bt.stopCn:
				bt.ticker.Stop()
				return
			}
		}
	})
}

func (bt *bucketTicker) stop() {
	close(bt.stopCn)
}

type batch struct {
	items         atomic.Value // []*tracepb.Span
	currCap       uint32
	nextEmptyItem uint32
	sendItemsSize uint32

	closed  uint32
	growMu  sync.Mutex
	pending int32
}

func newBatch(initCapacity uint32, sendBatchSize uint32) *batch {
	batch := &batch{
		currCap:       initCapacity,
		nextEmptyItem: uint32(0),
		sendItemsSize: sendBatchSize,

		closed: uint32(0),
	}

	batch.items.Store(make([]*tracepb.Span, initCapacity))
	return batch
}

func (b *batch) add(spans []*tracepb.Span) (cut bool, closed bool) {
	if atomic.LoadUint32(&b.closed) == batchStatusClosed {
		return false, true
	}
	// Add to pending and then check closed again. If closed, just return and decrement pending.
	atomic.AddInt32(&b.pending, int32(1))
	defer atomic.AddInt32(&b.pending, int32(-1))
	if atomic.LoadUint32(&b.closed) == batchStatusClosed {
		return false, true
	}
	openTill := atomic.AddUint32(&b.nextEmptyItem, uint32(len(spans)))
	openFrom := openTill - uint32(len(spans))
	currCap := atomic.LoadUint32(&b.currCap)
	if openTill > currCap {
		b.grow(openTill)
	}

	for spanIndex, span := range spans {
		b.items.Load().([]*tracepb.Span)[openFrom+uint32(spanIndex)] = span
	}

	return openTill > b.sendItemsSize, false
}

func (b *batch) closeBatch() {
	atomic.StoreUint32(&b.closed, batchStatusClosed)
	for {
		// We only need to wait for goroutines currently executing `add`
		// We are safe from closing during a items mutation due to the second
		// closed check after incrementing pending
		pending := atomic.LoadInt32(&b.pending)
		if pending == int32(0) {
			return
		}
		runtime.Gosched()
	}
}

func (b *batch) grow(neededSize uint32) {
	b.growMu.Lock()
	defer b.growMu.Unlock()
	currCap := atomic.LoadUint32(&b.currCap)
	// If we entered this function concurrently with another call to grow,
	// make sure we don't needlessly re-copy items
	if neededSize < currCap {
		return
	}
	newCap := currCap
	for newCap < neededSize {
		newCap = newCap * 2
	}
	newItems := make([](*tracepb.Span), newCap, newCap)
	copy(newItems, b.items.Load().([]*tracepb.Span))
	b.items.Store(newItems)
	// It is important that we store the new cap after copying all items, so
	// that concurrent adds also attempt to grow and acquire the lock
	atomic.StoreUint32(&b.currCap, newCap)
}

func (b *batch) getCurrentCap() uint32 {
	return atomic.LoadUint32(&b.currCap)
}

func (b *batch) getCurrentItemCount() uint32 {
	return atomic.LoadUint32(&b.nextEmptyItem)
}

func (b *batch) getSpans() []*tracepb.Span {
	// This shouldn't copy the backing array, just create a new slice with the correct
	// length
	return b.items.Load().([]*tracepb.Span)[:b.getCurrentItemCount()]
}
