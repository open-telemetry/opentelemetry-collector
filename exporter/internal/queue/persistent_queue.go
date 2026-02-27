// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/internal/experr"
	"go.opentelemetry.io/collector/extension/experimental/storage"
)

// persistentQueue provides a persistent LIFO queue implementation backed by file storage extension
type persistentQueue[T any] struct {
	*sizedChannel[permanentQueueEl]

	set    PersistentQueueSettings[T]
	logger *zap.Logger
	client storage.Client

	isRequestSized bool

	mu                       sync.Mutex
	readIndex                uint64
	writeIndex               uint64
	lifoConsumeIndex         uint64
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
	queueSizeKey                = "si"
)

var (
	errValueNotSet        = errors.New("value not set")
	errInvalidValue       = errors.New("invalid value")
	errNoStorageClient    = errors.New("no storage client extension found")
	errWrongExtensionType = errors.New("requested extension is not a storage extension")
)

type PersistentQueueSettings[T any] struct {
	Sizer            Sizer[T]
	Capacity         int64
	DataType         component.DataType
	StorageID        component.ID
	Marshaler        func(req T) ([]byte, error)
	Unmarshaler      func([]byte) (T, error)
	ExporterSettings exporter.Settings
}

// NewPersistentQueue creates a new LIFO queue backed by file storage
func NewPersistentQueue[T any](set PersistentQueueSettings[T]) Queue[T] {
	_, isRequestSized := set.Sizer.(*RequestSizer[T])
	return &persistentQueue[T]{
		set:            set,
		logger:         set.ExporterSettings.Logger,
		isRequestSized: isRequestSized,
	}
}

func (pq *persistentQueue[T]) Start(ctx context.Context, host component.Host) error {
	storageClient, err := toStorageClient(ctx, pq.set.StorageID, host, pq.set.ExporterSettings.ID, pq.set.DataType)
	if err != nil {
		return err
	}
	pq.initClient(ctx, storageClient)
	return nil
}

func (pq *persistentQueue[T]) initClient(ctx context.Context, client storage.Client) {
	pq.client = client
	pq.refClient = 1
	pq.initPersistentContiguousStorage(ctx)
	pq.retrieveAndEnqueueNotDispatchedReqs(ctx)
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
			pq.logger.Info("Initializing new persistent queue")
		} else {
			pq.logger.Error("Failed getting read/write index, starting with new ones", zap.Error(err))
		}
		pq.readIndex = 0
		pq.writeIndex = 0
	}

	initIndexSize := pq.writeIndex - pq.readIndex

	var (
		initEls       []permanentQueueEl
		initQueueSize uint64
	)

	if initIndexSize > 0 {
		initQueueSize = initIndexSize
		if !pq.isRequestSized {
			if restoredQueueSize, err := pq.restoreQueueSizeFromStorage(ctx); err == nil {
				initQueueSize = restoredQueueSize
			}
		}
		initEls = make([]permanentQueueEl, initIndexSize)
	}

	pq.sizedChannel = newSizedChannel[permanentQueueEl](pq.set.Capacity, initEls, int64(initQueueSize))
}

type permanentQueueEl struct{}

func (pq *persistentQueue[T]) restoreQueueSizeFromStorage(ctx context.Context) (uint64, error) {
	val, err := pq.client.Get(ctx, queueSizeKey)
	if err != nil {
		if errors.Is(err, errValueNotSet) {
			pq.logger.Warn("Cannot read the queue size snapshot from storage. "+
				"The reported queue size will be inaccurate until the initial queue is drained. "+
				"It's expected when the items sized queue enabled for the first time", zap.Error(err))
		} else {
			pq.logger.Error("Failed to read the queue size snapshot from storage. "+
				"The reported queue size will be inaccurate until the initial queue is drained.", zap.Error(err))
		}
		return 0, err
	}
	return bytesToItemIndex(val)
}

// LIFO: Consume applies the provided function on the most recently added item (top of stack).
func (pq *persistentQueue[T]) Consume(consumeFunc func(context.Context, T) error) bool {
	for {
		var (
			req                  T
			onProcessingFinished func(error)
			consumed             bool
		)

		_, ok := pq.sizedChannel.pop(func(permanentQueueEl) int64 {
			req, onProcessingFinished, consumed = pq.getNextItem(context.Background())
			if !consumed {
				return 0
			}
			return pq.set.Sizer.Sizeof(req)
		})
		if !ok {
			return false
		}
		if consumed {
			onProcessingFinished(consumeFunc(context.Background(), req))
			return true
		}
	}
}

func (pq *persistentQueue[T]) Shutdown(ctx context.Context) error {
	if pq.client == nil {
		return nil
	}

	pq.mu.Lock()
	defer pq.mu.Unlock()
	backupErr := pq.backupQueueSize(ctx)
	pq.sizedChannel.shutdown()
	pq.stopped = true
	return multierr.Combine(backupErr, pq.unrefClient(ctx))
}

func (pq *persistentQueue[T]) backupQueueSize(ctx context.Context) error {
	if pq.isRequestSized {
		return nil
	}
	return pq.client.Set(ctx, queueSizeKey, itemIndexToBytes(uint64(pq.Size())))
}

func (pq *persistentQueue[T]) unrefClient(ctx context.Context) error {
	pq.refClient--
	if pq.refClient == 0 {
		return pq.client.Close(ctx)
	}
	return nil
}

func (pq *persistentQueue[T]) Offer(ctx context.Context, req T) error {
	pq.logger.Info("LIFO_DEBUG: Offer called")
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return pq.putInternal(ctx, req)
}

func (pq *persistentQueue[T]) putInternal(ctx context.Context, req T) error {
	err := pq.sizedChannel.push(permanentQueueEl{}, pq.set.Sizer.Sizeof(req), func() error {
		itemKey := getItemKey(pq.writeIndex)
		newIndex := pq.writeIndex + 1

		reqBuf, err := pq.set.Marshaler(req)
		if err != nil {
			return err
		}

		ops := []storage.Operation{
			storage.SetOperation(writeIndexKey, itemIndexToBytes(newIndex)),
			storage.SetOperation(itemKey, reqBuf),
		}
		if storageErr := pq.client.Batch(ctx, ops...); storageErr != nil {
			return storageErr
		}

		pq.lifoConsumeIndex = newIndex
		pq.writeIndex = newIndex
		pq.logger.Info("LIFO_DEBUG: Item offered to queue",
			zap.Uint64("writeIndex", pq.writeIndex),
			zap.Uint64("readIndex", pq.readIndex),
			zap.Uint64("queueSize", pq.writeIndex-pq.readIndex),
			zap.Int64("timestamp_ns", time.Now().UnixNano()))
		return nil
	})
	if err != nil {
		return err
	}

	if (pq.writeIndex % 10) == 5 {
		if err := pq.backupQueueSize(ctx); err != nil {
			pq.logger.Error("Error writing queue size to storage", zap.Error(err))
		}
	}

	return nil
}

// LIFO: getNextItem pulls the most recently added item from persistent storage.
func (pq *persistentQueue[T]) getNextItem(ctx context.Context) (T, func(error), bool) {
	pq.logger.Info("LIFO_DEBUG: getNextItem called")
	pq.mu.Lock()
	defer pq.mu.Unlock()

	var request T

	if pq.stopped {
		return request, nil, false
	}

	if pq.writeIndex == pq.readIndex {
		return request, nil, false
	}

	// Check if queue is empty (all items consumed)
	if pq.lifoConsumeIndex <= pq.readIndex {
		pq.logger.Info("LIFO_DEBUG: Queue empty, all consumed",
			zap.Uint64("lifoConsumeIndex", pq.lifoConsumeIndex),
			zap.Uint64("readIndex", pq.readIndex))
		return request, nil, false
	}

	// LIFO: Get the most recent unconsumed item
	index := pq.lifoConsumeIndex - 1
	pq.lifoConsumeIndex-- // Decrement LIFO pointer (NOT writeIndex!)

	pq.logger.Info("LIFO_DEBUG: About to consume item",
		zap.Uint64("indexBeingRetrieved", index),
		zap.Uint64("lifoConsumeIndex", pq.lifoConsumeIndex),
		zap.Uint64("writeIndex_before", pq.writeIndex),
		zap.Uint64("readIndex", pq.readIndex),
		zap.Uint64("remainingInQueue", pq.writeIndex-pq.readIndex-1),
		zap.Int64("timestamp_ns", time.Now().UnixNano())) // ADD THIS LINE
	pq.currentlyDispatchedItems = append(pq.currentlyDispatchedItems, index)
	getOp := storage.GetOperation(getItemKey(index))

	err := pq.client.Batch(ctx,
		storage.SetOperation(currentlyDispatchedItemsKey, itemIndexArrayToBytes(pq.currentlyDispatchedItems)),
		getOp)

	if err == nil {
		request, err = pq.set.Unmarshaler(getOp.Value)
	}

	if err != nil {
		pq.logger.Debug("Failed to dispatch item", zap.Error(err))
		if err = pq.itemDispatchingFinish(ctx, index); err != nil {
			pq.logger.Error("Error deleting item from queue", zap.Error(err))
		}
		return request, nil, false
	}

	pq.refClient++
	return request, func(consumeErr error) {
		pq.mu.Lock()
		defer func() {
			if err = pq.unrefClient(ctx); err != nil {
				pq.logger.Error("Error closing the storage client", zap.Error(err))
			}
			pq.mu.Unlock()
		}()

		if experr.IsShutdownErr(consumeErr) {
			return
		}

		if err = pq.itemDispatchingFinish(ctx, index); err != nil {
			pq.logger.Error("Error deleting item from queue", zap.Error(err))
		}

		if (pq.writeIndex % 10) == 0 {
			if qsErr := pq.backupQueueSize(ctx); qsErr != nil {
				pq.logger.Error("Error writing queue size to storage", zap.Error(qsErr))
			}
		}

		pq.sizedChannel.syncSize()

	}, true
}

// LIFO: re-enqueue not dispatched items in reverse order
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
	retrieveBatch := make([]storage.Operation, len(dispatchedItems))
	cleanupBatch := make([]storage.Operation, len(dispatchedItems))
	for i, it := range dispatchedItems {
		key := getItemKey(it)
		retrieveBatch[i] = storage.GetOperation(key)
		cleanupBatch[i] = storage.DeleteOperation(key)
		pq.logger.Info("LIFO_DEBUG: Recovery re-enqueueing item",
			zap.Int("loopIndex", len(retrieveBatch)-1-i),
			zap.Uint64("itemOriginalIndex", dispatchedItems[len(retrieveBatch)-1-i]),
			zap.Uint64("currentWriteIndex", pq.writeIndex),
			zap.Int64("timestamp_ns", time.Now().UnixNano()))
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
	// LIFO: re-enqueue in reverse order
	for i := len(retrieveBatch) - 1; i >= 0; i-- {
		op := retrieveBatch[i]
		if op.Value == nil {
			pq.logger.Warn("Failed retrieving item", zap.String(zapKey, op.Key), zap.Error(errValueNotSet))
			continue
		}
		req, err := pq.set.Unmarshaler(op.Value)
		if err != nil {
			pq.logger.Warn("Failed unmarshalling item", zap.String(zapKey, op.Key), zap.Error(err))
			continue
		}
		if pq.putInternal(ctx, req) != nil {
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
		pq.logger.Warn("Failed updating currently dispatched items, trying to delete the item first",
			zap.Error(err))
	} else {
		return nil
	}

	if err := pq.client.Batch(ctx, deleteOp); err != nil {
		return fmt.Errorf("failed deleting item from queue, got error from storage: %w", err)
	}

	if err := pq.client.Batch(ctx, setOp); err != nil {
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

	if len(buf) < 4 {
		return nil, errInvalidValue
	}
	size := int(binary.LittleEndian.Uint32(buf))
	if size == 0 {
		return nil, nil
	}

	buf = buf[4:]
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
