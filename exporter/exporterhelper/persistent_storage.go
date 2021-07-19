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
	"context"
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
	errValueNotSet = errors.New("value not set")

	errKeyNotPresentInBatch = errors.New("key was not present in get batchStruct")
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

func initPersistentContiguousStorage(ctx context.Context, pcs *persistentContiguousStorage) {
	var writeIndex itemIndex
	var readIndex itemIndex
	batch, err := newBatch(pcs).get(readIndexKey, writeIndexKey).execute(ctx)

	if err == nil {
		readIndex, err = batch.getItemIndexResult(readIndexKey)
	}

	if err == nil {
		writeIndex, err = batch.getItemIndexResult(writeIndexKey)
	}

	if err != nil {
		pcs.logger.Debug("failed getting read/write index, starting with new ones",
			zap.String(zapQueueNameKey, pcs.queueName),
			zap.Error(err))
		pcs.readIndex = 0
		pcs.writeIndex = 0
	} else {
		pcs.readIndex = readIndex
		pcs.writeIndex = writeIndex
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
	// Nil requests are ignored
	if req == nil {
		return nil
	}

	pcs.mu.Lock()
	defer pcs.mu.Unlock()

	ctx := context.Background()

	itemKey := pcs.itemKey(pcs.writeIndex)
	pcs.writeIndex++

	_, err := newBatch(pcs).setItemIndex(writeIndexKey, pcs.writeIndex).setRequest(itemKey, req).execute(ctx)
	if err != nil {
		return err
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

		batch, err := newBatch(pcs).get(pcs.itemKey(index)).execute(ctx)
		if err != nil {
			return nil, false
		}

		req, err := batch.getRequestResult(pcs.itemKey(index))
		if err != nil || req == nil {
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
	var processedItems []itemIndex

	pcs.mu.Lock()
	defer pcs.mu.Unlock()

	pcs.logger.Debug("checking if there are items left by consumers", zap.String(zapQueueNameKey, pcs.queueName))
	batch, err := newBatch(pcs).get(currentlyProcessedItemsKey).execute(ctx)
	if err == nil {
		processedItems, err = batch.getItemIndexArrayResult(currentlyProcessedItemsKey)
	}
	if err != nil {
		pcs.logger.Warn("could not fetch items left by consumers", zap.String(zapQueueNameKey, pcs.queueName))
		return reqs
	}

	if len(processedItems) > 0 {
		pcs.logger.Info("fetching items left for processing by consumers",
			zap.String(zapQueueNameKey, pcs.queueName), zap.Int(zapNumberOfItems, len(processedItems)))
	} else {
		pcs.logger.Debug("no items left for processing by consumers")
	}

	reqs = make([]request, len(processedItems))
	keys := make([]string, len(processedItems))
	batch = newBatch(pcs)
	for i, it := range processedItems {
		keys[i] = pcs.itemKey(it)
		batch.
			get(keys[i]).
			delete(keys[i])
	}

	_, err = batch.execute(ctx)
	if err != nil {
		pcs.logger.Warn("failed cleaning items left by consumers", zap.String(zapQueueNameKey, pcs.queueName))
		return reqs
	}

	for i, key := range keys {
		req, err := batch.getRequestResult(key)
		if err != nil {
			pcs.logger.Warn("failed unmarshalling item",
				zap.String(zapQueueNameKey, pcs.queueName), zap.String(zapKey, key))
		} else {
			reqs[i] = req
		}
	}

	return reqs
}

// itemProcessingStart appends the item to the list of currently processed items
func (pcs *persistentContiguousStorage) itemProcessingStart(ctx context.Context, index itemIndex) {
	pcs.currentlyProcessedItems = append(pcs.currentlyProcessedItems, index)
	_, err := newBatch(pcs).
		setItemIndexArray(currentlyProcessedItemsKey, pcs.currentlyProcessedItems).
		execute(ctx)
	if err != nil {
		pcs.logger.Warn("failed updating currently processed items",
			zap.String(zapQueueNameKey, pcs.queueName), zap.Error(err))
	}
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

	_, err := newBatch(pcs).
		setItemIndexArray(currentlyProcessedItemsKey, pcs.currentlyProcessedItems).
		delete(pcs.itemKey(index)).
		execute(ctx)
	if err != nil {
		pcs.logger.Warn("failed updating currently processed items",
			zap.String(zapQueueNameKey, pcs.queueName), zap.Error(err))
	}
}

func (pcs *persistentContiguousStorage) updateReadIndex(ctx context.Context) {
	_, err := newBatch(pcs).
		setItemIndex(readIndexKey, pcs.readIndex).
		execute(ctx)

	if err != nil {
		pcs.logger.Warn("failed updating read index",
			zap.String(zapQueueNameKey, pcs.queueName), zap.Error(err))
	}
}

func (pcs *persistentContiguousStorage) itemKey(index itemIndex) string {
	return fmt.Sprintf("%d", index)
}
