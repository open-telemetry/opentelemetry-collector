// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"go.uber.org/zap"

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

	// legacy keys for backward compatibility with old versions of the queue.
	legacyReadIndexKey                = "ri"
	legacyWriteIndexKey               = "wi"
	legacyCurrentlyDispatchedItemsKey = "di"
	legacyQueueSizeKey                = "si"

	// queueMetadataKey is the new single key for all queue metadata.
	queueMetadataKey = "qmv0"
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

type persistentQueueSettings[T any] struct {
	sizer           request.Sizer[T]
	sizerType       request.SizerType
	availableSizers map[request.SizerType]request.Sizer[T]
	capacity        int64
	blockOnOverflow bool
	signal          pipeline.Signal
	storageID       component.ID
	encoding        Encoding[T]
	id              component.ID
	telemetry       component.TelemetrySettings
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
type persistentQueue[T any] struct {
	set    persistentQueueSettings[T]
	logger *zap.Logger
	client storage.Client

	// mu guards everything declared below.
	mu              sync.Mutex
	hasMoreElements *sync.Cond
	hasMoreSpace    *cond
	metadata        PersistentMetadata
	refClient       int64
	stopped         bool

	legacyWriteIndex uint64 // legacy write index for backward compatibility
}

// newPersistentQueue creates a new queue backed by file storage; name and signal must be a unique combination that identifies the queue storage
func newPersistentQueue[T any](set persistentQueueSettings[T]) readableQueue[T] {
	pq := &persistentQueue[T]{
		set:    set,
		logger: set.telemetry.Logger,
	}
	pq.hasMoreElements = sync.NewCond(&pq.mu)
	pq.hasMoreSpace = newCond(&pq.mu)
	return pq
}

// Start starts the persistentQueue with the given number of consumers.
func (pq *persistentQueue[T]) Start(ctx context.Context, host component.Host) error {
	storageClient, err := toStorageClient(ctx, pq.set.storageID, host, pq.set.id, pq.set.signal)
	if err != nil {
		return err
	}
	pq.initClient(ctx, storageClient)
	return nil
}

func (pq *persistentQueue[T]) Size() int64 {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return pq.internalSize()
}

func (pq *persistentQueue[T]) internalSize() int64 {
	switch pq.set.sizerType {
	case request.SizerTypeBytes:
		return pq.metadata.BytesSize + pq.legacySize()
	case request.SizerTypeItems:
		return pq.metadata.ItemsSize + pq.legacySize()
	default:
		return pq.requestsSize() + pq.legacySize()
	}
}

func (pq *persistentQueue[T]) legacySize() int64 {
	//nolint:gosec
	diff := int64(pq.legacyWriteIndex - pq.metadata.ReadIndex)
	if diff > 0 {
		return diff
	}
	return 0
}

func (pq *persistentQueue[T]) requestsSize() int64 {
	//nolint:gosec
	return int64(pq.metadata.WriteIndex-pq.metadata.ReadIndex) + int64(len(pq.metadata.CurrentlyDispatchedItems))
}

func (pq *persistentQueue[T]) itemsSize(req T) int64 {
	return pq.set.availableSizers[request.SizerTypeItems].Sizeof(req)
}

func (pq *persistentQueue[T]) bytesSize(req T) int64 {
	return pq.set.availableSizers[request.SizerTypeBytes].Sizeof(req)
}

func (pq *persistentQueue[T]) Capacity() int64 {
	return pq.set.capacity
}

func (pq *persistentQueue[T]) initClient(ctx context.Context, client storage.Client) {
	pq.client = client
	// Start with a reference 1 which is the reference we use for the producer goroutines and initialization.
	pq.refClient = 1
	pq.initPersistentContiguousStorage(ctx)
	// Make sure the leftover requests are handled
	pq.retrieveAndEnqueueNotDispatchedReqs(ctx)
}

func (pq *persistentQueue[T]) initPersistentContiguousStorage(ctx context.Context) {
	// 1. Try to load from new consolidated metadata first
	err := pq.loadQueueMetadata(ctx)
	switch {
	case err == nil:
		pq.logger.Info("Successfully loaded queue metadata.")
		return
	case !errors.Is(err, errValueNotSet):
		pq.logger.Error("Unable to retrieve queue metadata from storage, non-missing value error occurred", zap.Error(err))
		return
	default:
		pq.logger.Info("New queue metadata key not found, attempting to load legacy format.")
	}

	// TODO: Remove legacy format support after 6 months (target: December 2025)
	// 2. Fallback to legacy individual keys for backward compatibility
	pq.logger.Info("Loading queue metadata from legacy format")
	riOp := storage.GetOperation(legacyReadIndexKey)
	wiOp := storage.GetOperation(legacyWriteIndexKey)

	err = pq.client.Batch(ctx, riOp, wiOp)
	if err == nil {
		pq.metadata.ReadIndex, err = bytesToItemIndex(riOp.Value)
	}

	if err == nil {
		pq.metadata.WriteIndex, err = bytesToItemIndex(wiOp.Value)
		pq.legacyWriteIndex, err = bytesToItemIndex(wiOp.Value)
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

	// Load legacy dispatched items
	var itemKeysBuf []byte
	if itemKeysBuf, err = pq.client.Get(ctx, legacyCurrentlyDispatchedItemsKey); err == nil {
		var dispatchedItems []uint64
		if dispatchedItems, err = bytesToItemIndexArray(itemKeysBuf); err == nil {
			pq.metadata.CurrentlyDispatchedItems = dispatchedItems
		}
	}

	// 3. Save to a new format and clean up legacy keys
	if err := pq.backupCurrentMetadata(ctx); err != nil {
		pq.logger.Error("Failed to persist migrated queue metadata", zap.Error(err))
		return
	}
	pq.cleanupLegacyKeys(ctx)
}

// loadQueueMetadata loads queue metadata from the consolidated key
func (pq *persistentQueue[T]) loadQueueMetadata(ctx context.Context) error {
	buf, err := pq.client.Get(ctx, queueMetadataKey)
	if err != nil {
		return err
	}

	if len(buf) == 0 {
		return errValueNotSet
	}

	metadata := &pq.metadata
	if err = metadata.Unmarshal(buf); err != nil {
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

// cleanupLegacyKeys removes the old individual metadata keys
func (pq *persistentQueue[T]) cleanupLegacyKeys(ctx context.Context) {
	ops := []*storage.Operation{
		storage.DeleteOperation(legacyReadIndexKey),
		storage.DeleteOperation(legacyWriteIndexKey),
		storage.DeleteOperation(legacyCurrentlyDispatchedItemsKey),
		storage.DeleteOperation(legacyQueueSizeKey),
	}

	if err := pq.client.Batch(ctx, ops...); err != nil {
		pq.logger.Warn("Failed to cleanup legacy metadata keys", zap.Error(err))
	} else {
		pq.logger.Info("Successfully migrated to consolidated metadata format")
	}
}

// backupCurrentMetadata is used for standalone metadata persistence like in Shutdown or initialization.
func (pq *persistentQueue[T]) backupCurrentMetadata(ctx context.Context) error {
	metadataBytes, err := metadataToBytes(&pq.metadata)
	if err != nil {
		return err
	}

	if err := pq.client.Set(ctx, queueMetadataKey, metadataBytes); err != nil {
		pq.logger.Error("Failed to persist current metadata to storage", zap.Error(err))
		return err
	}
	return nil
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
	return pq.putInternal(ctx, req)
}

// putInternal is the internal version that requires caller to hold the mutex lock.
func (pq *persistentQueue[T]) putInternal(ctx context.Context, req T) error {
	size := pq.set.sizer.Sizeof(req)
	for pq.internalSize()+size > pq.set.capacity {
		if !pq.set.blockOnOverflow {
			return ErrQueueIsFull
		}
		if err := pq.hasMoreSpace.Wait(ctx); err != nil {
			return err
		}
	}

	reqBuf, err := pq.set.encoding.Marshal(ctx, req)
	if err != nil {
		return err
	}

	itemIndex := pq.metadata.WriteIndex
	pq.metadata.WriteIndex++
	pq.metadata.ItemsSize += pq.itemsSize(req)
	pq.metadata.BytesSize += pq.bytesSize(req)

	metadataBytes, err := metadataToBytes(&pq.metadata)
	if err != nil {
		return err
	}

	// Carry out a transaction where we both add the item and update the metadata.
	ops := []*storage.Operation{
		storage.SetOperation(queueMetadataKey, metadataBytes),
		storage.SetOperation(getItemKey(itemIndex), reqBuf),
	}
	if err = pq.client.Batch(ctx, ops...); err != nil {
		pq.metadata.WriteIndex--
		pq.metadata.ItemsSize -= pq.itemsSize(req)
		pq.metadata.BytesSize -= pq.bytesSize(req)
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
		for pq.metadata.ReadIndex != pq.metadata.WriteIndex {
			index, req, reqCtx, consumed := pq.getNextItem(ctx)
			// Ensure the used size and the metadata sizes are in sync.
			if pq.requestsSize() == 0 {
				pq.metadata.BytesSize = 0
				pq.metadata.ItemsSize = 0
				pq.hasMoreSpace.Signal()
			}
			if consumed {
				id := indexDonePool.Get().(*indexDone)
				itemsSize := pq.itemsSize(req)
				bytesSize := pq.bytesSize(req)

				id.reset(index, itemsSize, bytesSize, pq)
				return reqCtx, req, id, true
			}
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
	// Increase here, so even if errors happen below, it always iterates
	pq.metadata.ReadIndex++
	pq.metadata.CurrentlyDispatchedItems = append(pq.metadata.CurrentlyDispatchedItems, index)

	metadataBytes, err := metadataToBytes(&pq.metadata)
	var req T
	restoredCtx := context.Background()

	if err != nil {
		return 0, req, restoredCtx, false
	}

	getOp := storage.GetOperation(getItemKey(index))
	err = pq.client.Batch(ctx,
		storage.SetOperation(queueMetadataKey, metadataBytes),
		getOp)

	if err == nil {
		restoredCtx, req, err = pq.set.encoding.Unmarshal(getOp.Value)
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
func (pq *persistentQueue[T]) onDone(index uint64, itemsSize int64, bytesSize int64, consumeErr error) {
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

	// Legacy data doesn't track itemsSize and bytesSize, so we only decrement when not in legacy mode.
	if pq.legacySize() == 0 {
		pq.metadata.BytesSize -= bytesSize
		pq.metadata.ItemsSize -= itemsSize
	}
	pq.hasMoreSpace.Signal()

	// itemDispatchingFinish will persist the new metadata (including the updated queue size) to storage.
	if err := pq.itemDispatchingFinish(context.Background(), index); err != nil {
		pq.logger.Error("Error deleting item from queue", zap.Error(err))
	}
}

// retrieveAndEnqueueNotDispatchedReqs gets the items for which sending was not finished, cleans the storage
// and moves the items at the back of the queue.
func (pq *persistentQueue[T]) retrieveAndEnqueueNotDispatchedReqs(ctx context.Context) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.logger.Debug("Checking if there are items left for dispatch by consumers")

	dispatchedItems := pq.metadata.CurrentlyDispatchedItems

	if len(dispatchedItems) == 0 {
		pq.logger.Debug("No items left for dispatch by consumers")
		return
	}

	pq.logger.Info("Fetching items left for dispatch by consumers", zap.Int(zapNumberOfItems,
		len(dispatchedItems)))
	pq.metadata.CurrentlyDispatchedItems = pq.metadata.CurrentlyDispatchedItems[:0]

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
		reqCtx, req, err := pq.set.encoding.Unmarshal(op.Value)
		// Subtract the item size from the queue size before re-enqueuing to avoid double counting.
		// Legacy data doesn't track itemsSize and bytesSize, so we only decrement when not in legacy mode.
		if pq.legacySize() == 0 {
			pq.metadata.ItemsSize -= pq.itemsSize(req)
			pq.metadata.BytesSize -= pq.bytesSize(req)
		}

		// If error happened or item is nil, it will be efficiently ignored
		if err != nil {
			pq.logger.Warn("Failed unmarshalling item", zap.String(zapKey, op.Key), zap.Error(err))
			continue
		}
		if pq.putInternal(reqCtx, req) != nil {
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

// itemDispatchingFinish removes the item from the list of currently dispatched items and deletes it from the persistent queue
func (pq *persistentQueue[T]) itemDispatchingFinish(ctx context.Context, index uint64) error {
	lenCDI := len(pq.metadata.CurrentlyDispatchedItems)
	for i := 0; i < lenCDI; i++ {
		if pq.metadata.CurrentlyDispatchedItems[i] == index {
			pq.metadata.CurrentlyDispatchedItems[i] = pq.metadata.CurrentlyDispatchedItems[lenCDI-1]
			pq.metadata.CurrentlyDispatchedItems = pq.metadata.CurrentlyDispatchedItems[:lenCDI-1]
			break
		}
	}

	metadataBytes, err := metadataToBytes(&pq.metadata)
	if err != nil {
		return err
	}

	setOp := storage.SetOperation(queueMetadataKey, metadataBytes)
	deleteOp := storage.DeleteOperation(getItemKey(index))
	if err := pq.client.Batch(ctx, setOp, deleteOp); err != nil {
		// got an error, try to gracefully handle it
		pq.logger.Warn("Failed updating currently dispatched items, trying to delete the item first",
			zap.Error(err))
	} else {
		// Everything ok, exit
		return nil
	}

	if err := pq.client.Batch(ctx, deleteOp); err != nil {
		// Return an error here, as this indicates an issue with the underlying storage medium
		return fmt.Errorf("failed deleting item from queue, got error from storage: %w", err)
	}

	if err := pq.client.Batch(ctx, setOp); err != nil {
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

func itemIndexToBytes(value uint64) []byte {
	return binary.LittleEndian.AppendUint64([]byte{}, value)
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

func itemIndexArrayToBytes(arr []uint64) []byte {
	size := len(arr)
	buf := make([]byte, 0, 4+size*8)
	//nolint:gosec
	buf = binary.LittleEndian.AppendUint32(buf, uint32(size))
	for _, item := range arr {
		buf = binary.LittleEndian.AppendUint64(buf, item)
	}
	return buf
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
	for i := 0; i < size; i++ {
		val[i] = binary.LittleEndian.Uint64(buf)
		buf = buf[8:]
	}
	return val, nil
}

func metadataToBytes(meta *PersistentMetadata) ([]byte, error) {
	return meta.Marshal()
}

type indexDone struct {
	index     uint64
	itemsSize int64
	bytesSize int64
	queue     interface {
		onDone(uint64, int64, int64, error)
	}
}

func (id *indexDone) reset(index uint64, itemsSize, bytesSize int64, queue interface {
	onDone(uint64, int64, int64, error)
}) {
	id.index = index
	id.itemsSize = itemsSize
	id.bytesSize = bytesSize
	id.queue = queue
}

func (id *indexDone) OnDone(err error) {
	id.queue.onDone(id.index, id.itemsSize, id.bytesSize, err)
}
