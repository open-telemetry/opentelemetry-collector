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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaegertracing/jaeger/pkg/queue"
	"github.com/nsqio/go-diskqueue"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// WALQueue holds the WAL-backed queue
type WALQueue struct {
	logger      *zap.Logger
	stopWG      sync.WaitGroup
	unmarshaler requestUnmarshaler
	wal         diskqueue.Interface
	numWorkers  int
	stopped     *atomic.Uint32
	quit        chan struct{}
	quitOnce    sync.Once
}

const (
	// TODO: expose those as configuration parameters
	maxBytesPerFile = int64(1 << 28)
	minMsgSize      = int32(1)
	maxMsgSize      = int32(1 << 27)
	syncEvery       = int64(1000)
)

var (
	errNotFound = errors.New("not found")
)

func dqLogOption(logger *zap.Logger, lvl diskqueue.LogLevel, f string, args ...interface{}) {
	if logger == nil {
		return
	}

	switch lvl {
	case diskqueue.DEBUG:
		logger.Debug(fmt.Sprintf(f, args...))
	case diskqueue.INFO:
		logger.Info(fmt.Sprintf(f, args...))
	case diskqueue.WARN:
		logger.Warn(fmt.Sprintf(f, args...))
	case diskqueue.ERROR:
		logger.Error(fmt.Sprintf(f, args...))
	case diskqueue.FATAL:
		logger.Fatal(fmt.Sprintf(f, args...))
	}
}

func newWALQueue(logger *zap.Logger, path string, id string, syncFrequency time.Duration, unmarshaler requestUnmarshaler) (*WALQueue, error) {
	syncOption := syncFrequency == 0
	syncEveryOption := syncEvery
	if syncOption {
		syncEveryOption = 1
		// FIXME: this must be always a positive number, lets figure out something nicer
		syncFrequency = 1 * time.Second
	}

	walPath := filepath.Join(path, "wal")
	err := os.MkdirAll(walPath, 0700)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	// TODO: sanitize id?
	_wal := diskqueue.New(id, walPath,
		maxBytesPerFile, minMsgSize, maxMsgSize, syncEveryOption, syncFrequency,
		func(lvl diskqueue.LogLevel, f string, args ...interface{}) {
			dqLogOption(logger, lvl, f, args)
		})

	wq := &WALQueue{
		logger:      logger,
		wal:         _wal,
		unmarshaler: unmarshaler,
		stopped:     atomic.NewUint32(0),
		quit:        make(chan struct{}),
	}

	return wq, nil
}

// StartConsumers starts the given number of consumers which will be consuming items
func (wq *WALQueue) StartConsumers(num int, callback func(item interface{})) {
	wq.numWorkers = num
	var startWG sync.WaitGroup

	factory := func() queue.Consumer {
		return queue.ConsumerFunc(callback)
	}

	for i := 0; i < wq.numWorkers; i++ {
		wq.stopWG.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			defer wq.stopWG.Done()
			consumer := factory()

			for {
				if wq.stopped.Load() != 0 {
					return
				}
				req, err := wq.get()
				if err == errNotFound || req == nil {
					time.Sleep(1 * time.Second)
				} else {
					consumer.Consume(req)
				}
			}
		}()
	}
	startWG.Wait()
}

// Produce adds an item to the queue and returns true if it was accepted
func (wq *WALQueue) Produce(item interface{}) bool {
	if wq.stopped.Load() != 0 {
		return false
	}

	err := wq.put(item.(request))
	return err == nil
}

// Stop stops accepting items and shuts-down the queue and closes the WAL
func (wq *WALQueue) Stop() {
	wq.logger.Debug("Stopping WAL")
	wq.stopped.Store(1)
	wq.quitOnce.Do(func() {
		close(wq.quit)
	})
	wq.stopWG.Wait()
	err := wq.close()
	if err != nil {
		wq.logger.Error("Error when closing WAL", zap.Error(err))
	}
}

// Size returns the current depth of the queue
func (wq *WALQueue) Size() int {
	return int(wq.wal.Depth())
}

// put marshals the request and puts it into the WAL
func (wq *WALQueue) put(req request) error {
	bytes, err := req.marshall()
	if err != nil {
		return err
	}

	writeErr := wq.wal.Put(bytes)
	if writeErr != nil {
		return writeErr
	}

	return nil
}

// get returns the next request from the queue; note that it might be blocking if there are no entries in the WAL
func (wq *WALQueue) get() (request, error) {
	// TODO: Consider making it nonblocking, e.g. timeout after 1 second or so?
	// ticker := time.NewTicker(1*time.Second)
	// case <-ticker.C....

	select {
	case bytes := <-wq.wal.ReadChan():
		return wq.unmarshaler(bytes)
	case <-wq.quit:
		return nil, nil
	}
}

func (wq *WALQueue) close() error {
	return wq.wal.Close()
}
