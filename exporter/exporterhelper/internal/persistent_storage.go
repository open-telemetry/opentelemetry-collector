// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/extension/experimental/storage"
)

// persistentContiguousStorage provides a persistent queue implementation backed by file storage extension
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
type persistentContiguousStorage[T any] struct {
	logger      *zap.Logger
	client      storage.Client
	unmarshaler func(data []byte) (T, error)
	marshaler   func(req T) ([]byte, error)

	putChan  chan struct{}
	stopChan chan struct{}
	capacity uint64

	mu                       sync.Mutex
	readIndex                uint64
	writeIndex               uint64
	currentlyDispatchedItems []uint64
	refClient                int64
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
	errValueNotSet = errors.New("value not set")
)

// newPersistentContiguousStorage creates a new file-storage extension backed queue;
// queueName parameter must be a unique value that identifies the queue.
func newPersistentContiguousStorage[T any](
	logger *zap.Logger, capacity uint64, marshaler func(req T) ([]byte, error), unmarshaler func([]byte) (T, error)) *persistentContiguousStorage[T] {
	return &persistentContiguousStorage[T]{
		logger:      logger,
		unmarshaler: unmarshaler,
		marshaler:   marshaler,
		capacity:    capacity,
		putChan:     make(chan struct{}, capacity),
		stopChan:    make(chan struct{}),
	}

}

func (pcs *persistentContiguousStorage[T]) start(ctx context.Context, client storage.Client) {
	pcs.client = client
	pcs.refClient = 1
	pcs.initPersistentContiguousStorage(ctx)
	// Make sure the leftover requests are handled
	pcs.retrieveAndEnqueueNotDispatchedReqs(ctx)

	// Ensure the communication channel has the same size as the queue
	// We might already have items here from requeueing non-dispatched requests
	for len(pcs.putChan) < int(pcs.size()) {
		pcs.putChan <- struct{}{}
	}
}

func (pcs *persistentContiguousStorage[T]) initPersistentContiguousStorage(ctx context.Context) {
	riOp := storage.GetOperation(readIndexKey)
	wiOp := storage.GetOperation(writeIndexKey)

	err := pcs.client.Batch(ctx, riOp, wiOp)
	if err == nil {
		pcs.readIndex, err = bytesToItemIndex(riOp.Value)
	}

	if err == nil {
		pcs.writeIndex, err = bytesToItemIndex(wiOp.Value)
	}

	if err != nil {
		if errors.Is(err, errValueNotSet) {
			pcs.logger.Info("Initializing new persistent queue")
		} else {
			pcs.logger.Error("Failed getting read/write index, starting with new ones", zap.Error(err))
		}
		pcs.readIndex = 0
		pcs.writeIndex = 0
	}
}

// get returns the request channel that all the requests will be send on
func (pcs *persistentContiguousStorage[T]) get() (QueueRequest[T], bool) {
	for {
		select {
		case <-pcs.stopChan:
			return QueueRequest[T]{}, false
		case <-pcs.putChan:
			req, found := pcs.getNextItem(context.Background())
			if found {
				return req, true
			}
		}
	}
}

func (pcs *persistentContiguousStorage[T]) size() uint64 {
	return pcs.writeIndex - pcs.readIndex
}

// Size returns the number of currently available items, which were not picked by consumers yet
func (pcs *persistentContiguousStorage[T]) Size() int {
	pcs.mu.Lock()
	defer pcs.mu.Unlock()
	return int(pcs.size())
}

// Capacity returns the number of currently available items, which were not picked by consumers yet
func (pcs *persistentContiguousStorage[T]) Capacity() int {
	return int(pcs.capacity)
}

func (pcs *persistentContiguousStorage[T]) Shutdown(ctx context.Context) error {
	close(pcs.stopChan)
	// Hold the lock only for `refClient`.
	pcs.mu.Lock()
	defer pcs.mu.Unlock()
	return pcs.unrefClient(ctx)
}

// unrefClient unrefs the client, and closes if no more references. Callers MUST hold the mutex.
// This is needed because consumers of the queue may still process the requests while the queue is shutting down or immediately after.
func (pcs *persistentContiguousStorage[T]) unrefClient(ctx context.Context) error {
	pcs.refClient--
	if pcs.refClient == 0 {
		return pcs.client.Close(ctx)
	}
	return nil
}

// Offer inserts the specified element into this queue if it is possible to do so immediately
// without violating capacity restrictions. If success returns no error.
// It returns ErrQueueIsFull if no space is currently available.
func (pcs *persistentContiguousStorage[T]) Offer(ctx context.Context, req T) error {
	pcs.mu.Lock()
	defer pcs.mu.Unlock()
	return pcs.putInternal(ctx, req)
}

// putInternal is the internal version that requires caller to hold the mutex lock.
func (pcs *persistentContiguousStorage[T]) putInternal(ctx context.Context, req T) error {
	if pcs.size() >= pcs.capacity {
		pcs.logger.Warn("Maximum queue capacity reached")
		return ErrQueueIsFull
	}

	itemKey := getItemKey(pcs.writeIndex)
	pcs.writeIndex++

	reqBuf, err := pcs.marshaler(req)
	if err != nil {
		return err
	}
	err = pcs.client.Batch(ctx,
		storage.SetOperation(writeIndexKey, itemIndexToBytes(pcs.writeIndex)),
		storage.SetOperation(itemKey, reqBuf))

	// Inform the loop that there's some data to process
	pcs.putChan <- struct{}{}

	return err
}

// getNextItem pulls the next available item from the persistent storage; if none is found, returns (nil, false)
func (pcs *persistentContiguousStorage[T]) getNextItem(ctx context.Context) (QueueRequest[T], bool) {
	pcs.mu.Lock()
	defer pcs.mu.Unlock()

	// If called in the same time with Shutdown, make sure client is not closed.
	if pcs.refClient <= 0 {
		return QueueRequest[T]{}, false
	}

	if pcs.readIndex == pcs.writeIndex {
		return QueueRequest[T]{}, false
	}
	index := pcs.readIndex
	// Increase here, so even if errors happen below, it always iterates
	pcs.readIndex++

	pcs.currentlyDispatchedItems = append(pcs.currentlyDispatchedItems, index)
	getOp := storage.GetOperation(getItemKey(index))
	err := pcs.client.Batch(ctx,
		storage.SetOperation(readIndexKey, itemIndexToBytes(pcs.readIndex)),
		storage.SetOperation(currentlyDispatchedItemsKey, itemIndexArrayToBytes(pcs.currentlyDispatchedItems)),
		getOp)

	var request T
	if err == nil {
		request, err = pcs.unmarshaler(getOp.Value)
	}

	if err != nil {
		pcs.logger.Debug("Failed to dispatch item", zap.Error(err))
		// We need to make sure that currently dispatched items list is cleaned
		if err = pcs.itemDispatchingFinish(ctx, index); err != nil {
			pcs.logger.Error("Error deleting item from queue", zap.Error(err))
		}

		return QueueRequest[T]{}, false
	}

	req := newQueueRequest[T](context.Background(), request)
	// If all went well so far, cleanup will be handled by callback
	pcs.refClient++
	req.onProcessingFinishedFunc = func() {
		pcs.mu.Lock()
		defer pcs.mu.Unlock()
		if err = pcs.itemDispatchingFinish(ctx, index); err != nil {
			pcs.logger.Error("Error deleting item from queue", zap.Error(err))
		}
		if err = pcs.unrefClient(ctx); err != nil {
			pcs.logger.Error("Error closing the storage client", zap.Error(err))
		}
	}
	return req, true
}

// retrieveAndEnqueueNotDispatchedReqs gets the items for which sending was not finished, cleans the storage
// and moves the items at the back of the queue.
func (pcs *persistentContiguousStorage[T]) retrieveAndEnqueueNotDispatchedReqs(ctx context.Context) {
	var dispatchedItems []uint64

	pcs.mu.Lock()
	defer pcs.mu.Unlock()
	pcs.logger.Debug("Checking if there are items left for dispatch by consumers")
	itemKeysBuf, err := pcs.client.Get(ctx, currentlyDispatchedItemsKey)
	if err == nil {
		dispatchedItems, err = bytesToItemIndexArray(itemKeysBuf)
	}
	if err != nil {
		pcs.logger.Error("Could not fetch items left for dispatch by consumers", zap.Error(err))
		return
	}

	if len(dispatchedItems) == 0 {
		pcs.logger.Debug("No items left for dispatch by consumers")
		return
	}

	pcs.logger.Info("Fetching items left for dispatch by consumers", zap.Int(zapNumberOfItems, len(dispatchedItems)))
	retrieveBatch := make([]storage.Operation, len(dispatchedItems))
	cleanupBatch := make([]storage.Operation, len(dispatchedItems))
	for i, it := range dispatchedItems {
		key := getItemKey(it)
		retrieveBatch[i] = storage.GetOperation(key)
		cleanupBatch[i] = storage.DeleteOperation(key)
	}
	retrieveErr := pcs.client.Batch(ctx, retrieveBatch...)
	cleanupErr := pcs.client.Batch(ctx, cleanupBatch...)

	if cleanupErr != nil {
		pcs.logger.Debug("Failed cleaning items left by consumers", zap.Error(cleanupErr))
	}

	if retrieveErr != nil {
		pcs.logger.Warn("Failed retrieving items left by consumers", zap.Error(retrieveErr))
		return
	}

	errCount := 0
	for _, op := range retrieveBatch {
		if op.Value == nil {
			pcs.logger.Warn("Failed retrieving item", zap.String(zapKey, op.Key), zap.Error(errValueNotSet))
			continue
		}
		req, err := pcs.unmarshaler(op.Value)
		// If error happened or item is nil, it will be efficiently ignored
		if err != nil {
			pcs.logger.Warn("Failed unmarshalling item", zap.String(zapKey, op.Key), zap.Error(err))
			continue
		}
		if pcs.putInternal(ctx, req) != nil {
			errCount++
		}
	}

	if errCount > 0 {
		pcs.logger.Error("Errors occurred while moving items for dispatching back to queue",
			zap.Int(zapNumberOfItems, len(retrieveBatch)), zap.Int(zapErrorCount, errCount))
	} else {
		pcs.logger.Info("Moved items for dispatching back to queue",
			zap.Int(zapNumberOfItems, len(retrieveBatch)))
	}
}

// itemDispatchingFinish removes the item from the list of currently dispatched items and deletes it from the persistent queue
func (pcs *persistentContiguousStorage[T]) itemDispatchingFinish(ctx context.Context, index uint64) error {
	lenCDI := len(pcs.currentlyDispatchedItems)
	for i := 0; i < lenCDI; i++ {
		if pcs.currentlyDispatchedItems[i] == index {
			pcs.currentlyDispatchedItems[i] = pcs.currentlyDispatchedItems[lenCDI-1]
			pcs.currentlyDispatchedItems = pcs.currentlyDispatchedItems[:lenCDI-1]
			break
		}
	}

	setOp := storage.SetOperation(currentlyDispatchedItemsKey, itemIndexArrayToBytes(pcs.currentlyDispatchedItems))
	deleteOp := storage.DeleteOperation(getItemKey(index))
	if err := pcs.client.Batch(ctx, setOp, deleteOp); err != nil {
		// got an error, try to gracefully handle it
		pcs.logger.Warn("Failed updating currently dispatched items, trying to delete the item first", zap.Error(err))
	} else {
		// Everything ok, exit
		return nil
	}

	if err := pcs.client.Batch(ctx, deleteOp); err != nil {
		// Return an error here, as this indicates an issue with the underlying storage medium
		return fmt.Errorf("failed deleting item from queue, got error from storage: %w", err)
	}

	if err := pcs.client.Batch(ctx, setOp); err != nil {
		// even if this fails, we still have the right dispatched items in memory
		// at worst, we'll have the wrong list in storage, and we'll discard the nonexistent items during startup
		return fmt.Errorf("failed updating currently dispatched items, but deleted item successfully: %w", err)
	}

	return nil
}

func getItemKey(index uint64) string {
	return strconv.FormatUint(index, 10)
}

func itemIndexToBytes(value uint64) []byte {
	return binary.LittleEndian.AppendUint64([]byte{}, value)
}

func bytesToItemIndex(b []byte) (uint64, error) {
	val := uint64(0)
	if b == nil {
		return val, errValueNotSet
	}
	err := binary.Read(bytes.NewReader(b), binary.LittleEndian, &val)
	return val, err
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

func bytesToItemIndexArray(b []byte) ([]uint64, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var size uint32
	reader := bytes.NewReader(b)
	if err := binary.Read(reader, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	val := make([]uint64, size)
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}
