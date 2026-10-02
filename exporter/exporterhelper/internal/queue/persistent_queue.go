// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/pipeline"
)

const (
	zapKey           = "key"
	zapErrorCount    = "errorCount"
	zapNumberOfItems = "numberOfItems"

	legacyReadIndexKey                = "ri"
	legacyWriteIndexKey               = "wi"
	legacyCurrentlyDispatchedItemsKey = "di"
	replayItemsKey                    = "qrv0"

	// metadataKey is the new single key for all queue metadata.
	metadataKey = "qmv0"
)

var (
	errValueNotSet        = errors.New("value not set")
	errInvalidValue       = errors.New("invalid value")
	errNoStorageClient    = errors.New("no storage client extension found")
	errWrongExtensionType = errors.New("requested extension is not a storage extension")
)

var indexDonePool = sync.Pool{
	New: func() any {
		return &indexDone{}
	},
}

// persistentQueue provides a persistent queue implementation backed by file storage extension
//
// Write index describes the position at which next item is going to be stored.
// Read index describes which item needs to be read next.
// When Write index = Read index, no elements are in the queue.
//
// The items currently dispatched by consumers are not deleted until the processing is finished.
// Their list is stored under a separate key.
//
//	┌───────file extension-backed queue───────┐
//	│                                         │
//	│     ┌───┐     ┌───┐ ┌───┐ ┌───┐ ┌───┐   │
//	│ n+1 │ n │ ... │ 4 │ │ 3 │ │ 2 │ │ 1 │   │
//	│     └───┘     └───┘ └─x─┘ └─|─┘ └─x─┘   │
//	│                       x     |     x     │
//	└───────────────────────x─────|─────x─────┘
//	   ▲              ▲     x     |     x
//	   │              │     x     |     xxxx deleted
//	   │              │     x     |
//	 write          read    x     └── currently dispatched item
//	 index          index   x
//	                        xxxx deleted
type persistentQueue[T request.Request] struct {
	logger      *zap.Logger
	client      storage.Client
	encoding    Encoding[T]
	capacity    int64
	sizerType   request.SizerType
	activeSizer request.Sizer
	itemsSizer  request.Sizer
	bytesSizer  request.Sizer
	storageID   component.ID
	id          component.ID
	signal      pipeline.Signal

	// mu guards everything declared below.
	mu              sync.Mutex
	hasMoreElements *sync.Cond
	hasMoreSpace    *cond
	metadata        PersistentMetadata
	replayItems     []uint64
	refClient       int64
	stopped         bool
	startupErr      error

	blockOnOverflow bool
	replayInOrder   bool
}

// newPersistentQueue creates a new queue backed by file storage; name and signal must be a unique combination that identifies the queue storage
func newPersistentQueue[T request.Request](set Settings[T]) readableQueue[T] {
	pq := &persistentQueue[T]{
		logger:          set.Telemetry.Logger,
		encoding:        set.Encoding,
		capacity:        set.Capacity,
		sizerType:       set.SizerType,
		activeSizer:     request.NewSizer(set.SizerType),
		itemsSizer:      request.NewItemsSizer(),
		bytesSizer:      request.NewBytesSizer(),
		storageID:       *set.StorageID,
		id:              set.ID,
		signal:          set.Signal,
		blockOnOverflow: set.BlockOnOverflow,
		replayInOrder:   set.ReplayInOrder,
	}
	pq.hasMoreElements = sync.NewCond(&pq.mu)
	pq.hasMoreSpace = newCond(&pq.mu)
	return pq
}

// Start starts the persistentQueue with the given number of consumers.
func (pq *persistentQueue[T]) Start(ctx context.Context, host component.Host) error {
	storageClient, err := toStorageClient(ctx, pq.storageID, host, pq.id, pq.signal)
	if err != nil {
		return err
	}
	pq.initClient(ctx, storageClient)
	if pq.startupErr != nil {
		_ = storageClient.Close(ctx)
		pq.client = nil
		return pq.startupErr
	}
	return nil
}

func (pq *persistentQueue[T]) Size() int64 {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return pq.internalSize()
}

func (pq *persistentQueue[T]) internalSize() int64 {
	switch pq.sizerType {
	case request.SizerTypeBytes:
		return pq.metadata.BytesSize
	case request.SizerTypeItems:
		return pq.metadata.ItemsSize
	default:
		return pq.metadata.RequestsSize
	}
}

func (pq *persistentQueue[T]) requestSize() int64 {
	return int64(pq.metadata.WriteIndex-pq.metadata.ReadIndex) + int64(len(pq.metadata.CurrentlyDispatchedItems)+len(pq.replayItems))
}

func (pq *persistentQueue[T]) Capacity() int64 {
	return pq.capacity
}

func (pq *persistentQueue[T]) initClient(ctx context.Context, client storage.Client) {
	pq.client = client
	// Start with a reference 1 which is the reference we use for the producer goroutines and initialization.
	pq.refClient = 1

	// Try to load from new consolidated metadata first
	err := pq.loadQueueMetadata(ctx)
	switch {
	case err == nil:
		previousRequests := pq.requestSize()
		pq.enqueueNotDispatchedReqs(ctx, pq.metadata.CurrentlyDispatchedItems)
		if pq.startupErr != nil {
			return
		}
		pq.metadata.CurrentlyDispatchedItems = nil
		if !pq.replayInOrder && pq.metadata.RequestsSize > 0 {
			// The existing recovery path drops unreadable in-flight payloads.
			// Keep their admission charge consistent with that behavior.
			pq.metadata.RequestsSize -= previousRequests - pq.requestSize()
		}
		if pq.metadata.RequestsSize == 0 {
			pq.metadata.RequestsSize = pq.requestSize()
		}
	case !errors.Is(err, errValueNotSet):
		pq.logger.Error("Failed getting metadata, starting with new ones", zap.Error(err))
		pq.metadata = PersistentMetadata{}
	default:
		pq.logger.Info("New queue metadata key not found, attempting to load legacy format.")
		pq.loadLegacyMetadata(ctx)
	}
}

// loadQueueMetadata loads queue metadata from the consolidated key
func (pq *persistentQueue[T]) loadQueueMetadata(ctx context.Context) error {
	buf, err := pq.client.Get(ctx, metadataKey)
	if err != nil {
		return err
	}

	if len(buf) == 0 {
		return errValueNotSet
	}

	if err := proto.Unmarshal(buf, &pq.metadata); err != nil {
		return err
	}

	pq.logger.Info("Loaded queue metadata",
		zap.Uint64("readIndex", pq.metadata.ReadIndex),
		zap.Uint64("writeIndex", pq.metadata.WriteIndex),
		zap.Int64("itemsSize", pq.metadata.ItemsSize),
		zap.Int64("bytesSize", pq.metadata.BytesSize),
		zap.Int("dispatchedItems", len(pq.metadata.CurrentlyDispatchedItems)))

	return nil
}

// TODO: Remove legacy format support after 6 months (target: December 2025)
func (pq *persistentQueue[T]) loadLegacyMetadata(ctx context.Context) {
	// Fallback to legacy individual keys for backward compatibility
	riOp := storage.GetOperation(legacyReadIndexKey)
	wiOp := storage.GetOperation(legacyWriteIndexKey)

	err := pq.client.Batch(ctx, riOp, wiOp)
	if err == nil {
		pq.metadata.ReadIndex, err = bytesToItemIndex(riOp.Value)
	}

	if err == nil {
		pq.metadata.WriteIndex, err = bytesToItemIndex(wiOp.Value)
	}

	if err != nil {
		if errors.Is(err, errValueNotSet) {
			pq.logger.Info("Initializing new persistent queue")
		} else {
			pq.logger.Error("Failed getting read/write index, starting with new ones", zap.Error(err))
		}
		pq.metadata.ReadIndex = 0
		pq.metadata.WriteIndex = 0
	}

	pq.retrieveAndEnqueueNotDispatchedReqs(ctx)
	if pq.startupErr != nil {
		return
	}
	pq.metadata.RequestsSize = pq.requestSize()

	// Save to a new format and clean up legacy keys
	metadataBytes, err := proto.Marshal(&pq.metadata)
	if err != nil {
		pq.logger.Error("Failed to marshal metadata", zap.Error(err))
		return
	}

	if err = pq.client.Set(ctx, metadataKey, metadataBytes); err != nil {
		pq.logger.Error("Failed to persist current metadata to storage", zap.Error(err))
		return
	}

	if err = pq.client.Batch(ctx,
		storage.DeleteOperation(legacyReadIndexKey),
		storage.DeleteOperation(legacyWriteIndexKey),
		storage.DeleteOperation(legacyCurrentlyDispatchedItemsKey)); err != nil {
		pq.logger.Warn("Failed to cleanup legacy metadata keys", zap.Error(err))
	} else {
		pq.logger.Info("Successfully migrated to consolidated metadata format")
	}
}

func (pq *persistentQueue[T]) Shutdown(ctx context.Context) error {
	// If the queue is not initialized, there is nothing to shut down.
	if pq.client == nil {
		return nil
	}

	pq.mu.Lock()
	defer pq.mu.Unlock()
	// Mark this queue as stopped, so consumer don't start any more work.
	pq.stopped = true
	pq.hasMoreElements.Broadcast()
	return pq.unrefClient(ctx)
}

// LoadCheckpoint reads an exporter checkpoint from the same storage client as
// this queue. Checkpoints share the queue's storage namespace but use a
// disjoint key prefix from queue metadata and item bodies.
func (pq *persistentQueue[T]) LoadCheckpoint(ctx context.Context, key string) ([]byte, bool, error) {
	if key == "" {
		return nil, false, errors.New("persistent queue checkpoint key is empty")
	}
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if pq.client == nil || pq.stopped {
		return nil, false, errors.New("persistent queue is not available for checkpoints")
	}
	data, err := pq.client.Get(ctx, "ocp/"+key)
	if errors.Is(err, errValueNotSet) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if len(data) == 0 {
		return nil, false, nil
	}
	value, updates, err := decodeCheckpointJournal(data)
	if err != nil {
		return nil, false, err
	}
	if len(updates) > 0 {
		if err := pq.applyCheckpointItems(ctx, updates); err != nil {
			return nil, false, err
		}
		if err := pq.client.Set(ctx, "ocp/"+key, value); err != nil {
			return nil, false, err
		}
	}
	return value, true, nil
}

func (pq *persistentQueue[T]) LoadQueueItem(ctx context.Context, token uint64) ([]byte, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if pq.client == nil || pq.stopped {
		return nil, errors.New("persistent queue is not available")
	}
	return pq.client.Get(ctx, getItemKey(token))
}

func (pq *persistentQueue[T]) SaveCheckpoint(ctx context.Context, key string, value []byte) error {
	return pq.SaveCheckpointAndItems(ctx, key, value, nil)
}

// SaveCheckpointAndItems publishes one journal value containing both the tail
// and its retired child envelopes. This single-key commit avoids relying on
// storage.Client.Batch being transactional. Envelope copies may be applied
// later; recovery completes that work before exposing the saved tail.
func (pq *persistentQueue[T]) SaveCheckpointAndItems(ctx context.Context, key string, value []byte, updates []request.QueueItemUpdate) error {
	if key == "" {
		return errors.New("persistent queue checkpoint key is empty")
	}
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if pq.client == nil || pq.stopped {
		return errors.New("persistent queue is not available for checkpoints")
	}
	previous, err := pq.client.Get(ctx, "ocp/"+key)
	if err != nil && !errors.Is(err, errValueNotSet) {
		return err
	}
	_, pending, err := decodeCheckpointJournal(previous)
	if err != nil {
		return err
	}
	merged := make(map[uint64][]byte, len(pending)+len(updates))
	for _, update := range pending {
		merged[update.Token] = update.Value
	}
	for _, update := range updates {
		merged[update.Token] = update.Value
	}
	tokens := make([]uint64, 0, len(merged))
	for token := range merged {
		tokens = append(tokens, token)
	}
	slices.Sort(tokens)
	updates = updates[:0]
	for _, token := range tokens {
		// A completed envelope may already have been removed by OnDone.
		body, err := pq.client.Get(ctx, getItemKey(token))
		if err != nil && !errors.Is(err, errValueNotSet) {
			return err
		}
		if len(body) != 0 {
			updates = append(updates, request.QueueItemUpdate{Token: token, Value: merged[token]})
		}
	}
	journal := encodeCheckpointJournal(value, updates)
	if err := pq.client.Set(ctx, "ocp/"+key, journal); err != nil {
		return err
	}
	if len(updates) > 0 {
		if err := pq.applyCheckpointItems(ctx, updates); err != nil {
			pq.logger.Warn("Ordered queue progress is durable in its recovery journal", zap.Error(err))
			return nil
		}
		// Failure to trim is safe: applying the journal again is idempotent.
		if err := pq.client.Set(ctx, "ocp/"+key, value); err != nil {
			pq.logger.Warn("Unable to trim ordered queue recovery journal", zap.Error(err))
		}
	}
	return nil
}

func (pq *persistentQueue[T]) applyCheckpointItems(ctx context.Context, updates []request.QueueItemUpdate) error {
	for _, update := range updates {
		body, err := pq.client.Get(ctx, getItemKey(update.Token))
		if err != nil && !errors.Is(err, errValueNotSet) {
			return err
		}
		if len(body) == 0 {
			continue
		}
		if err := pq.client.Set(ctx, getItemKey(update.Token), update.Value); err != nil {
			return err
		}
	}
	return nil
}

// unrefClient unrefs the client, and closes if no more references. Callers MUST hold the mutex.
// This is needed because consumers of the queue may still process the requests while the queue is shutting down or immediately after.
func (pq *persistentQueue[T]) unrefClient(ctx context.Context) error {
	pq.refClient--
	if pq.refClient == 0 {
		return pq.client.Close(ctx)
	}
	return nil
}

// Offer inserts the specified element into this queue if it is possible to do so immediately
// without violating capacity restrictions. If success returns no error.
// It returns ErrQueueIsFull if no space is currently available.
func (pq *persistentQueue[T]) Offer(ctx context.Context, req T) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	size := pq.activeSizer.Sizeof(req)
	for pq.internalSize()+size > pq.capacity {
		if !pq.blockOnOverflow {
			return ErrQueueIsFull
		}
		if err := pq.hasMoreSpace.Wait(ctx); err != nil {
			return err
		}
	}

	pq.metadata.RequestsSize += (request.RequestsSizer{}).Sizeof(req)
	pq.metadata.ItemsSize += pq.itemsSizer.Sizeof(req)
	pq.metadata.BytesSize += pq.bytesSizer.Sizeof(req)

	return pq.putInternal(ctx, req)
}

// putInternal adds the request to the storage without updating items/bytes sizes.
func (pq *persistentQueue[T]) putInternal(ctx context.Context, req T) error {
	pq.metadata.WriteIndex++

	metadataBuf, err := proto.Marshal(&pq.metadata)
	if err != nil {
		return err
	}

	reqBuf, err := pq.encoding.Marshal(ctx, req)
	if err != nil {
		return err
	}
	// Carry out a transaction where we both add the item and update the write index
	ops := []*storage.Operation{
		storage.SetOperation(metadataKey, metadataBuf),
		storage.SetOperation(getItemKey(pq.metadata.WriteIndex-1), reqBuf),
	}
	if err := pq.client.Batch(ctx, ops...); err != nil {
		// At this moment, metadata may be updated in the storage, so we cannot just revert changes to the
		// metadata, rely on the sizes being fixed on complete draining.
		return err
	}

	pq.hasMoreElements.Signal()

	return nil
}

func (pq *persistentQueue[T]) Read(ctx context.Context) (context.Context, T, Done, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for {
		if pq.stopped {
			var req T
			return context.Background(), req, nil, false
		}

		// Read until either a successful retrieved element or no more elements in the storage.
		for len(pq.replayItems) > 0 || pq.metadata.ReadIndex != pq.metadata.WriteIndex {
			index, req, reqCtx, consumed := pq.getNextItem(ctx)
			// Ensure the used size are in sync when queue is drained.
			if pq.requestSize() == 0 {
				pq.metadata.BytesSize = 0
				pq.metadata.RequestsSize = 0
				pq.metadata.ItemsSize = 0
			}
			if consumed {
				if setter, ok := any(req).(request.QueueItemTokenSetter); ok {
					setter.SetQueueItemToken(index)
				}
				id := indexDonePool.Get().(*indexDone)
				id.reset(index, pq.itemsSizer.Sizeof(req), pq.bytesSizer.Sizeof(req), (request.RequestsSizer{}).Sizeof(req), pq)
				return reqCtx, req, id, true
			}
			// More space available, data was dropped.
			pq.hasMoreSpace.Signal()
		}

		// TODO: Need to change the Queue interface to return an error to allow distinguish between shutdown and context canceled.
		//  Until then use the sync.Cond.
		pq.hasMoreElements.Wait()
	}
}

// getNextItem pulls the next available item from the persistent storage along with its index. Once processing is
// finished, the index should be called with onDone to clean up the storage. If no new item is available,
// returns false.
func (pq *persistentQueue[T]) getNextItem(ctx context.Context) (uint64, T, context.Context, bool) {
	index := pq.metadata.ReadIndex
	var replayMetadataOp *storage.Operation
	if len(pq.replayItems) > 0 {
		index = pq.replayItems[0]
		pq.replayItems = pq.replayItems[1:]
		replayMetadataOp = storage.SetOperation(replayItemsKey, encodeItemIndexArray(pq.replayItems))
	} else {
		// Increase here, so even if errors happen below, it always iterates.
		pq.metadata.ReadIndex++
	}
	pq.metadata.CurrentlyDispatchedItems = append(pq.metadata.CurrentlyDispatchedItems, index)

	var req T
	restoredCtx := context.Background()
	metadataBytes, err := proto.Marshal(&pq.metadata)
	if err != nil {
		return 0, req, restoredCtx, false
	}

	getOp := storage.GetOperation(getItemKey(index))
	ops := []*storage.Operation{storage.SetOperation(metadataKey, metadataBytes)}
	if replayMetadataOp != nil {
		ops = append(ops, replayMetadataOp)
	}
	ops = append(ops, getOp)
	err = pq.client.Batch(ctx, ops...)
	if err == nil {
		restoredCtx, req, err = pq.encoding.Unmarshal(getOp.Value)
	}

	if err != nil {
		pq.logger.Debug("Failed to dispatch item", zap.Error(err))
		// We need to make sure that currently dispatched items list is cleaned
		if err = pq.itemDispatchingFinish(ctx, index); err != nil {
			pq.logger.Error("Error deleting item from queue", zap.Error(err))
		}

		return 0, req, restoredCtx, false
	}

	// Increase the reference count, so the client is not closed while the request is being processed.
	// The client cannot be closed because we hold the lock since last we checked `stopped`.
	pq.refClient++

	return index, req, restoredCtx, true
}

// onDone should be called to remove the item of the given index from the queue once processing is finished.
func (pq *persistentQueue[T]) onDone(index uint64, itemsSize, bytesSize, requestsSize int64, consumeErr error) {
	// Delete the item from the persistent storage after it was processed.
	pq.mu.Lock()
	// Always unref client even if the consumer is shutdown because we always ref it for every valid request.
	defer func() {
		if err := pq.unrefClient(context.Background()); err != nil {
			pq.logger.Error("Error closing the storage client", zap.Error(err))
		}
		pq.mu.Unlock()
	}()

	if experr.IsShutdownErr(consumeErr) {
		// The queue is shutting down, don't mark the item as dispatched, so it's picked up again after restart.
		// TODO: Handle partially delivered requests by updating their values in the storage.
		return
	}

	pq.metadata.RequestsSize -= requestsSize
	if pq.metadata.RequestsSize < 0 {
		pq.metadata.RequestsSize = 0
	}
	pq.metadata.BytesSize -= bytesSize
	if pq.metadata.BytesSize < 0 {
		pq.metadata.BytesSize = 0
	}
	pq.metadata.ItemsSize -= itemsSize
	if pq.metadata.ItemsSize < 0 {
		pq.metadata.ItemsSize = 0
	}

	if err := pq.itemDispatchingFinish(context.Background(), index); err != nil {
		pq.logger.Error("Error deleting item from queue", zap.Error(err))
	}

	// More space available after data are removed from the storage.
	pq.hasMoreSpace.Signal()
}

// retrieveAndEnqueueNotDispatchedReqs recovers legacy in-flight items using
// the configured replay order.
func (pq *persistentQueue[T]) retrieveAndEnqueueNotDispatchedReqs(ctx context.Context) {
	var dispatchedItems []uint64

	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.logger.Debug("Checking if there are items left for dispatch by consumers")
	itemKeysBuf, err := pq.client.Get(ctx, legacyCurrentlyDispatchedItemsKey)
	if err == nil {
		dispatchedItems, err = bytesToItemIndexArray(itemKeysBuf)
	}
	if err != nil {
		pq.logger.Error("Could not fetch items left for dispatch by consumers", zap.Error(err))
		return
	}

	pq.enqueueNotDispatchedReqs(ctx, dispatchedItems)
}

func (pq *persistentQueue[T]) enqueueNotDispatchedReqs(ctx context.Context, dispatchedItems []uint64) {
	if pq.replayInOrder {
		pq.replayNotDispatchedReqs(ctx, dispatchedItems)
		return
	}
	if len(dispatchedItems) == 0 {
		pq.logger.Debug("No items left for dispatch by consumers")
		return
	}

	pq.logger.Info("Fetching items left for dispatch by consumers", zap.Int(zapNumberOfItems,
		len(dispatchedItems)))
	retrieveBatch := make([]*storage.Operation, len(dispatchedItems))
	cleanupBatch := make([]*storage.Operation, len(dispatchedItems))
	for i, it := range dispatchedItems {
		key := getItemKey(it)
		retrieveBatch[i] = storage.GetOperation(key)
		cleanupBatch[i] = storage.DeleteOperation(key)
	}
	retrieveErr := pq.client.Batch(ctx, retrieveBatch...)
	cleanupErr := pq.client.Batch(ctx, cleanupBatch...)

	if cleanupErr != nil {
		pq.logger.Debug("Failed cleaning items left by consumers", zap.Error(cleanupErr))
	}

	if retrieveErr != nil {
		pq.logger.Warn("Failed retrieving items left by consumers", zap.Error(retrieveErr))
		return
	}

	errCount := 0
	for _, op := range retrieveBatch {
		if op.Value == nil {
			pq.logger.Warn("Failed retrieving item", zap.String(zapKey, op.Key), zap.Error(errValueNotSet))
			continue
		}
		reqCtx, req, err := pq.encoding.Unmarshal(op.Value)
		// If error happened or item is nil, it will be efficiently ignored
		if err != nil {
			pq.logger.Warn("Failed unmarshalling item", zap.String(zapKey, op.Key), zap.Error(err))
			continue
		}
		if pq.putInternal(reqCtx, req) != nil { //nolint:contextcheck
			errCount++
		}
	}

	if errCount > 0 {
		pq.logger.Error("Errors occurred while moving items for dispatching back to queue",
			zap.Int(zapNumberOfItems, len(retrieveBatch)), zap.Int(zapErrorCount, errCount))
	} else {
		pq.logger.Info("Moved items for dispatching back to queue",
			zap.Int(zapNumberOfItems, len(retrieveBatch)))
	}
}

// replayNotDispatchedReqs merges in-flight item indices into the persisted
// front-replay list. Existing item bodies stay at their original indices.
func (pq *persistentQueue[T]) replayNotDispatchedReqs(ctx context.Context, dispatchedItems []uint64) {
	stored, err := pq.client.Get(ctx, replayItemsKey)
	if err != nil {
		pq.startupErr = fmt.Errorf("load ordered queue replay: %w", err)
		pq.logger.Error("Could not fetch ordered replay items", zap.Error(err))
		return
	}
	existing, err := bytesToItemIndexArray(stored)
	if err != nil {
		pq.startupErr = fmt.Errorf("decode ordered queue replay: %w", err)
		pq.logger.Error("Could not decode ordered replay items", zap.Error(err))
		return
	}
	seen := make(map[uint64]struct{}, len(dispatchedItems)+len(existing))
	merged := make([]uint64, 0, len(dispatchedItems)+len(existing))
	for _, index := range append(append([]uint64(nil), dispatchedItems...), existing...) {
		if _, ok := seen[index]; ok {
			continue
		}
		seen[index] = struct{}{}
		merged = append(merged, index)
	}
	slices.Sort(merged)
	next := proto.Clone(&pq.metadata).(*PersistentMetadata)
	next.CurrentlyDispatchedItems = nil
	metadataBytes, err := proto.Marshal(next)
	if err == nil {
		err = pq.client.Set(ctx, replayItemsKey, encodeItemIndexArray(merged))
	}
	if err == nil {
		err = pq.client.Set(ctx, metadataKey, metadataBytes)
	}
	if err != nil {
		pq.startupErr = fmt.Errorf("persist ordered queue replay: %w", err)
		pq.logger.Error("Could not store ordered replay metadata", zap.Error(err))
		return
	}
	// Publish the replay list before clearing the old ownership metadata.
	// A crash between those writes leaves duplicates that merge safely.
	pq.replayItems = merged
	pq.metadata.CurrentlyDispatchedItems = nil
	if len(merged) > 0 {
		pq.logger.Info("Queued in-flight items ahead of newer requests for recovery", zap.Int(zapNumberOfItems, len(merged)))
	}
}

// itemDispatchingFinish removes the item from the list of currently dispatched items and deletes it from the persistent queue
func (pq *persistentQueue[T]) itemDispatchingFinish(ctx context.Context, index uint64) error {
	lenCDI := len(pq.metadata.CurrentlyDispatchedItems)
	for i := range lenCDI {
		if pq.metadata.CurrentlyDispatchedItems[i] == index {
			pq.metadata.CurrentlyDispatchedItems[i] = pq.metadata.CurrentlyDispatchedItems[lenCDI-1]
			pq.metadata.CurrentlyDispatchedItems = pq.metadata.CurrentlyDispatchedItems[:lenCDI-1]
			break
		}
	}

	// Ensure the used size are in sync when queue is drained.
	if pq.requestSize() == 0 {
		pq.metadata.BytesSize = 0
		pq.metadata.RequestsSize = 0
		pq.metadata.ItemsSize = 0
	}

	metadataBytes, err := proto.Marshal(&pq.metadata)
	if err != nil {
		return err
	}

	setOp := storage.SetOperation(metadataKey, metadataBytes)
	deleteOp := storage.DeleteOperation(getItemKey(index))
	err = pq.client.Batch(ctx, setOp, deleteOp)
	if err == nil {
		// Everything ok, exit
		return nil
	}

	// got an error, try to gracefully handle it
	pq.logger.Warn("Failed updating currently dispatched items, trying to delete the item first",
		zap.Error(err))

	if err = pq.client.Batch(ctx, deleteOp); err != nil {
		// Return an error here, as this indicates an issue with the underlying storage medium
		return fmt.Errorf("failed deleting item from queue, got error from storage: %w", err)
	}

	if err = pq.client.Batch(ctx, setOp); err != nil {
		// even if this fails, we still have the right dispatched items in memory
		// at worst, we'll have the wrong list in storage, and we'll discard the nonexistent items during startup
		return fmt.Errorf("failed updating currently dispatched items, but deleted item successfully: %w", err)
	}

	return nil
}

func toStorageClient(ctx context.Context, storageID component.ID, host component.Host, ownerID component.ID, signal pipeline.Signal) (storage.Client, error) {
	ext, found := host.GetExtensions()[storageID]
	if !found {
		return nil, errNoStorageClient
	}

	storageExt, ok := ext.(storage.Extension)
	if !ok {
		return nil, errWrongExtensionType
	}

	return storageExt.GetClient(ctx, component.KindExporter, ownerID, signal.String())
}

func getItemKey(index uint64) string {
	return strconv.FormatUint(index, 10)
}

func bytesToItemIndex(buf []byte) (uint64, error) {
	if buf == nil {
		return uint64(0), errValueNotSet
	}
	// The sizeof uint64 in binary is 8.
	if len(buf) < 8 {
		return 0, errInvalidValue
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func bytesToItemIndexArray(buf []byte) ([]uint64, error) {
	if len(buf) == 0 {
		return nil, nil
	}

	// The sizeof uint32 in binary is 4.
	if len(buf) < 4 {
		return nil, errInvalidValue
	}
	size := int(binary.LittleEndian.Uint32(buf))
	if size == 0 {
		return nil, nil
	}

	buf = buf[4:]
	// The sizeof uint64 in binary is 8, so we need to have size*8 bytes.
	if len(buf) < size*8 {
		return nil, errInvalidValue
	}

	val := make([]uint64, size)
	for i := range size {
		val[i] = binary.LittleEndian.Uint64(buf)
		buf = buf[8:]
	}
	return val, nil
}

func encodeItemIndexArray(indices []uint64) []byte {
	buf := make([]byte, 4+8*len(indices))
	binary.LittleEndian.PutUint32(buf, uint32(len(indices)))
	for i, index := range indices {
		binary.LittleEndian.PutUint64(buf[4+i*8:], index)
	}
	return buf
}

type indexDone struct {
	index        uint64
	itemsSize    int64
	bytesSize    int64
	requestsSize int64
	queue        interface {
		onDone(uint64, int64, int64, int64, error)
	}
}

func (id *indexDone) reset(index uint64, itemsSize, bytesSize, requestsSize int64, queue interface {
	onDone(uint64, int64, int64, int64, error)
},
) {
	id.index = index
	id.itemsSize = itemsSize
	id.bytesSize = bytesSize
	id.requestsSize = requestsSize
	id.queue = queue
}

func (id *indexDone) OnDone(err error) {
	id.queue.onDone(id.index, id.itemsSize, id.bytesSize, id.requestsSize, err)
}
