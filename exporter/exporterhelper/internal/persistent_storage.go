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
	"sync/atomic"

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
type persistentContiguousStorage struct {
	logger      *zap.Logger
	client      storage.Client
	unmarshaler QueueRequestUnmarshaler
	marshaler   QueueRequestMarshaler

	putChan  chan struct{}
	stopChan chan struct{}
	capacity uint64

	mu                       sync.Mutex
	readIndex                itemIndex
	writeIndex               itemIndex
	currentlyDispatchedItems []itemIndex

	itemsCount *atomic.Uint64
}

type itemIndex uint64

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
func newPersistentContiguousStorage(ctx context.Context, client storage.Client,
	logger *zap.Logger, capacity uint64, marshaler QueueRequestMarshaler, unmarshaler QueueRequestUnmarshaler) *persistentContiguousStorage {
	pcs := &persistentContiguousStorage{
		logger:      logger,
		client:      client,
		unmarshaler: unmarshaler,
		marshaler:   marshaler,
		capacity:    capacity,
		putChan:     make(chan struct{}, capacity),
		stopChan:    make(chan struct{}),
		itemsCount:  &atomic.Uint64{},
	}

	pcs.initPersistentContiguousStorage(ctx)
	notDispatchedReqs := pcs.retrieveNotDispatchedReqs(ctx)

	// Make sure the leftover requests are handled
	pcs.enqueueNotDispatchedReqs(notDispatchedReqs)

	// Ensure the communication channel has the same size as the queue
	// We might already have items here from requeueing non-dispatched requests
	for len(pcs.putChan) < int(pcs.size()) {
		pcs.putChan <- struct{}{}
	}

	return pcs
}

func (pcs *persistentContiguousStorage) initPersistentContiguousStorage(ctx context.Context) {
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

	pcs.itemsCount.Store(uint64(pcs.writeIndex - pcs.readIndex))
}

func (pcs *persistentContiguousStorage) enqueueNotDispatchedReqs(reqs []any) {
	if len(reqs) > 0 {
		errCount := 0
		for _, req := range reqs {
			if req == nil || pcs.put(req) != nil {
				errCount++
			}
		}
		if errCount > 0 {
			pcs.logger.Error("Errors occurred while moving items for dispatching back to queue",
				zap.Int(zapNumberOfItems, len(reqs)), zap.Int(zapErrorCount, errCount))

		} else {
			pcs.logger.Info("Moved items for dispatching back to queue",
				zap.Int(zapNumberOfItems, len(reqs)))
		}
	}
}

// get returns the request channel that all the requests will be send on
func (pcs *persistentContiguousStorage) get() (QueueRequest, bool) {
	for {
		select {
		case <-pcs.stopChan:
			return QueueRequest{}, false
		case <-pcs.putChan:
			req := pcs.getNextItem(context.Background())
			if req.Request != nil {
				return req, true
			}
		}
	}
}

// size returns the number of currently available items, which were not picked by consumers yet
func (pcs *persistentContiguousStorage) size() uint64 {
	return pcs.itemsCount.Load()
}

func (pcs *persistentContiguousStorage) stop(ctx context.Context) error {
	pcs.logger.Debug("Stopping persistentContiguousStorage")
	return pcs.client.Close(ctx)
}

// put marshals the request and puts it into the persistent queue
func (pcs *persistentContiguousStorage) put(req any) error {
	// Nil requests are ignored
	if req == nil {
		return nil
	}

	pcs.mu.Lock()
	defer pcs.mu.Unlock()

	if pcs.size() >= pcs.capacity {
		pcs.logger.Warn("Maximum queue capacity reached")
		return ErrQueueIsFull
	}

	itemKey := getItemKey(pcs.writeIndex)
	pcs.writeIndex++
	pcs.itemsCount.Store(uint64(pcs.writeIndex - pcs.readIndex))

	reqBuf, err := pcs.marshaler(req)
	if err != nil {
		return err
	}
	ctx := context.Background()
	err = pcs.client.Batch(ctx,
		storage.SetOperation(writeIndexKey, itemIndexToBytes(pcs.writeIndex)),
		storage.SetOperation(itemKey, reqBuf))

	// Inform the loop that there's some data to process
	pcs.putChan <- struct{}{}

	return err
}

// getNextItem pulls the next available item from the persistent storage; if none is found, returns (nil, false)
func (pcs *persistentContiguousStorage) getNextItem(ctx context.Context) QueueRequest {
	pcs.mu.Lock()
	defer pcs.mu.Unlock()

	if pcs.readIndex == pcs.writeIndex {
		return QueueRequest{}
	}
	index := pcs.readIndex
	// Increase here, so even if errors happen below, it always iterates
	pcs.readIndex++
	pcs.itemsCount.Store(uint64(pcs.writeIndex - pcs.readIndex))

	pcs.updateReadIndex(ctx)
	pcs.itemDispatchingStart(ctx, index)

	req := newQueueRequest(context.Background(), nil)
	itemKey := getItemKey(index)
	buf, err := pcs.client.Get(ctx, itemKey)
	if err == nil {
		req.Request, err = pcs.unmarshaler(buf)
	}

	if err != nil || req.Request == nil {
		// We need to make sure that currently dispatched items list is cleaned
		if err := pcs.itemDispatchingFinish(ctx, index); err != nil {
			pcs.logger.Error("Error deleting item from queue", zap.Error(err))
		}

		return QueueRequest{}
	}

	// If all went well so far, cleanup will be handled by callback
	req.onProcessingFinishedFunc = func() {
		pcs.mu.Lock()
		defer pcs.mu.Unlock()
		if err := pcs.itemDispatchingFinish(ctx, index); err != nil {
			pcs.logger.Error("Error deleting item from queue", zap.Error(err))
		}
	}
	return req
}

// retrieveNotDispatchedReqs gets the items for which sending was not finished, cleans the storage
// and moves the items back to the queue. The function returns an array which might contain nils
// if unmarshalling of the value at a given index was not possible.
func (pcs *persistentContiguousStorage) retrieveNotDispatchedReqs(ctx context.Context) []any {
	var dispatchedItems []itemIndex

	pcs.mu.Lock()
	defer pcs.mu.Unlock()

	pcs.logger.Debug("Checking if there are items left for dispatch by consumers")
	itemKeysBuf, err := pcs.client.Get(ctx, currentlyDispatchedItemsKey)
	if err == nil {
		dispatchedItems, err = bytesToItemIndexArray(itemKeysBuf)
	}
	if err != nil {
		pcs.logger.Error("Could not fetch items left for dispatch by consumers", zap.Error(err))
		return nil
	}

	if len(dispatchedItems) > 0 {
		pcs.logger.Info("Fetching items left for dispatch by consumers", zap.Int(zapNumberOfItems, len(dispatchedItems)))
	} else {
		pcs.logger.Debug("No items left for dispatch by consumers")
	}

	reqs := make([]any, len(dispatchedItems))
	retrieveBatch := make([]storage.Operation, len(dispatchedItems))
	cleanupBatch := make([]storage.Operation, len(dispatchedItems))
	for i, it := range dispatchedItems {
		key := getItemKey(it)
		retrieveBatch[i] = storage.GetOperation(key)
		cleanupBatch[i] = storage.DeleteOperation(key)
	}

	retrieveErr := pcs.client.Batch(ctx, retrieveBatch...)
	cleanupErr := pcs.client.Batch(ctx, cleanupBatch...)

	if retrieveErr != nil {
		pcs.logger.Warn("Failed retrieving items left by consumers", zap.Error(retrieveErr))
	}

	if cleanupErr != nil {
		pcs.logger.Debug("Failed cleaning items left by consumers", zap.Error(cleanupErr))
	}

	if retrieveErr != nil {
		return reqs
	}

	for i, op := range retrieveBatch {
		if op.Value == nil {
			pcs.logger.Warn("Failed unmarshalling item", zap.String(zapKey, op.Key), zap.Error(errValueNotSet))
			continue
		}
		req, err := pcs.unmarshaler(op.Value)
		// If error happened or item is nil, it will be efficiently ignored
		if err != nil {
			pcs.logger.Warn("Failed unmarshalling item", zap.String(zapKey, op.Key), zap.Error(err))
			continue
		}
		if req == nil {
			pcs.logger.Debug("Item value could not be retrieved", zap.String(zapKey, op.Key), zap.Error(err))
		} else {
			reqs[i] = req
		}
	}

	return reqs
}

// itemDispatchingStart appends the item to the list of currently dispatched items
func (pcs *persistentContiguousStorage) itemDispatchingStart(ctx context.Context, index itemIndex) {
	pcs.currentlyDispatchedItems = append(pcs.currentlyDispatchedItems, index)
	err := pcs.client.Set(ctx, currentlyDispatchedItemsKey, itemIndexArrayToBytes(pcs.currentlyDispatchedItems))
	if err != nil {
		pcs.logger.Debug("Failed updating currently dispatched items", zap.Error(err))
	}
}

// itemDispatchingFinish removes the item from the list of currently dispatched items and deletes it from the persistent queue
func (pcs *persistentContiguousStorage) itemDispatchingFinish(ctx context.Context, index itemIndex) error {
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

func (pcs *persistentContiguousStorage) updateReadIndex(ctx context.Context) {
	err := pcs.client.Set(ctx, readIndexKey, itemIndexToBytes(pcs.readIndex))
	if err != nil {
		pcs.logger.Debug("Failed updating read index", zap.Error(err))
	}
}

func getItemKey(index itemIndex) string {
	return strconv.FormatUint(uint64(index), 10)
}

func itemIndexToBytes(value itemIndex) []byte {
	return binary.LittleEndian.AppendUint64([]byte{}, uint64(value))
}

func bytesToItemIndex(b []byte) (itemIndex, error) {
	val := itemIndex(0)
	if b == nil {
		return val, errValueNotSet
	}
	err := binary.Read(bytes.NewReader(b), binary.LittleEndian, &val)
	return val, err
}

func itemIndexArrayToBytes(arr []itemIndex) []byte {
	size := len(arr)
	buf := make([]byte, 0, 4+size*8)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(size))
	for _, item := range arr {
		buf = binary.LittleEndian.AppendUint64(buf, uint64(item))
	}
	return buf
}

func bytesToItemIndexArray(b []byte) ([]itemIndex, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var size uint32
	reader := bytes.NewReader(b)
	if err := binary.Read(reader, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	val := make([]itemIndex, size)
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}
