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
	"fmt"
	"sync"
	"time"

	"github.com/jaegertracing/jaeger/pkg/queue"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/extension/storage"
)

// persistentQueue holds the queue backed by file storage
type persistentQueue struct {
	logger     *zap.Logger
	stopWG     sync.WaitGroup
	produceMu  sync.Mutex
	exitChan   chan int
	capacity   int
	numWorkers int

	storage persistentStorage
}

// persistentStorage provides an interface for request storage operations
type persistentStorage interface {
	// put appends the request to the storage
	put(req request) error
	// get returns the next available request; not the channel is unbuffered
	get() <-chan request
	// size returns the current size of the storage in number of requets
	size() int
	// stop gracefully stops the storage
	stop()
}

// persistentContiguousStorage provides a persistent queue implementation backed by file storage extension
//
// Write index describes the position at which next item is going to be stored
// Read index describes which item needs to be read next
// When Write index = Read index, no elements are in the queue
//
//   ┌─file extension-backed queue─┐
//   │                             │
//   │     ┌───┐     ┌───┐ ┌───┐   │
//   │ n+1 │ n │ ... │ 4 │ │ 3 │   │
//   │     └───┘     └───┘ └─x─┘   │
//   │                       x     │
//   └───────────────────────x─────┘
//      ▲              ▲     x
//      │              │     xxx deleted
//      │              │
//    write          read
//    index          index
//
type persistentContiguousStorage struct {
	logger        *zap.Logger
	client        storage.Client
	mu            sync.Mutex
	queueName     string
	readIndex     uint64
	writeIndex    uint64
	readIndexKey  string
	writeIndexKey string
	retryDelay    time.Duration
	unmarshaler   requestUnmarshaler
	reqChan       chan request
	stopped       *atomic.Uint32
}

const (
	queueNameKey      = "queueName"
	zapItemKey        = "itemKey"
	itemKeyTemplate   = "it-%d"
	readIndexKey      = "ri"
	writeIndexKey     = "wi"
	defaultRetryDelay = 500 * time.Millisecond
)

// newPersistentQueue creates a new queue backed by file storage
func newPersistentQueue(ctx context.Context, name string, capacity int, logger *zap.Logger, client storage.Client, unmarshaler requestUnmarshaler) *persistentQueue {
	return &persistentQueue{
		logger:   logger,
		exitChan: make(chan int),
		capacity: capacity,
		storage:  newPersistentContiguousStorage(ctx, name, logger, client, unmarshaler),
	}
}

// StartConsumers starts the given number of consumers which will be consuming items
func (pq *persistentQueue) StartConsumers(num int, callback func(item interface{})) {
	pq.numWorkers = num
	var startWG sync.WaitGroup

	factory := func() queue.Consumer {
		return queue.ConsumerFunc(callback)
	}

	for i := 0; i < pq.numWorkers; i++ {
		pq.stopWG.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			defer pq.stopWG.Done()
			consumer := factory()

			for {
				select {
				case req := <-pq.storage.get():
					consumer.Consume(req)
				case <-pq.exitChan:
					return
				}
			}
		}()
	}
	startWG.Wait()
}

// Produce adds an item to the queue and returns true if it was accepted
func (pq *persistentQueue) Produce(item interface{}) bool {
	pq.produceMu.Lock()
	defer pq.produceMu.Unlock()

	if pq.storage.size() >= pq.capacity {
		return false
	}
	err := pq.storage.put(item.(request))
	return err == nil
}

// Stop stops accepting items, shuts down the queue and closes the persistent queue
func (pq *persistentQueue) Stop() {
	pq.storage.stop()
	close(pq.exitChan)
	pq.stopWG.Wait()
}

// Size returns the current depth of the queue
func (pq *persistentQueue) Size() int {
	return pq.storage.size()
}

// newPersistentContiguousStorage creates a new file-storage extension backed queue. It needs to be initialized separately
func newPersistentContiguousStorage(ctx context.Context, queueName string, logger *zap.Logger, client storage.Client, unmarshaler requestUnmarshaler) *persistentContiguousStorage {
	wcs := &persistentContiguousStorage{
		logger:      logger,
		client:      client,
		queueName:   queueName,
		unmarshaler: unmarshaler,
		reqChan:     make(chan request),
		stopped:     atomic.NewUint32(0),
		retryDelay:  defaultRetryDelay,
	}
	initPersistentContiguousStorage(ctx, wcs)
	go wcs.loop(ctx)
	return wcs
}

func initPersistentContiguousStorage(ctx context.Context, wcs *persistentContiguousStorage) {
	wcs.readIndexKey = wcs.buildReadIndexKey()
	wcs.writeIndexKey = wcs.buildWriteIndexKey()

	readIndexBytes, err := wcs.client.Get(ctx, wcs.readIndexKey)
	if err != nil || readIndexBytes == nil {
		wcs.logger.Debug("failed getting read index, starting with a new one", zap.String(queueNameKey, wcs.queueName))
		wcs.readIndex = 0
	} else {
		val, conversionErr := bytesToUint64(readIndexBytes)
		if conversionErr != nil {
			wcs.logger.Warn("read index corrupted, starting with a new one", zap.String(queueNameKey, wcs.queueName))
			wcs.readIndex = 0
		} else {
			wcs.readIndex = val
		}
	}

	writeIndexBytes, err := wcs.client.Get(ctx, wcs.writeIndexKey)
	if err != nil || writeIndexBytes == nil {
		wcs.logger.Debug("failed getting write index, starting with a new one", zap.String(queueNameKey, wcs.queueName))
		wcs.writeIndex = 0
	} else {
		val, conversionErr := bytesToUint64(writeIndexBytes)
		if conversionErr != nil {
			wcs.logger.Warn("write index corrupted, starting with a new one", zap.String(queueNameKey, wcs.queueName))
			wcs.writeIndex = 0
		} else {
			wcs.writeIndex = val
		}
	}
}

// Put marshals the request and puts it into the persistent queue
func (pcs *persistentContiguousStorage) put(req request) error {
	buf, err := req.marshal()
	if err != nil {
		return err
	}

	pcs.mu.Lock()
	defer pcs.mu.Unlock()

	itemKey := pcs.buildItemKey(pcs.writeIndex)
	err = pcs.client.Set(context.Background(), itemKey, buf)
	if err != nil {
		return err
	}

	pcs.writeIndex++
	writeIndexBytes, err := uint64ToBytes(pcs.writeIndex)
	if err != nil {
		pcs.logger.Warn("failed converting write index uint64 to bytes", zap.Error(err))
	}

	return pcs.client.Set(context.Background(), pcs.writeIndexKey, writeIndexBytes)
}

// getNextItem pulls the next available item from the persistent storage; if none is found, returns (nil, false)
func (pcs *persistentContiguousStorage) getNextItem(ctx context.Context) (request, bool) {
	pcs.mu.Lock()
	defer pcs.mu.Unlock()

	if pcs.readIndex < pcs.writeIndex {
		itemKey := pcs.buildItemKey(pcs.readIndex)
		// Increase here, so despite errors it would still progress
		pcs.readIndex++

		buf, err := pcs.client.Get(ctx, itemKey)
		if err != nil {
			pcs.logger.Error("error when getting item from persistent storage",
				zap.String(queueNameKey, pcs.queueName), zap.String(zapItemKey, itemKey), zap.Error(err))
			return nil, false
		}
		req, unmarshalErr := pcs.unmarshaler(buf)
		pcs.updateItemRead(ctx, itemKey)
		if unmarshalErr != nil {
			pcs.logger.Error("error when unmarshalling item from persistent storage",
				zap.String(queueNameKey, pcs.queueName), zap.String(zapItemKey, itemKey), zap.Error(unmarshalErr))
			return nil, false
		}

		return req, true
	}

	return nil, false
}

func (pcs *persistentContiguousStorage) loop(ctx context.Context) {
	for {
		if pcs.stopped.Load() != 0 {
			return
		}

		req, found := pcs.getNextItem(ctx)
		if found {
			pcs.reqChan <- req
		} else {
			time.Sleep(pcs.retryDelay)
		}
	}
}

// get returns the next request from the queue as available via the channel
func (pcs *persistentContiguousStorage) get() <-chan request {
	return pcs.reqChan
}

func (pcs *persistentContiguousStorage) size() int {
	pcs.mu.Lock()
	defer pcs.mu.Unlock()
	return int(pcs.writeIndex - pcs.readIndex)
}

func (pcs *persistentContiguousStorage) stop() {
	pcs.logger.Debug("stopping persistentContiguousStorage", zap.String(queueNameKey, pcs.queueName))
	pcs.stopped.Store(1)
}

func (pcs *persistentContiguousStorage) updateItemRead(ctx context.Context, itemKey string) {
	err := pcs.client.Delete(ctx, itemKey)
	if err != nil {
		pcs.logger.Debug("failed deleting item", zap.String(zapItemKey, itemKey))
	}

	readIndexBytes, err := uint64ToBytes(pcs.readIndex)
	if err != nil {
		pcs.logger.Warn("failed converting read index uint64 to bytes", zap.Error(err))
	} else {
		err = pcs.client.Set(ctx, pcs.readIndexKey, readIndexBytes)
		if err != nil {
			pcs.logger.Warn("failed storing read index", zap.Error(err))
		}
	}
}

func (pcs *persistentContiguousStorage) buildItemKey(index uint64) string {
	return fmt.Sprintf(itemKeyTemplate, index)
}

func (pcs *persistentContiguousStorage) buildReadIndexKey() string {
	return readIndexKey
}

func (pcs *persistentContiguousStorage) buildWriteIndexKey() string {
	return writeIndexKey
}

func uint64ToBytes(val uint64) ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, val)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), err
}

func bytesToUint64(b []byte) (uint64, error) {
	var val uint64
	err := binary.Read(bytes.NewReader(b), binary.LittleEndian, &val)
	if err != nil {
		return val, err
	}
	return val, nil
}
