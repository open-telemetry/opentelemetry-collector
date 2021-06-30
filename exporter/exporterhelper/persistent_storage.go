// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporterhelper

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.opencensus.io/metric/metricdata"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/extension/storage"
)

// persistentStorage provides an interface for request storage operations
type persistentStorage interface {
	// put appends the request to the storage
	put(req request) error
	// get returns the next available request; note that the channel is unbuffered
	get() <-chan request
	// size returns the current size of the storage with items waiting for processing
	size() int
	// stop gracefully stops the storage
	stop()
}

// persistentContiguousStorage provides a persistent queue implementation backed by file storage extension
//
// Write index describes the position at which next item is going to be stored.
// Read index describes which item needs to be read next.
// When Write index = Read index, no elements are in the queue.
//
// The items currently processed by consumers are not deleted until the processing is finished.
// Their list is stored under a separate key.
//
//
//   ┌───────file extension-backed queue───────┐
//   │                                         │
//   │     ┌───┐     ┌───┐ ┌───┐ ┌───┐ ┌───┐   │
//   │ n+1 │ n │ ... │ 4 │ │ 3 │ │ 2 │ │ 1 │   │
//   │     └───┘     └───┘ └─x─┘ └─|─┘ └─x─┘   │
//   │                       x     |     x     │
//   └───────────────────────x─────|─────x─────┘
//      ▲              ▲     x     |     x
//      │              │     x     |     xxxx deleted
//      │              │     x     |
//    write          read    x     └── currently processed item
//    index          index   x
//                           xxxx deleted
//
type persistentContiguousStorage struct {
	logger      *zap.Logger
	queueName   string
	client      storage.Client
	retryDelay  time.Duration
	unmarshaler requestUnmarshaler

	reqChan  chan request
	stopOnce sync.Once
	stopChan chan struct{}

	mu                      sync.Mutex
	readIndex               itemIndex
	writeIndex              itemIndex
	currentlyProcessedItems []itemIndex
}

type itemIndex uint64

const (
	zapKey           = "key"
	zapQueueNameKey  = "queueName"
	zapErrorCount    = "errorCount"
	zapNumberOfItems = "numberOfItems"

	defaultRetryDelay = 100 * time.Millisecond

	readIndexKey               = "ri"
	writeIndexKey              = "wi"
	currentlyProcessedItemsKey = "pi"
)

var (
	errStoringItemToQueue = errors.New("item could not be stored to persistent queue")
	errUpdatingIndex      = errors.New("index could not be updated")
)

// newPersistentContiguousStorage creates a new file-storage extension backed queue. It needs to be initialized separately
func newPersistentContiguousStorage(ctx context.Context, queueName string, logger *zap.Logger, client storage.Client, unmarshaler requestUnmarshaler) *persistentContiguousStorage {
	wcs := &persistentContiguousStorage{
		logger:      logger,
		client:      client,
		queueName:   queueName,
		unmarshaler: unmarshaler,
		reqChan:     make(chan request),
		stopChan:    make(chan struct{}),
		retryDelay:  defaultRetryDelay,
	}

	initPersistentContiguousStorage(ctx, wcs)

	err := currentlyProcessedBatchesGauge.UpsertEntry(func() int64 {
		return int64(wcs.numberOfCurrentlyProcessedItems())
	}, metricdata.NewLabelValue(wcs.queueName))
	if err != nil {
		logger.Error("failed to create number of currently processed items metric", zap.Error(err))
	}

	go wcs.loop()
	return wcs
}

func initPersistentContiguousStorage(ctx context.Context, wcs *persistentContiguousStorage) {
	readIndexIf := wcs._clientGet(ctx, readIndexKey, bytesToItemIndex)
	if readIndexIf == nil {
		wcs.logger.Debug("failed getting read index, starting with a new one",
			zap.String(zapQueueNameKey, wcs.queueName))
		wcs.readIndex = 0
	} else {
		wcs.readIndex = readIndexIf.(itemIndex)
	}

	writeIndexIf := wcs._clientGet(ctx, writeIndexKey, bytesToItemIndex)
	if writeIndexIf == nil {
		wcs.logger.Debug("failed getting write index, starting with a new one",
			zap.String(zapQueueNameKey, wcs.queueName))
		wcs.writeIndex = 0
	} else {
		wcs.writeIndex = writeIndexIf.(itemIndex)
	}
}

// loop is the main loop that handles fetching items from the persistent buffer
func (pcs *persistentContiguousStorage) loop() {
	// We want to run it here so it's not blocking
	reqs := pcs.retrieveUnprocessedItems(context.Background())
	if len(reqs) > 0 {
		errCount := 0
		for _, req := range reqs {
			if pcs.put(req) != nil {
				errCount++
			}
		}
		pcs.logger.Info("moved items for processing back to queue",
			zap.String(zapQueueNameKey, pcs.queueName),
			zap.Int(zapNumberOfItems, len(reqs)), zap.Int(zapErrorCount, errCount))
	}

	for {
		req, found := pcs.getNextItem(context.Background())
		if found {
			select {
			case <-pcs.stopChan:
				return
			case pcs.reqChan <- req:
			}
		} else {
			select {
			case <-pcs.stopChan:
				return
			case <-time.After(pcs.retryDelay):
			}
		}
	}
}

// get returns the request channel that all the requests will be send on
func (pcs *persistentContiguousStorage) get() <-chan request {
	return pcs.reqChan
}

// size returns the number of currently available items, which were not picked by consumers yet
func (pcs *persistentContiguousStorage) size() int {
	pcs.mu.Lock()
	defer pcs.mu.Unlock()
	return int(pcs.writeIndex - pcs.readIndex)
}

// numberOfCurrentlyProcessedItems returns the count of batches for which processing started but hasn't finish yet
func (pcs *persistentContiguousStorage) numberOfCurrentlyProcessedItems() int {
	pcs.mu.Lock()
	defer pcs.mu.Unlock()
	return len(pcs.currentlyProcessedItems)
}

func (pcs *persistentContiguousStorage) stop() {
	pcs.logger.Debug("stopping persistentContiguousStorage", zap.String(zapQueueNameKey, pcs.queueName))
	pcs.stopOnce.Do(func() {
		_ = currentlyProcessedBatchesGauge.UpsertEntry(func() int64 {
			return int64(0)
		}, metricdata.NewLabelValue(pcs.queueName))
		close(pcs.stopChan)
	})
}

// put marshals the request and puts it into the persistent queue
func (pcs *persistentContiguousStorage) put(req request) error {
	pcs.mu.Lock()
	defer pcs.mu.Unlock()

	ctx := context.Background()

	itemKey := pcs.itemKey(pcs.writeIndex)
	if !pcs._clientSet(ctx, itemKey, req, requestToBytes) {
		return errStoringItemToQueue
	}

	pcs.writeIndex++
	if !pcs._clientSet(ctx, writeIndexKey, pcs.writeIndex, itemIndexToBytes) {
		return errUpdatingIndex
	}

	return nil
}

// getNextItem pulls the next available item from the persistent storage; if none is found, returns (nil, false)
func (pcs *persistentContiguousStorage) getNextItem(ctx context.Context) (request, bool) {
	pcs.mu.Lock()
	defer pcs.mu.Unlock()

	if pcs.readIndex < pcs.writeIndex {
		index := pcs.readIndex
		// Increase here, so even if errors happen below, it always iterates
		pcs.readIndex++

		pcs.updateReadIndex(ctx)
		pcs.itemProcessingStart(ctx, index)

		req := pcs._clientGet(ctx, pcs.itemKey(index), pcs.bytesToRequest).(request)
		if req == nil {
			return nil, false
		}

		req.setOnProcessingFinished(func() {
			pcs.mu.Lock()
			defer pcs.mu.Unlock()
			pcs.itemProcessingFinish(ctx, index)
		})
		return req, true
	}

	return nil, false
}

// retrieveUnprocessedItems gets the items for which processing was not finished, cleans the storage
// and moves the items back to the queue
func (pcs *persistentContiguousStorage) retrieveUnprocessedItems(ctx context.Context) []request {
	var reqs []request
	pcs.mu.Lock()
	defer pcs.mu.Unlock()

	pcs.logger.Debug("checking if there are items left by consumers")
	processedItemsIf := pcs._clientGet(ctx, currentlyProcessedItemsKey, bytesToItemIndexArray)
	if processedItemsIf == nil {
		return reqs
	}
	processedItems, ok := processedItemsIf.([]itemIndex)
	if !ok {
		pcs.logger.Warn("failed fetching list of unprocessed items",
			zap.String(zapQueueNameKey, pcs.queueName))
		return reqs
	}

	if len(processedItems) > 0 {
		pcs.logger.Info("fetching items left for processing by consumers",
			zap.String(zapQueueNameKey, pcs.queueName), zap.Int(zapNumberOfItems, len(processedItems)))
	} else {
		pcs.logger.Debug("no items left for processing by consumers")
	}

	for _, it := range processedItems {
		req := pcs._clientGet(ctx, pcs.itemKey(it), pcs.bytesToRequest).(request)
		pcs._clientDelete(ctx, pcs.itemKey(it))
		reqs = append(reqs, req)
	}

	return reqs
}

// itemProcessingStart appends the item to the list of currently processed items
func (pcs *persistentContiguousStorage) itemProcessingStart(ctx context.Context, index itemIndex) {
	pcs.currentlyProcessedItems = append(pcs.currentlyProcessedItems, index)
	pcs._clientSet(ctx, currentlyProcessedItemsKey, pcs.currentlyProcessedItems, itemIndexArrayToBytes)
}

// itemProcessingFinish removes the item from the list of currently processed items and deletes it from the persistent queue
func (pcs *persistentContiguousStorage) itemProcessingFinish(ctx context.Context, index itemIndex) {
	var updatedProcessedItems []itemIndex
	for _, it := range pcs.currentlyProcessedItems {
		if it != index {
			updatedProcessedItems = append(updatedProcessedItems, it)
		}
	}
	pcs.currentlyProcessedItems = updatedProcessedItems

	pcs._clientSet(ctx, currentlyProcessedItemsKey, pcs.currentlyProcessedItems, itemIndexArrayToBytes)
	pcs._clientDelete(ctx, pcs.itemKey(index))
}

func (pcs *persistentContiguousStorage) updateReadIndex(ctx context.Context) {
	pcs._clientSet(ctx, readIndexKey, pcs.readIndex, itemIndexToBytes)
}

func (pcs *persistentContiguousStorage) itemKey(index itemIndex) string {
	return fmt.Sprintf("%d", index)
}

func (pcs *persistentContiguousStorage) _clientSet(ctx context.Context, key string, value interface{}, marshal func(interface{}) ([]byte, error)) bool {
	valueBytes, err := marshal(value)
	if err != nil {
		pcs.logger.Warn("failed marshaling item",
			zap.String(zapQueueNameKey, pcs.queueName), zap.String(zapKey, key), zap.Error(err))
		return false
	}

	return pcs._clientSetBuf(ctx, key, valueBytes)
}

func (pcs *persistentContiguousStorage) _clientGet(ctx context.Context, key string, unmarshal func([]byte) (interface{}, error)) interface{} {
	valueBytes := pcs._clientGetBuf(ctx, key)
	if valueBytes == nil {
		return nil
	}

	item, err := unmarshal(valueBytes)
	if err != nil {
		pcs.logger.Warn("failed unmarshaling item",
			zap.String(zapQueueNameKey, pcs.queueName), zap.String(zapKey, key), zap.Error(err))
		return nil
	}

	return item
}

func (pcs *persistentContiguousStorage) _clientDelete(ctx context.Context, key string) {
	err := pcs.client.Delete(ctx, key)
	if err != nil {
		pcs.logger.Warn("failed deleting item",
			zap.String(zapQueueNameKey, pcs.queueName), zap.String(zapKey, key))
	}
}

func (pcs *persistentContiguousStorage) _clientGetBuf(ctx context.Context, key string) []byte {
	buf, err := pcs.client.Get(ctx, key)
	if err != nil {
		pcs.logger.Debug("error when getting item from persistent storage",
			zap.String(zapQueueNameKey, pcs.queueName), zap.String(zapKey, key), zap.Error(err))
		return nil
	}
	return buf
}

func (pcs *persistentContiguousStorage) _clientSetBuf(ctx context.Context, key string, buf []byte) bool {
	err := pcs.client.Set(ctx, key, buf)
	if err != nil {
		pcs.logger.Debug("error when storing item to persistent storage",
			zap.String(zapQueueNameKey, pcs.queueName), zap.String(zapKey, key), zap.Error(err))
		return false
	}
	return true
}

func itemIndexToBytes(val interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, val)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), err
}

func bytesToItemIndex(b []byte) (interface{}, error) {
	var val itemIndex
	err := binary.Read(bytes.NewReader(b), binary.LittleEndian, &val)
	if err != nil {
		return val, err
	}
	return val, nil
}

func itemIndexArrayToBytes(arr interface{}) ([]byte, error) {
	var buf bytes.Buffer
	count := 0

	if arr != nil {
		arrItemIndex, ok := arr.([]itemIndex)
		if ok {
			count = len(arrItemIndex)
		}
	}

	err := binary.Write(&buf, binary.LittleEndian, uint32(count))
	if err != nil {
		return nil, err
	}

	err = binary.Write(&buf, binary.LittleEndian, arr)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), err
}

func bytesToItemIndexArray(b []byte) (interface{}, error) {
	var size uint32
	reader := bytes.NewReader(b)
	err := binary.Read(reader, binary.LittleEndian, &size)
	if err != nil {
		return nil, err
	}

	val := make([]itemIndex, size)
	err = binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

func requestToBytes(req interface{}) ([]byte, error) {
	return req.(request).marshal()
}

func (pcs *persistentContiguousStorage) bytesToRequest(b []byte) (interface{}, error) {
	return pcs.unmarshaler(b)
}
