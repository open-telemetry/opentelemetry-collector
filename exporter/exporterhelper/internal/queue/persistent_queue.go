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

	readIndexKey                = "ri"
	writeIndexKey               = "wi"
	currentlyDispatchedItemsKey = "di"

	// queueMetadataKey is the new single key for all queue metadata.
	// TODO: Enable when https://github.com/open-telemetry/opentelemetry-collector/issues/12890 is done
	//nolint:unused
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
	logger      *zap.Logger
	client      storage.Client
	encoding    Encoding[T]
	capacity    int64
	sizerType   request.SizerType
	activeSizer request.Sizer[T]
	itemsSizer  request.Sizer[T]
	bytesSizer  request.Sizer[T]
	storageID   component.ID
	id          component.ID
	signal      pipeline.Signal

	// mu guards everything declared below.
	mu              sync.Mutex
	hasMoreElements *sync.Cond
	hasMoreSpace    *cond
	metadata        PersistentMetadata
	refClient       int64
	stopped         bool

	blockOnOverflow bool
}

// newPersistentQueue creates a new queue backed by file storage; name and signal must be a unique combination that identifies the queue storage
func newPersistentQueue[T any](set Settings[T]) readableQueue[T] {
	pq := &persistentQueue[T]{
		logger:          set.Telemetry.Logger,
		encoding:        set.Encoding,
		capacity:        set.Capacity,
		sizerType:       set.SizerType,
		activeSizer:     set.activeSizer(),
		itemsSizer:      set.ItemsSizer,
		bytesSizer:      set.BytesSizer,
		storageID:       *set.StorageID,
		id:              set.ID,
		signal:          set.Signal,
		blockOnOverflow: set.BlockOnOverflow,
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
		return pq.requestSize()
	}
}

func (pq *persistentQueue[T]) requestSize() int64 {
	//nolint:gosec
	return int64(pq.metadata.WriteIndex-pq.metadata.ReadIndex) + int64(len(pq.metadata.CurrentlyDispatchedItems))
}

func (pq *persistentQueue[T]) Capacity() int64 {
	return pq.capacity
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
	riOp := storage.GetOperation(readIndexKey)
	wiOp := storage.GetOperation(writeIndexKey)

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
	size := pq.activeSizer.Sizeof(req)
	for pq.internalSize()+size > pq.capacity {
		if !pq.blockOnOverflow {
			return ErrQueueIsFull
		}
		if err := pq.hasMoreSpace.Wait(ctx); err != nil {
			return err
		}
	}

	reqBuf, err := pq.encoding.Marshal(ctx, req)
	if err != nil {
		return err
	}

	// Carry out a transaction where we both add the item and update the write index
	ops := []*storage.Operation{
		storage.SetOperation(writeIndexKey, itemIndexToBytes(pq.metadata.WriteIndex+1)),
		storage.SetOperation(getItemKey(pq.metadata.WriteIndex), reqBuf),
	}
	if err = pq.client.Batch(ctx, ops...); err != nil {
		return err
	}

	pq.metadata.ItemsSize += pq.itemsSizer.Sizeof(req)
	pq.metadata.BytesSize += pq.bytesSizer.Sizeof(req)
	pq.metadata.WriteIndex++
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
			// Ensure the used size are in sync when queue is drained.
			if pq.requestSize() == 0 {
				pq.metadata.BytesSize = 0
				pq.metadata.ItemsSize = 0
			}
			if consumed {
				id := indexDonePool.Get().(*indexDone)
				id.reset(index, pq.itemsSizer.Sizeof(req), pq.bytesSizer.Sizeof(req), pq)
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
	// Increase here, so even if errors happen below, it always iterates
	pq.metadata.ReadIndex++
	pq.metadata.CurrentlyDispatchedItems = append(pq.metadata.CurrentlyDispatchedItems, index)
	getOp := storage.GetOperation(getItemKey(index))
	err := pq.client.Batch(ctx,
		storage.SetOperation(readIndexKey, itemIndexToBytes(pq.metadata.ReadIndex)),
		storage.SetOperation(currentlyDispatchedItemsKey, itemIndexArrayToBytes(pq.metadata.CurrentlyDispatchedItems)),
		getOp)

	var req T
	restoredCtx := context.Background()
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

	if err := pq.itemDispatchingFinish(context.Background(), index); err != nil {
		pq.logger.Error("Error deleting item from queue", zap.Error(err))
	}
	pq.metadata.BytesSize -= bytesSize
	if pq.metadata.BytesSize < 0 {
		pq.metadata.BytesSize = 0
	}
	pq.metadata.ItemsSize -= itemsSize
	if pq.metadata.ItemsSize < 0 {
		pq.metadata.ItemsSize = 0
	}

	// Ensure the used size are in sync when queue is drained.
	if pq.requestSize() == 0 {
		pq.metadata.BytesSize = 0
		pq.metadata.ItemsSize = 0
	}

	// More space available after data are removed from the storage.
	pq.hasMoreSpace.Signal()
}

// retrieveAndEnqueueNotDispatchedReqs gets the items for which sending was not finished, cleans the storage
// and moves the items at the back of the queue.
func (pq *persistentQueue[T]) retrieveAndEnqueueNotDispatchedReqs(ctx context.Context) {
	var dispatchedItems []uint64

	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.logger.Debug("Checking if there are items left for dispatch by consumers")
	itemKeysBuf, err := pq.client.Get(ctx, currentlyDispatchedItemsKey)
	if err == nil {
		dispatchedItems, err = bytesToItemIndexArray(itemKeysBuf)
	}
	if err != nil {
		pq.logger.Error("Could not fetch items left for dispatch by consumers", zap.Error(err))
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

	setOp := storage.SetOperation(currentlyDispatchedItemsKey, itemIndexArrayToBytes(pq.metadata.CurrentlyDispatchedItems))
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
},
) {
	id.index = index
	id.itemsSize = itemsSize
	id.bytesSize = bytesSize
	id.queue = queue
}

func (id *indexDone) OnDone(err error) {
	id.queue.onDone(id.index, id.itemsSize, id.bytesSize, err)
}
