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
	"crypto/sha256"
	"fmt"
	"math/rand"
	"sync"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/proto"
	"go.opencensus.io/stats"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

const (
	initialBatchCapacity     = uint32(16)
	nodeStatusDead           = uint32(1)
	tickerPendingNodesBuffer = 16

	defaultRemoveAfterCycles = uint32(10)
	defaultSendBatchSize     = uint32(8192)
	defaultNumTickers        = 4
	defaultTickTime          = 1 * time.Second
	defaultTimeout           = 1 * time.Second
)

// batcher is a component that accepts spans, and places them into batches grouped by node and resource.
//
// batcher implements consumer.TraceConsumer
//
// batcher is a composition of four main pieces. First is its buckets map which maps nodes to buckets.
// Second is the nodebatcher which keeps a batch associated with a single node, and sends it downstream.
// Third is a bucketTicker that ticks every so often and closes any open and not recently sent batches.
//
// When we no longer have to batch by node, the following changes should be made:
//   1) batcher should be removed and nodebatcher should be promoted to batcher
//   2) bucketTicker should be simplified significantly and replaced with a single ticker, since
//      tracking by node is no longer needed.
type batcher struct {
	buckets sync.Map
	sender  consumer.TraceConsumer
	tickers []*bucketTicker
	name    string
	logger  *zap.Logger

	removeAfterCycles uint32
	sendBatchSize     uint32
	numTickers        int
	tickTime          time.Duration
	timeout           time.Duration
}

var _ consumer.TraceConsumer = (*batcher)(nil)

// NewBatcher creates a new batcher that batches spans by node and resource
func NewBatcher(name string, logger *zap.Logger, sender consumer.TraceConsumer, opts ...Option) processor.TraceProcessor {
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

// ConsumeTraceData implements batcher as a SpanProcessor and takes the provided spans and adds them to
// batches
func (b *batcher) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	bucketID := b.genBucketID(td.Node, td.Resource, td.SourceFormat)
	bucket := b.getOrAddBucket(bucketID, td.Node, td.Resource, td.SourceFormat)
	bucket.add(td.Spans)
	return nil
}

func (b *batcher) GetCapabilities() processor.Capabilities {
	return processor.Capabilities{MutatesConsumedData: false}
}

// Start is invoked during service startup.
func (b *batcher) Start(host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (b *batcher) Shutdown() error {
	// TODO: flush accumulated data.
	return nil
}

func (b *batcher) genBucketID(node *commonpb.Node, resource *resourcepb.Resource, spanFormat string) string {
	h := sha256.New()
	if node != nil {
		nodeKey, err := proto.Marshal(node)
		if err != nil {
			b.logger.Error("Error marshaling node to batcher mapkey.", zap.Error(err))
		} else {
			h.Write(nodeKey)
		}
	}
	if resource != nil {
		resourceKey, err := proto.Marshal(resource) // TODO: remove once resource is in span
		if err != nil {
			b.logger.Error("Error marshaling resource to batcher mapkey.", zap.Error(err))
		} else {
			h.Write(resourceKey)
		}
	}
	return fmt.Sprintf("%x", h.Sum([]byte(spanFormat)))
}

func (b *batcher) getBucket(bucketID string) *nodeBatch {
	bucket, ok := b.buckets.Load(bucketID)
	if ok {
		return bucket.(*nodeBatch)
	}
	return nil
}

func (b *batcher) getOrAddBucket(
	bucketID string, node *commonpb.Node, resource *resourcepb.Resource, spanFormat string,
) *nodeBatch {
	bucket, loaded := b.buckets.Load(bucketID)
	if !loaded {
		bucket, loaded = b.buckets.LoadOrStore(
			bucketID,
			newNodeBatch(b, spanFormat, node, resource),
		)
		// Add this bucket to a random ticker
		if !loaded {
			stats.Record(context.Background(), statNodesAddedToBatches.M(1))
			b.tickers[rand.Intn(len(b.tickers))].add(bucketID)
		}
	}

	return bucket.(*nodeBatch)
}

func (b *batcher) removeBucket(bucketID string) {
	stats.Record(context.Background(), statNodesRemovedFromBatches.M(1))
	b.buckets.Delete(bucketID)
}

type nodeBatch struct {
	mu              sync.RWMutex
	items           [][]*tracepb.Span
	totalItemCount  uint32
	cyclesUntouched uint32
	dead            uint32
	lastSent        int64

	parent   *batcher
	format   string
	node     *commonpb.Node
	resource *resourcepb.Resource
}

func newNodeBatch(
	parent *batcher,
	format string,
	node *commonpb.Node,
	resource *resourcepb.Resource,
) *nodeBatch {
	return &nodeBatch{
		parent:   parent,
		format:   format,
		node:     node,
		resource: resource,
		items:    make([][]*tracepb.Span, 0, initialBatchCapacity),
	}
}

func (nb *nodeBatch) add(spans []*tracepb.Span) {
	nb.mu.Lock()
	nb.items = append(nb.items, spans)
	nb.totalItemCount = nb.totalItemCount + uint32(len(spans))
	nb.cyclesUntouched = 0

	itemCount := nb.totalItemCount
	var itemsToProcess [][]*tracepb.Span
	if nb.totalItemCount > nb.parent.sendBatchSize || nb.dead == nodeStatusDead {
		itemsToProcess, itemCount = nb.getAndReset()
	}
	nb.mu.Unlock()

	if len(itemsToProcess) > 0 {
		nb.sendItems(itemsToProcess, itemCount, statBatchSizeTriggerSend)
	}
}

func (nb *nodeBatch) sendItems(
	itemsToProcess [][]*tracepb.Span,
	itemCount uint32,
	measure *stats.Int64Measure,
) {
	tdItems := make([]*tracepb.Span, 0, itemCount)
	for _, items := range itemsToProcess {
		tdItems = append(tdItems, items...)
	}
	td := consumerdata.TraceData{
		Node:         nb.node,
		Resource:     nb.resource,
		Spans:        tdItems,
		SourceFormat: nb.format,
	}
	statsTags := processor.StatsTagsForBatch(
		nb.parent.name, processor.ServiceNameForNode(nb.node), nb.format,
	)
	_ = stats.RecordWithTags(context.Background(), statsTags, measure.M(1))

	// TODO: This process should be done in an async way, perhaps with a channel + goroutine worker(s)
	ctx := observability.ContextWithReceiverName(context.Background(), nb.format)
	_ = nb.parent.sender.ConsumeTraceData(ctx, td)
}

func (nb *nodeBatch) getAndReset() ([][]*tracepb.Span, uint32) {
	itemsToProcess := nb.items
	itemsCount := nb.totalItemCount
	nb.items = make([][]*tracepb.Span, 0, len(itemsToProcess))
	nb.lastSent = time.Now().UnixNano()
	nb.totalItemCount = 0
	return itemsToProcess, itemsCount
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
	bt.once.Do(bt.runTicker)
}

func (bt *bucketTicker) runTicker() {
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
				} else {
					bt.processNodeBatch(nbKey, nb)
				}
			}
		case newBucketKey := <-bt.pendingNodes:
			bt.nodes[newBucketKey] = true
		case <-bt.stopCn:
			bt.ticker.Stop()
			return
		}
	}
}

func (bt *bucketTicker) processNodeBatch(nbKey string, nb *nodeBatch) {
	nb.mu.Lock()
	if nb.totalItemCount > 0 {
		// If the batch is non-empty, go ahead and send it
		var itemCount uint32
		var itemsToProcess [][]*tracepb.Span
		if nb.lastSent+bt.parent.timeout.Nanoseconds() < time.Now().UnixNano() {
			itemsToProcess, itemCount = nb.getAndReset()
		}
		nb.mu.Unlock()

		if len(itemsToProcess) > 0 {
			nb.sendItems(itemsToProcess, itemCount, statTimeoutTriggerSend)
		}
	} else {
		nb.cyclesUntouched++
		if nb.cyclesUntouched > nb.parent.removeAfterCycles {
			nb.parent.removeBucket(nbKey)
			delete(bt.nodes, nbKey)
			nb.dead = nodeStatusDead
		}
		nb.mu.Unlock()
	}
}

func (bt *bucketTicker) stop() {
	close(bt.stopCn)
}
