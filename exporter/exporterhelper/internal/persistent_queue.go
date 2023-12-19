// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension/experimental/storage"
)

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
	set         exporter.CreateSettings
	storageID   component.ID
	dataType    component.DataType
	client      storage.Client
	unmarshaler func(data []byte) (T, error)
	marshaler   func(req T) ([]byte, error)

	putChan  chan struct{}
	capacity uint64

	// mu guards everything declared below.
	mu                       sync.Mutex
	readIndex                uint64
	writeIndex               uint64
	currentlyDispatchedItems []uint64
	refClient                int64
	stopped                  bool
}

const (
	zapKey           = "key"
	zapErrorCount    = "errorCount"
	zapNumberOfItems = "numberOfItems"

	readIndexKey                = "ri"
	writeIndexKey               = "wi"
	currentlyDispatchedItemsKey = "di"
)

var (
	errValueNotSet        = errors.New("value not set")
	errInvalidValue       = errors.New("invalid value")
	errNoStorageClient    = errors.New("no storage client extension found")
	errWrongExtensionType = errors.New("requested extension is not a storage extension")
)

// NewPersistentQueue creates a new queue backed by file storage; name and signal must be a unique combination that identifies the queue storage
func NewPersistentQueue[T any](capacity int, dataType component.DataType, storageID component.ID, marshaler func(req T) ([]byte, error), unmarshaler func([]byte) (T, error), set exporter.CreateSettings) Queue[T] {
	return &persistentQueue[T]{
		set:         set,
		storageID:   storageID,
		dataType:    dataType,
		unmarshaler: unmarshaler,
		marshaler:   marshaler,
		capacity:    uint64(capacity),
		putChan:     make(chan struct{}, capacity),
	}
}

// Start starts the persistentQueue with the given number of consumers.
func (pq *persistentQueue[T]) Start(ctx context.Context, host component.Host) error {
	storageClient, err := toStorageClient(ctx, pq.storageID, host, pq.set.ID, pq.dataType)
	if err != nil {
		return err
	}
	pq.initClient(ctx, storageClient)
	return nil
}

func (pq *persistentQueue[T]) initClient(ctx context.Context, client storage.Client) {
	pq.client = client
	// Start with a reference 1 which is the reference we use for the producer goroutines and initialization.
	pq.refClient = 1
	pq.initPersistentContiguousStorage(ctx)
	// Make sure the leftover requests are handled
	pq.retrieveAndEnqueueNotDispatchedReqs(ctx)

	// Ensure the communication channel has the same size as the queue
	// We might already have items here from requeueing non-dispatched requests
	for len(pq.putChan) < int(pq.size()) {
		pq.putChan <- struct{}{}
	}
}

func (pq *persistentQueue[T]) initPersistentContiguousStorage(ctx context.Context) {
	riOp := storage.GetOperation(readIndexKey)
	wiOp := storage.GetOperation(writeIndexKey)

	err := pq.client.Batch(ctx, riOp, wiOp)
	if err == nil {
		pq.readIndex, err = bytesToItemIndex(riOp.Value)
	}

	if err == nil {
		pq.writeIndex, err = bytesToItemIndex(wiOp.Value)
	}

	if err != nil {
		if errors.Is(err, errValueNotSet) {
			pq.set.Logger.Info("Initializing new persistent queue")
		} else {
			pq.set.Logger.Error("Failed getting read/write index, starting with new ones", zap.Error(err))
		}
		pq.readIndex = 0
		pq.writeIndex = 0
	}
}

// Consume applies the provided function on the head of queue.
// The call blocks until there is an item available or the queue is stopped.
// The function returns true when an item is consumed or false if the queue is stopped.
func (pq *persistentQueue[T]) Consume(consumeFunc func(context.Context, T) error) bool {
	for {
		// If we are stopped we still process all the other events in the channel before, but we
		// return fast in the `getNextItem`, so we will free the channel fast and get to the stop.
		_, ok := <-pq.putChan
		if !ok {
			return false
		}

		req, onProcessingFinished, consumed := pq.getNextItem(context.Background())
		if consumed {
			onProcessingFinished(consumeFunc(context.Background(), req))
			return true
		}
	}
}

func (pq *persistentQueue[T]) size() uint64 {
	return pq.writeIndex - pq.readIndex
}

// Size returns the number of currently available items, which were not picked by consumers yet
func (pq *persistentQueue[T]) Size() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return int(pq.size())
}

// Capacity returns the number of currently available items, which were not picked by consumers yet
func (pq *persistentQueue[T]) Capacity() int {
	return int(pq.capacity)
}

func (pq *persistentQueue[T]) Shutdown(ctx context.Context) error {
	close(pq.putChan)
	pq.mu.Lock()
	defer pq.mu.Unlock()
	// Mark this queue as stopped, so consumer don't start any more work.
	pq.stopped = true
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
	if pq.size() >= pq.capacity {
		pq.set.Logger.Warn("Maximum queue capacity reached")
		return ErrQueueIsFull
	}

	itemKey := getItemKey(pq.writeIndex)
	newIndex := pq.writeIndex + 1

	reqBuf, err := pq.marshaler(req)
	if err != nil {
		return err
	}

	// Carry out a transaction where we both add the item and update the write index
	ops := []storage.Operation{
		storage.SetOperation(writeIndexKey, itemIndexToBytes(newIndex)),
		storage.SetOperation(itemKey, reqBuf),
	}
	if storageErr := pq.client.Batch(ctx, ops...); storageErr != nil {
		return storageErr
	}

	pq.writeIndex = newIndex
	// Inform the loop that there's some data to process
	pq.putChan <- struct{}{}

	return nil
}

// getNextItem pulls the next available item from the persistent storage along with a callback function that should be
// called after the item is processed to clean up the storage. If no new item is available, returns false.
func (pq *persistentQueue[T]) getNextItem(ctx context.Context) (T, func(error), bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	var request T

	if pq.stopped {
		return request, nil, false
	}

	if pq.readIndex == pq.writeIndex {
		return request, nil, false
	}

	index := pq.readIndex
	// Increase here, so even if errors happen below, it always iterates
	pq.readIndex++
	pq.currentlyDispatchedItems = append(pq.currentlyDispatchedItems, index)
	getOp := storage.GetOperation(getItemKey(index))
	err := pq.client.Batch(ctx,
		storage.SetOperation(readIndexKey, itemIndexToBytes(pq.readIndex)),
		storage.SetOperation(currentlyDispatchedItemsKey, itemIndexArrayToBytes(pq.currentlyDispatchedItems)),
		getOp)

	if err == nil {
		request, err = pq.unmarshaler(getOp.Value)
	}

	if err != nil {
		pq.set.Logger.Debug("Failed to dispatch item", zap.Error(err))
		// We need to make sure that currently dispatched items list is cleaned
		if err = pq.itemDispatchingFinish(ctx, index); err != nil {
			pq.set.Logger.Error("Error deleting item from queue", zap.Error(err))
		}

		return request, nil, false
	}

	// Increase the reference count, so the client is not closed while the request is being processed.
	// The client cannot be closed because we hold the lock since last we checked `stopped`.
	pq.refClient++
	return request, func(consumeErr error) {
		// Delete the item from the persistent storage after it was processed.
		pq.mu.Lock()
		// Always unref client even if the consumer is shutdown because we always ref it for every valid request.
		defer func() {
			if err = pq.unrefClient(ctx); err != nil {
				pq.set.Logger.Error("Error closing the storage client", zap.Error(err))
			}
			pq.mu.Unlock()
		}()

		if errors.As(consumeErr, &shutdownErr{}) {
			// The queue is shutting down, don't mark the item as dispatched, so it's picked up again after restart.
			// TODO: Handle partially delivered requests by updating their values in the storage.
			return
		}

		if err = pq.itemDispatchingFinish(ctx, index); err != nil {
			pq.set.Logger.Error("Error deleting item from queue", zap.Error(err))
		}

	}, true
}

// retrieveAndEnqueueNotDispatchedReqs gets the items for which sending was not finished, cleans the storage
// and moves the items at the back of the queue.
func (pq *persistentQueue[T]) retrieveAndEnqueueNotDispatchedReqs(ctx context.Context) {
	var dispatchedItems []uint64

	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.set.Logger.Debug("Checking if there are items left for dispatch by consumers")
	itemKeysBuf, err := pq.client.Get(ctx, currentlyDispatchedItemsKey)
	if err == nil {
		dispatchedItems, err = bytesToItemIndexArray(itemKeysBuf)
	}
	if err != nil {
		pq.set.Logger.Error("Could not fetch items left for dispatch by consumers", zap.Error(err))
		return
	}

	if len(dispatchedItems) == 0 {
		pq.set.Logger.Debug("No items left for dispatch by consumers")
		return
	}

	pq.set.Logger.Info("Fetching items left for dispatch by consumers", zap.Int(zapNumberOfItems,
		len(dispatchedItems)))
	retrieveBatch := make([]storage.Operation, len(dispatchedItems))
	cleanupBatch := make([]storage.Operation, len(dispatchedItems))
	for i, it := range dispatchedItems {
		key := getItemKey(it)
		retrieveBatch[i] = storage.GetOperation(key)
		cleanupBatch[i] = storage.DeleteOperation(key)
	}
	retrieveErr := pq.client.Batch(ctx, retrieveBatch...)
	cleanupErr := pq.client.Batch(ctx, cleanupBatch...)

	if cleanupErr != nil {
		pq.set.Logger.Debug("Failed cleaning items left by consumers", zap.Error(cleanupErr))
	}

	if retrieveErr != nil {
		pq.set.Logger.Warn("Failed retrieving items left by consumers", zap.Error(retrieveErr))
		return
	}

	errCount := 0
	for _, op := range retrieveBatch {
		if op.Value == nil {
			pq.set.Logger.Warn("Failed retrieving item", zap.String(zapKey, op.Key), zap.Error(errValueNotSet))
			continue
		}
		req, err := pq.unmarshaler(op.Value)
		// If error happened or item is nil, it will be efficiently ignored
		if err != nil {
			pq.set.Logger.Warn("Failed unmarshalling item", zap.String(zapKey, op.Key), zap.Error(err))
			continue
		}
		if pq.putInternal(ctx, req) != nil {
			errCount++
		}
	}

	if errCount > 0 {
		pq.set.Logger.Error("Errors occurred while moving items for dispatching back to queue",
			zap.Int(zapNumberOfItems, len(retrieveBatch)), zap.Int(zapErrorCount, errCount))
	} else {
		pq.set.Logger.Info("Moved items for dispatching back to queue",
			zap.Int(zapNumberOfItems, len(retrieveBatch)))
	}
}

// itemDispatchingFinish removes the item from the list of currently dispatched items and deletes it from the persistent queue
func (pq *persistentQueue[T]) itemDispatchingFinish(ctx context.Context, index uint64) error {
	lenCDI := len(pq.currentlyDispatchedItems)
	for i := 0; i < lenCDI; i++ {
		if pq.currentlyDispatchedItems[i] == index {
			pq.currentlyDispatchedItems[i] = pq.currentlyDispatchedItems[lenCDI-1]
			pq.currentlyDispatchedItems = pq.currentlyDispatchedItems[:lenCDI-1]
			break
		}
	}

	setOp := storage.SetOperation(currentlyDispatchedItemsKey, itemIndexArrayToBytes(pq.currentlyDispatchedItems))
	deleteOp := storage.DeleteOperation(getItemKey(index))
	if err := pq.client.Batch(ctx, setOp, deleteOp); err != nil {
		// got an error, try to gracefully handle it
		pq.set.Logger.Warn("Failed updating currently dispatched items, trying to delete the item first",
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

func toStorageClient(ctx context.Context, storageID component.ID, host component.Host, ownerID component.ID, signal component.DataType) (storage.Client, error) {
	ext, found := host.GetExtensions()[storageID]
	if !found {
		return nil, errNoStorageClient
	}

	storageExt, ok := ext.(storage.Extension)
	if !ok {
		return nil, errWrongExtensionType
	}

	return storageExt.GetClient(ctx, component.KindExporter, ownerID, string(signal))
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
