// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestTiered_QueuesDisabled(t *testing.T) {
	qCfg := QueueSettings{Enabled: false}
	bCfg := QueueSettings{Enabled: false}
	be, err := newBaseExporter(defaultSettings, fromOptions(WithQueue(qCfg), WithBacklog(bCfg)), "", nopRequestUnmarshaler())
	require.NoError(t, err)
	require.NotNil(t, be.sender.primary)
	assert.Nil(t, be.sender.backlog)

	primaryObserver := newObservabilityConsumerSender(be.sender.primary.consumerSender)
	be.sender.primary.consumerSender = primaryObserver

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	primaryCounter := &atomic.Uint32{}
	be.sender.primary.queue = &producerConsumerQueueWithCounter{
		ProducerConsumerQueue: be.sender.primary.queue,
		produceCounter:        primaryCounter,
	}

	req := newMockRequest(context.Background(), 1, nil)
	for i := 0; i < 5; i++ {
		primaryObserver.run(func() {
			require.NoError(t, be.sender.send(req))
		})
	}

	// received by consumer sender, but not by queue
	primaryObserver.awaitAsyncProcessing()
	primaryObserver.checkSendItemsCount(t, 5)
	assert.EqualValues(t, 0, primaryCounter.Load())
}

func TestTiered_PrimaryQueueFull(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0
	qCfg.QueueSize = 0
	bCfg := NewDefaultQueueSettings()
	bCfg.NumConsumers = 1
	bCfg.QueueSize = 1
	be, err := newBaseExporter(defaultSettings, fromOptions(WithQueue(qCfg), WithBacklog(bCfg)), "", nopRequestUnmarshaler())
	require.NoError(t, err)
	require.NotNil(t, be.sender.backlog)

	be.sender.primary.requeuingEnabled = true
	backlogObserver := newObservabilityConsumerSender(be.sender.backlog.consumerSender)
	be.sender.backlog.consumerSender = backlogObserver

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(context.Background()))
	})

	mockR := newMockRequest(context.Background(), 2, nil)
	backlogObserver.run(func() {
		require.NoError(t, be.sender.send(mockR))
	})
	backlogObserver.awaitAsyncProcessing()

	assert.Zero(t, be.sender.primary.queue.Size())
	assert.Zero(t, be.sender.backlog.queue.Size())

	// only the backlog actually gets sent
	mockR.checkNumRequests(t, 1)
	backlogObserver.checkSendItemsCount(t, 2)
	backlogObserver.checkDroppedItemsCount(t, 0)
}

func TestTiered_PrimaryQueueOnError(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 1
	bCfg := NewDefaultQueueSettings()
	bCfg.NumConsumers = 1
	bCfg.QueueSize = 1
	rCfg := NewDefaultRetrySettings()
	rCfg.MaxElapsedTime = time.Nanosecond // we don't want to retry at all, but requeue instead
	be, err := newBaseExporter(defaultSettings, fromOptions(WithRetry(rCfg), WithQueue(qCfg), WithBacklog(bCfg)), "", nopRequestUnmarshaler())
	require.NoError(t, err)
	require.NotNil(t, be.sender.backlog)

	primaryObserver := newObservabilityConsumerSender(be.sender.primary.consumerSender)
	be.sender.primary.consumerSender = primaryObserver
	be.sender.primary.requeuingEnabled = true
	backlogObserver := newObservabilityConsumerSender(be.sender.backlog.consumerSender)
	be.sender.backlog.consumerSender = backlogObserver

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(context.Background()))
	})

	traceErr := consumererror.NewTraces(errors.New("some error"), testdata.GenerateTraces(1))
	mockR := newMockRequest(context.Background(), 2, traceErr)
	primaryObserver.waitGroup.Add(1)
	backlogObserver.waitGroup.Add(1)
	require.NoError(t, be.sender.send(mockR))
	backlogObserver.awaitAsyncProcessing()

	assert.Zero(t, be.sender.primary.queue.Size())
	assert.Zero(t, be.sender.backlog.queue.Size())

	// error on primary, success on backlog
	mockR.checkNumRequests(t, 2)
	// original request dropped
	primaryObserver.checkSendItemsCount(t, 0)
	primaryObserver.checkDroppedItemsCount(t, 2)
	// backlog sends OnError request instead
	backlogObserver.checkSendItemsCount(t, 1)
	backlogObserver.checkDroppedItemsCount(t, 0)
}

func TestTiered_BacklogQueueFull(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0
	qCfg.QueueSize = 0
	bCfg := NewDefaultQueueSettings()
	bCfg.NumConsumers = 0
	bCfg.QueueSize = 0
	be, err := newBaseExporter(defaultSettings, fromOptions(WithQueue(qCfg), WithBacklog(bCfg)), "", nopRequestUnmarshaler())
	require.NoError(t, err)
	require.NotNil(t, be.sender.backlog)
	be.sender.primary.requeuingEnabled = true
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(context.Background()))
	})

	mockR := newMockRequest(context.Background(), 2, nil)
	require.NoError(t, be.sender.send(mockR))

	// replace overflow function to add counter
	overflowCounter := &atomic.Uint32{}
	be.sender.backlog.queue.OnOverflow(func(item internal.Request) {
		overflowCounter.Add(1)
	})
	require.NoError(t, be.sender.send(mockR))
	assert.Eventually(t, func() bool {
		return overflowCounter.Load() == uint32(1)
	}, time.Second, 1*time.Millisecond)
}

func TestTiered_BacklogProduceError(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0
	qCfg.QueueSize = 0
	bCfg := NewDefaultQueueSettings()
	bCfg.NumConsumers = 0
	bCfg.QueueSize = 0
	be, err := newBaseExporter(defaultSettings, fromOptions(WithQueue(qCfg), WithBacklog(bCfg)), "", nopRequestUnmarshaler())
	require.NoError(t, err)
	require.NotNil(t, be.sender.backlog)

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	droppedCounter := &atomic.Uint32{}
	be.onDropped(func(req internal.Request) {
		droppedCounter.Add(1)
	})
	be.sender.backlog.cfg.Enabled = false
	backlogObserver := newObservabilityConsumerSender(&errorRequestSender{
		errToReturn: errors.New("some error"),
	})
	be.sender.backlog.consumerSender = backlogObserver

	req := newMockRequest(context.Background(), 1, nil)
	backlogObserver.run(func() {
		require.NoError(t, be.sender.send(req))
	})
	backlogObserver.awaitAsyncProcessing()
	backlogObserver.checkDroppedItemsCount(t, 1)
	assert.Eventually(t, func() bool {
		return droppedCounter.Load() == uint32(1)
	}, time.Second, 1*time.Millisecond)
}

func TestTiered_BacklogStartError(t *testing.T) {
	storageError := errors.New("could not get storage client")

	qCfg := NewDefaultQueueSettings()
	bCfg := NewDefaultQueueSettings()
	storageID := component.NewIDWithName("file_storage", "storage")
	bCfg.StorageID = &storageID // enable persistence
	be, err := newBaseExporter(defaultSettings, fromOptions(WithQueue(qCfg), WithBacklog(bCfg)), "", nopRequestUnmarshaler())
	require.NoError(t, err)

	var extensions = map[component.ID]component.Component{
		storageID: &mockStorageExtension{GetClientError: storageError},
	}
	host := &mockHost{ext: extensions}

	// we fail to start if we get an error creating the storage client
	assert.Error(t, be.Start(context.Background(), host), "could not get storage client")
}

func TestTiered_BacklogEnabledWithoutPrimary(t *testing.T) {
	bCfg := NewDefaultQueueSettings()
	bCfg.NumConsumers = 0
	bCfg.QueueSize = 0
	be, err := newBaseExporter(defaultSettings, fromOptions(WithBacklog(bCfg)), "", nopRequestUnmarshaler())
	require.NoError(t, err)
	assert.NotNil(t, be.sender.primary)
	assert.Nil(t, be.sender.backlog)
}

func TestTiered_Shutdown_PrimaryRequestsFlushedToBacklog(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	bCfg := NewDefaultQueueSettings()
	bCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	rCfg.InitialInterval = time.Millisecond
	rCfg.MaxElapsedTime = 0
	be, err := newBaseExporter(defaultSettings, fromOptions(WithRetry(rCfg), WithQueue(qCfg), WithBacklog(bCfg)), "", nopRequestUnmarshaler())
	require.NoError(t, err)
	require.NotNil(t, be.sender.backlog)

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	primaryCounter := &atomic.Uint32{}
	be.sender.primary.queue = &producerConsumerQueueWithCounter{
		ProducerConsumerQueue: be.sender.primary.queue,
		produceCounter:        primaryCounter,
	}
	be.sender.primary.requeuingEnabled = true
	backlogCounter := &atomic.Uint32{}
	be.sender.backlog.queue = &producerConsumerQueueWithCounter{
		ProducerConsumerQueue: be.sender.backlog.queue,
		produceCounter:        backlogCounter,
	}
	be.sender.backlog.requeuingEnabled = true

	// replace nextSender inside retrySender to always return error so it doesn't exit send loop
	be.sender.wrapConsumerSender(func(consumer requestSender) requestSender {
		consumer.(*retrySender).nextSender = &errorRequestSender{
			errToReturn: errors.New("some error"),
		}
		return consumer
	})

	for i := 0; i < 5; i++ {
		req := newMockRequest(context.Background(), i, errors.New("some error"))
		require.NoError(t, be.sender.send(req))
	}

	// first wait for all the items to be produced to the queue initially
	assert.Eventually(t, func() bool {
		return primaryCounter.Load() == uint32(5)
	}, time.Second, 1*time.Millisecond)

	// shuts down and ensure the item is produced in the backlog queue again
	require.NoError(t, be.Shutdown(context.Background()))
	assert.Eventually(t, func() bool {
		return backlogCounter.Load() == uint32(5)
	}, time.Second, 1*time.Millisecond)
}
