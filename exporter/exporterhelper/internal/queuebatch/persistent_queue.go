// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

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

	// queueMetadataKey is the new single key for all queue metadata.
	queueMetadataKey = "qm"

	// Old keys, kept for backward compatibility during loading.
	oldReadIndexKey                = "ri"
	oldWriteIndexKey               = "wi"
	oldCurrentlyDispatchedItemsKey = "di"
	oldQueueSizeKey                = "si"
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
	capacity        int64
	blockOnOverflow bool
	signal          pipeline.Signal
	storageID       component.ID
	encoding        Encoding[T]
	id              component.ID
	telemetry       component.TelemetrySettings
}

// queueMetadata holds all persistent metadata for the queue.
type queueMetadata struct {
	ReadIndex                uint64
	WriteIndex               uint64
	QueueSize                int64 // Represents actual size if not isRequestSized, otherwise calculated
	CurrentlyDispatchedItems []uint64
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

	// isRequestSized indicates whether the queue is sized by the number of requests.
	isRequestSized bool

	// mu guards everything declared below.
	mu                       sync.Mutex
	hasMoreElements          *sync.Cond
	hasMoreSpace             *cond
	readIndex                uint64
	writeIndex               uint64
	currentlyDispatchedItems []uint64
	queueSize                int64
	refClient                int64
	stopped                  bool
}

// newPersistentQueue creates a new queue backed by file storage; name and signal must be a unique combination that identifies the queue storage
func newPersistentQueue[T any](set persistentQueueSettings[T]) readableQueue[T] {
	_, isRequestSized := set.sizer.(request.RequestsSizer[T])
	pq := &persistentQueue[T]{
		set:            set,
		logger:         set.telemetry.Logger,
		isRequestSized: isRequestSized,
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
	return pq.queueSize
}

func (pq *persistentQueue[T]) Capacity() int64 {
	return pq.set.capacity
}

func (pq *persistentQueue[T]) initClient(ctx context.Context, client storage.Client) {
	pq.client = client
	// Start with a reference 1 which is the reference we use for the producer goroutines and initialization.
	pq.refClient = 1
	// pq.mu must be held before calling initPersistentContiguousStorage and retrieveAndEnqueueNotDispatchedReqs
	pq.mu.Lock()
	pq.initPersistentContiguousStorage(ctx)
	// Make sure the leftover requests are handled
	pq.retrieveAndEnqueueNotDispatchedReqs(ctx)
	pq.mu.Unlock()
}

func (pq *persistentQueue[T]) initPersistentContiguousStorage(ctx context.Context) {
	// This function must be called with pq.mu held.
	loadedFromStorage := false
	migratedFromOldFormat := false

	// 1. Try to load new metadata format
	metaBytes, err := pq.client.Get(ctx, queueMetadataKey)
	switch {
	case err == nil:
		meta, umErr := unmarshalQueueMetadata(metaBytes)
		if umErr == nil {
			pq.readIndex = meta.ReadIndex
			pq.writeIndex = meta.WriteIndex
			pq.currentlyDispatchedItems = meta.CurrentlyDispatchedItems
			if pq.currentlyDispatchedItems == nil { // Ensure it's an empty slice, not nil
				pq.currentlyDispatchedItems = []uint64{}
			}

			if pq.isRequestSized {
				//nolint:gosec
				pq.queueSize = int64(pq.writeIndex - pq.readIndex)
				if pq.queueSize < 0 {
					pq.queueSize = 0
				}
			} else {
				pq.queueSize = meta.QueueSize
			}
			pq.logger.Info("Successfully loaded queue metadata from new format.")
			loadedFromStorage = true
		} else {
			pq.logger.Error("Failed to unmarshal new queue metadata, attempting to load old format.", zap.Error(umErr))
		}
	case !errors.Is(err, errValueNotSet):
		pq.logger.Error("Failed to get new queue metadata from storage (and not 'value not set'), attempting to load old format.", zap.Error(err))
	default:
		pq.logger.Info("New queue metadata key not found, attempting to load old format.")
	}

	// 2. If new format failed, try to load old metadata format
	if !loadedFromStorage {
		pq.logger.Info("Attempting to load queue metadata from old format.")
		riOp := storage.GetOperation(oldReadIndexKey)
		wiOp := storage.GetOperation(oldWriteIndexKey)
		cdiOp := storage.GetOperation(oldCurrentlyDispatchedItemsKey)
		qsOp := storage.GetOperation(oldQueueSizeKey) // Will only be used if !pq.isRequestSized

		opsToBatch := []*storage.Operation{riOp, wiOp, cdiOp}
		if !pq.isRequestSized { // Only fetch queueSize if we are not calculating it by request count
			opsToBatch = append(opsToBatch, qsOp)
		}

		batchErr := pq.client.Batch(ctx, opsToBatch...)
		if batchErr == nil {
			var riErr, wiErr, cdiErr, qsErr error
			pq.readIndex, riErr = bytesToItemIndex(riOp.Value)
			pq.writeIndex, wiErr = bytesToItemIndex(wiOp.Value)
			pq.currentlyDispatchedItems, cdiErr = bytesToItemIndexArray(cdiOp.Value)
			if pq.currentlyDispatchedItems == nil { // Ensure it's an empty slice, not nil
				pq.currentlyDispatchedItems = []uint64{}
			}

			// Todo: if-else chain
			if riErr == nil && wiErr == nil { // cdiErr can be errValueNotSet for an empty list
				//nolint:gosec
				queueSize := int64(pq.writeIndex - pq.readIndex)
				if !pq.isRequestSized {
					var oldQsVal uint64
					oldQsVal, qsErr = bytesToItemIndex(qsOp.Value)

					switch {
					case qsErr == nil:
						//nolint:gosec
						pq.queueSize = int64(oldQsVal)
					case errors.Is(qsErr, errValueNotSet):
						// If not set, calculate from indices, but log it.
						pq.logger.Warn("Old queue size key not found, calculating from indices.", zap.Error(qsErr))
						pq.queueSize = queueSize
					default:
						pq.logger.Error("Failed to read old queue size, calculating from indices.", zap.Error(qsErr))
						pq.queueSize = queueSize
					}
				} else {
					pq.queueSize = queueSize
				}
				if pq.queueSize < 0 {
					pq.queueSize = 0
				}

				pq.logger.Info("Successfully loaded queue metadata from old format.")
				loadedFromStorage = true
				migratedFromOldFormat = true
			} else {
				pq.logger.Error("Failed to parse critical old metadata (read/write index). Initializing as new queue.",
					zap.Error(riErr), zap.Error(wiErr), zap.Error(cdiErr))
			}
		} else {
			pq.logger.Error("Failed to batch load old queue metadata. Initializing as new queue.", zap.Error(batchErr))
		}
	}

	// 3. If nothing loaded, initialize as new.
	if !loadedFromStorage {
		pq.logger.Info("Initializing new persistent queue (no valid old or new metadata found).")
		pq.readIndex = 0
		pq.writeIndex = 0
		pq.queueSize = 0
		pq.currentlyDispatchedItems = []uint64{}
	}

	// 4. Persist metadata in new format if it was newly initialized or migrated.
	if !loadedFromStorage || migratedFromOldFormat {
		pq.logger.Info("Persisting initial/migrated queue metadata in new format.")
		if err := pq.persistCurrentMetadata(ctx); err != nil {
			pq.logger.Error("Failed to persist initial/migrated queue metadata", zap.Error(err))
			// This is a critical error, as the queue state might not be saved.
		}
		if migratedFromOldFormat {
			// Optionally, delete old keys after successful migration
			// For safety, this might be a manual step or a separate utility.
			// Here, we'll log that migration occurred.
			pq.logger.Info("Migration from old metadata format complete. Old keys are no longer used for writing but were read for this migration.")
		}
	}
}

// marshalCurrentMetadata constructs metadata from current pq state and returns the marshaled bytes.
// This is a helper and does not perform the Set operation itself.
// pq.mu must be held.
func (pq *persistentQueue[T]) marshalCurrentMetadata() ([]byte, error) {
	meta := queueMetadata{
		ReadIndex:                pq.readIndex,
		WriteIndex:               pq.writeIndex,
		CurrentlyDispatchedItems: pq.currentlyDispatchedItems,
	}
	if !pq.isRequestSized {
		meta.QueueSize = pq.queueSize
	} else {
		//nolint:gosec
		calculatedSize := int64(pq.writeIndex - pq.readIndex)
		if calculatedSize < 0 {
			calculatedSize = 0
		}
		meta.QueueSize = calculatedSize
	}

	estimatedBufSize := 8 + 8 + 8 + 4 + (len(pq.currentlyDispatchedItems) * 8)
	buf := make([]byte, 0, estimatedBufSize)
	return marshalQueueMetadata(&meta, buf)
}

// persistCurrentMetadata is used for standalone metadata persistence like in Shutdown or initialization.
// pq.mu must be held.
func (pq *persistentQueue[T]) persistCurrentMetadata(ctx context.Context) error {
	metadataBytes, err := pq.marshalCurrentMetadata()
	if err != nil {
		pq.logger.Error("Failed to marshal current metadata for persistence", zap.Error(err))
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

	var backupErr error
	// Persist final state if not already stopped and client exists.
	if !pq.stopped && pq.client != nil {
		pq.logger.Info("Shutting down persistent queue, persisting final metadata.")
		backupErr = pq.persistCurrentMetadata(ctx)
		if backupErr != nil {
			pq.logger.Error("Failed to persist metadata during shutdown", zap.Error(backupErr))
		}
	}

	// Mark this queue as stopped, so consumer don't start any more work.
	pq.stopped = true
	pq.hasMoreElements.Broadcast()
	return errors.Join(backupErr, pq.unrefClient(ctx))
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

// marshalQueueMetadata serializes the queue metadata.
func marshalQueueMetadata(meta *queueMetadata, buf []byte) ([]byte, error) {
	buf = buf[:0] // Clear the buffer before use if it's being reused
	buf = binary.LittleEndian.AppendUint64(buf, meta.ReadIndex)
	buf = binary.LittleEndian.AppendUint64(buf, meta.WriteIndex)
	//nolint:gosec
	buf = binary.LittleEndian.AppendUint64(buf, uint64(meta.QueueSize))

	cdiLen := len(meta.CurrentlyDispatchedItems)
	//nolint:gosec
	buf = binary.LittleEndian.AppendUint32(buf, uint32(cdiLen))
	for _, item := range meta.CurrentlyDispatchedItems {
		buf = binary.LittleEndian.AppendUint64(buf, item)
	}
	return buf, nil
}

// unmarshalQueueMetadata deserializes the queue metadata.
func unmarshalQueueMetadata(data []byte) (*queueMetadata, error) {
	if data == nil {
		return nil, errValueNotSet
	}
	// Minimum size: ReadIndex(8) + WriteIndex(8) + QueueSize(8) + CDILen(4) = 28
	if len(data) < 28 {
		return nil, fmt.Errorf("queue metadata too short: %d bytes, expected at least 28", len(data))
	}

	meta := &queueMetadata{}
	offset := 0
	meta.ReadIndex = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	meta.WriteIndex = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	queueSize := binary.LittleEndian.Uint64(data[offset : offset+8])
	//nolint:gosec
	meta.QueueSize = int64(queueSize)
	offset += 8
	cdiLen := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4

	if cdiLen < 0 {
		return nil, fmt.Errorf("invalid negative length for currently dispatched items: %d", cdiLen)
	}

	expectedCDIBytes := cdiLen * 8
	if len(data)-offset < expectedCDIBytes {
		return nil, fmt.Errorf("queue metadata too short for currently dispatched items: got %d, want %d", len(data)-offset, expectedCDIBytes)
	}

	if cdiLen == 0 {
		meta.CurrentlyDispatchedItems = []uint64{}
	} else {
		meta.CurrentlyDispatchedItems = make([]uint64, cdiLen)
		for i := 0; i < cdiLen; i++ {
			meta.CurrentlyDispatchedItems[i] = binary.LittleEndian.Uint64(data[offset : offset+8])
			offset += 8
		}
	}
	return meta, nil
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
	reqSize := pq.set.sizer.Sizeof(req)
	for pq.queueSize+reqSize > pq.set.capacity {
		if !pq.set.blockOnOverflow {
			return ErrQueueIsFull
		}
		if err := pq.hasMoreSpace.Wait(ctx); err != nil {
			return err
		}
	}

	reqBuf, err := pq.set.encoding.Marshal(req)
	if err != nil {
		return err
	}

	// Prepare metadata for the state *after* this operation succeeds.
	pendingWriteIndex := pq.writeIndex + 1
	pendingQueueSize := pq.queueSize + reqSize

	metaToPersist := queueMetadata{
		ReadIndex:                pq.readIndex,
		WriteIndex:               pendingWriteIndex,
		CurrentlyDispatchedItems: pq.currentlyDispatchedItems, // Does not change on put
	}
	if !pq.isRequestSized {
		metaToPersist.QueueSize = pendingQueueSize
	} else {
		//nolint:gosec
		calculatedSize := int64(pendingWriteIndex - pq.readIndex)
		if calculatedSize < 0 {
			calculatedSize = 0
		}
		metaToPersist.QueueSize = calculatedSize
	}

	// Marshal the pending state for persistence
	estimatedBufSize := 8 + 8 + 8 + 4 + (len(metaToPersist.CurrentlyDispatchedItems) * 8)
	metadataBytesForPersistence := make([]byte, 0, estimatedBufSize)
	metadataBytesForPersistence, err = marshalQueueMetadata(&metaToPersist, metadataBytesForPersistence)
	if err != nil {
		pq.logger.Error("Failed to marshal pending metadata for put operation", zap.Error(err))
		return fmt.Errorf("failed to marshal pending metadata for put: %w", err)
	}

	ops := []*storage.Operation{
		storage.SetOperation(getItemKey(pq.writeIndex), reqBuf), // Item uses current writeIndex
		storage.SetOperation(queueMetadataKey, metadataBytesForPersistence),
	}

	if err = pq.client.Batch(ctx, ops...); err != nil {
		pq.logger.Error("Failed to batch write item and metadata", zap.Error(err))
		return err
	}

	// If batch succeeded, update in-memory state
	pq.writeIndex = pendingWriteIndex
	pq.queueSize = pendingQueueSize
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
		for pq.readIndex != pq.writeIndex {
			index, req, consumed := pq.getNextItem(ctx)
			// Ensure the used size and the channel size are in sync.
			if pq.readIndex == pq.writeIndex {
				pq.queueSize = 0
				pq.hasMoreSpace.Signal()
			}
			if consumed {
				id := indexDonePool.Get().(*indexDone)
				id.reset(index, pq.set.sizer.Sizeof(req), pq)
				return context.Background(), req, id, true
			}
		}

		// TODO: Need to change the Queue interface to return an error to allow distinguish between shutdown and context canceled.
		//  Until then use the sync.Cond.
		pq.hasMoreElements.Wait()
	}
}

// getNextItem pulls the next available item from the persistent storage along with its index.
func (pq *persistentQueue[T]) getNextItem(ctx context.Context) (uint64, T, bool) {
	// pq.mu is held
	indexToDispatch := pq.readIndex

	pendingReadIndex := pq.readIndex + 1
	// Create a new slice for currentlyDispatchedItems to avoid modifying the existing one before persistence
	var pendingDispatchedItems []uint64
	if pq.currentlyDispatchedItems == nil {
		pendingDispatchedItems = make([]uint64, 1)
		pendingDispatchedItems[0] = indexToDispatch
	} else {
		pendingDispatchedItems = make([]uint64, len(pq.currentlyDispatchedItems)+1)
		copy(pendingDispatchedItems, pq.currentlyDispatchedItems)
		pendingDispatchedItems[len(pendingDispatchedItems)-1] = indexToDispatch
	}

	metaToPersist := queueMetadata{
		ReadIndex:                pendingReadIndex,
		WriteIndex:               pq.writeIndex,
		CurrentlyDispatchedItems: pendingDispatchedItems,
	}
	if !pq.isRequestSized {
		metaToPersist.QueueSize = pq.queueSize
	} else {
		//nolint:gosec
		calculatedSize := int64(pq.writeIndex - pendingReadIndex)
		if calculatedSize < 0 {
			calculatedSize = 0
		}
		metaToPersist.QueueSize = calculatedSize
	}

	estimatedBufSize := 8 + 8 + 8 + 4 + (len(pendingDispatchedItems) * 8)
	metadataBytesBuf := make([]byte, 0, estimatedBufSize)
	metadataBytes, err := marshalQueueMetadata(&metaToPersist, metadataBytesBuf)
	if err != nil {
		pq.logger.Error("Failed to marshal metadata for getNextItem operation", zap.Error(err))
		var emptyReq T
		return 0, emptyReq, false
	}

	getOp := storage.GetOperation(getItemKey(indexToDispatch))
	batchOps := []*storage.Operation{
		storage.SetOperation(queueMetadataKey, metadataBytes),
		getOp,
	}

	batchErr := pq.client.Batch(ctx, batchOps...)

	var request T
	var unmarshalErr error
	if batchErr == nil {
		if getOp.Value == nil {
			unmarshalErr = fmt.Errorf("item %d not found in storage (value is nil after batch get)", indexToDispatch)
		} else {
			request, unmarshalErr = pq.set.encoding.Unmarshal(getOp.Value)
		}
	}

	finalErr := batchErr
	if finalErr == nil {
		finalErr = unmarshalErr
	}

	if finalErr != nil {
		pq.logger.Debug("Failed to dispatch item", zap.Uint64("index", indexToDispatch), zap.Error(finalErr))
		var emptyReq T
		return 0, emptyReq, false
	}

	// If batch and unmarshal succeeded, update in-memory state
	pq.readIndex = pendingReadIndex
	pq.currentlyDispatchedItems = pendingDispatchedItems
	pq.refClient++
	return indexToDispatch, request, true
}

// onDone should be called to remove the item of the given index from the queue once processing is finished.
func (pq *persistentQueue[T]) onDone(index uint64, elSize int64, consumeErr error) {
	// Delete the item from the persistent storage after it was processed.
	pq.mu.Lock()
	// Always unref client even if the consumer is shutdown because we always ref it for every valid request.
	defer func() {
		if err := pq.unrefClient(context.Background()); err != nil {
			pq.logger.Error("Error closing the storage client", zap.Error(err))
		}
		pq.mu.Unlock()
	}()

	pq.queueSize -= elSize
	if pq.queueSize < 0 {
		pq.queueSize = 0
	}
	pq.hasMoreSpace.Signal()

	if experr.IsShutdownErr(consumeErr) {
		// The queue is shutting down, don't mark the item as dispatched, so it's picked up again after restart.
		// TODO: Handle partially delivered requests by updating their values in the storage.
		return
	}

	// itemDispatchingFinish will persist the new metadata (including the updated queue size) to storage.
	// and delete the item from the persistent queue.
	if err := pq.itemDispatchingFinish(context.Background(), index); err != nil {
		pq.logger.Error("Error deleting item from queue", zap.Error(err))
	}

	// The periodic backup every 10 writes is removed.
}

// retrieveAndEnqueueNotDispatchedReqs gets the items for which sending was not finished
func (pq *persistentQueue[T]) retrieveAndEnqueueNotDispatchedReqs(ctx context.Context) {
	// pq.mu must be held
	if len(pq.currentlyDispatchedItems) == 0 {
		pq.logger.Debug("No items left for dispatch by consumers based on loaded metadata.")
		return
	}

	itemsToProcess := make([]uint64, len(pq.currentlyDispatchedItems))
	copy(itemsToProcess, pq.currentlyDispatchedItems)

	pq.logger.Info("Processing items potentially left for dispatch by consumers", zap.Int(zapNumberOfItems, len(itemsToProcess)))

	retrieveOps := make([]*storage.Operation, len(itemsToProcess))
	for i, itemIndex := range itemsToProcess {
		retrieveOps[i] = storage.GetOperation(getItemKey(itemIndex))
	}

	if err := pq.client.Batch(ctx, retrieveOps...); err != nil {
		pq.logger.Error("Failed to batch retrieve items left by consumers during startup. Some items may not be re-enqueued.", zap.Error(err))
	}

	var successfullyReEnqueuedIndices []uint64
	var failedOrBadItemIndices []uint64

	for i, itemIndex := range itemsToProcess {
		op := retrieveOps[i]
		itemKey := getItemKey(itemIndex)

		if op.Value == nil {
			pq.logger.Warn("Failed retrieving item or item was nil during startup processing, will be removed from dispatched list.", zap.String(zapKey, itemKey), zap.Error(errValueNotSet))
			failedOrBadItemIndices = append(failedOrBadItemIndices, itemIndex)
			continue
		}

		req, err := pq.set.encoding.Unmarshal(op.Value)
		if err != nil {
			pq.logger.Warn("Failed unmarshalling item during startup processing, item will be dropped and removed from dispatched list.", zap.String(zapKey, itemKey), zap.Error(err))
			failedOrBadItemIndices = append(failedOrBadItemIndices, itemIndex)
			continue
		}

		if err := pq.putInternal(ctx, req); err != nil {
			pq.logger.Error("Error re-enqueueing item during startup processing. Item remains in dispatched list for now.", zap.String(zapKey, itemKey), zap.Error(err))
		} else {
			successfullyReEnqueuedIndices = append(successfullyReEnqueuedIndices, itemIndex)
		}
	}

	newCurrentlyDispatched := make([]uint64, 0, len(pq.currentlyDispatchedItems))
originalLoop:
	for _, originalDispatchedIdx := range pq.currentlyDispatchedItems {
		for _, reEnqueuedIdx := range successfullyReEnqueuedIndices {
			if originalDispatchedIdx == reEnqueuedIdx {
				continue originalLoop
			}
		}
		for _, badIdx := range failedOrBadItemIndices {
			if originalDispatchedIdx == badIdx {
				continue originalLoop
			}
		}
		newCurrentlyDispatched = append(newCurrentlyDispatched, originalDispatchedIdx)
	}
	pq.currentlyDispatchedItems = newCurrentlyDispatched

	pq.logger.Info("Finished processing items left by consumers.",
		zap.Int("attempted", len(itemsToProcess)),
		zap.Int("reEnqueued", len(successfullyReEnqueuedIndices)),
		zap.Int("droppedOrFailedRetrieve", len(failedOrBadItemIndices)),
		zap.Int("remainingInDispatched", len(pq.currentlyDispatchedItems)))

	cleanupOps := make([]*storage.Operation, 0, len(successfullyReEnqueuedIndices)+len(failedOrBadItemIndices)+1)

	for _, itemIndex := range successfullyReEnqueuedIndices {
		cleanupOps = append(cleanupOps, storage.DeleteOperation(getItemKey(itemIndex)))
	}
	for _, itemIndex := range failedOrBadItemIndices {
		cleanupOps = append(cleanupOps, storage.DeleteOperation(getItemKey(itemIndex)))
	}

	if len(successfullyReEnqueuedIndices) > 0 || len(failedOrBadItemIndices) > 0 {
		metadataBytes, err := pq.marshalCurrentMetadata()
		if err != nil {
			pq.logger.Error("Failed to marshal metadata after processing dispatched items. State may be inconsistent.", zap.Error(err))
		} else {
			cleanupOps = append(cleanupOps, storage.SetOperation(queueMetadataKey, metadataBytes))
		}

		if len(cleanupOps) > 0 {
			if err := pq.client.Batch(ctx, cleanupOps...); err != nil {
				pq.logger.Error("Failed to batch update metadata and delete processed dispatched items during startup.", zap.Error(err))
			} else {
				pq.logger.Info("Successfully updated metadata and cleaned up processed dispatched items from startup.")
			}
		}
	} else if len(pq.currentlyDispatchedItems) != len(itemsToProcess) {
		pq.logger.Info("Persisting metadata due to changes in dispatched items list after re-enqueue attempts.")
		if err := pq.persistCurrentMetadata(ctx); err != nil {
			pq.logger.Error("Failed to persist metadata after re-enqueue attempts.", zap.Error(err))
		}
	}
}

// itemDispatchingFinish removes the item from the list of currently dispatched items,
// deletes it from storage, and persists the updated metadata.
// pq.mu must be held.
func (pq *persistentQueue[T]) itemDispatchingFinish(ctx context.Context, index uint64) error {
	found := false
	newCDI := make([]uint64, 0, len(pq.currentlyDispatchedItems))
	for _, item := range pq.currentlyDispatchedItems {
		if item == index {
			found = true
		} else {
			newCDI = append(newCDI, item)
		}
	}
	pq.currentlyDispatchedItems = newCDI

	if !found {
		pq.logger.Warn("Item to finish dispatching not found in currently dispatched list. Metadata will still be persisted.", zap.Uint64("index", index))
	}

	metadataBytes, err := pq.marshalCurrentMetadata()
	if err != nil {
		pq.logger.Error("Failed to marshal metadata for itemDispatchingFinish", zap.Error(err))
		return fmt.Errorf("failed to marshal metadata for itemDispatchingFinish: %w", err)
	}

	ops := []*storage.Operation{
		storage.SetOperation(queueMetadataKey, metadataBytes),
		storage.DeleteOperation(getItemKey(index)),
	}

	if err := pq.client.Batch(ctx, ops...); err != nil {
		pq.logger.Warn("Failed to batch update itemDispatchingFinish, trying operations separately.", zap.Error(err), zap.Uint64("index", index))

		// Try deleting item first
		if delErr := pq.client.Delete(ctx, getItemKey(index)); delErr != nil {
			pq.logger.Error("Failed to delete item from queue storage", zap.Error(delErr))
			return fmt.Errorf("failed to delete item from queue storage: %w, metadata not updated", delErr)
		}

		// Item deleted, now try to set metadata
		if metaErr := pq.client.Set(ctx, queueMetadataKey, metadataBytes); metaErr != nil {
			pq.logger.Error("itemDispatchingFinish: Failed to set metadata after deleting item", zap.Error(metaErr))
			return fmt.Errorf("deleted item successfully, but failed to set metadata: %w", metaErr)
		}

		return nil // Both succeeded separately
	}
	return nil // Batch succeeded
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
	for i := 0; i < size; i++ {
		val[i] = binary.LittleEndian.Uint64(buf)
		buf = buf[8:]
	}
	return val, nil
}

type indexDone struct {
	index uint64
	size  int64
	queue interface {
		onDone(uint64, int64, error)
	}
}

func (id *indexDone) reset(index uint64, size int64, queue interface{ onDone(uint64, int64, error) }) {
	id.index = index
	id.size = size
	id.queue = queue
}

func (id *indexDone) OnDone(err error) {
	id.queue.onDone(id.index, id.size, err)
}
