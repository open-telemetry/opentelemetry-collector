// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type mockHost struct {
	component.Host
	ext map[component.ID]component.Component
}

func (nh *mockHost) GetExtensions() map[component.ID]component.Component {
	return nh.ext
}

// createTestQueue creates and starts a fake queue with the given capacity and number of consumers.
func createTestQueue(t *testing.T, capacity, numConsumers int, callback func(item Request)) ProducerConsumerQueue {
	pq := NewPersistentQueue(capacity, numConsumers, component.ID{}, newFakeTracesRequestMarshalerFunc(),
		newFakeTracesRequestUnmarshalerFunc())
	host := &mockHost{ext: map[component.ID]component.Component{
		{}: createStorageExtension(t.TempDir()),
	}}
	err := pq.Start(context.Background(), host, newNopQueueSettings(callback))
	require.NoError(t, err)
	t.Cleanup(pq.Stop)
	return pq
}

func TestPersistentQueue_Capacity(t *testing.T) {
	path := t.TempDir()

	for i := 0; i < 100; i++ {
		pq := NewPersistentQueue(5, 1, component.ID{}, newFakeTracesRequestMarshalerFunc(),
			newFakeTracesRequestUnmarshalerFunc())
		host := &mockHost{ext: map[component.ID]component.Component{
			{}: createStorageExtension(path),
		}}
		err := pq.Start(context.Background(), host, newNopQueueSettings(func(req Request) {}))
		require.NoError(t, err)

		// Stop consumer to imitate queue overflow
		close(pq.(*persistentQueue).stopChan)
		pq.(*persistentQueue).stopWG.Wait()

		assert.Equal(t, 0, pq.Size())

		traces := newTraces(1, 10)
		req := newFakeTracesRequest(traces)

		for i := 0; i < 10; i++ {
			result := pq.Produce(req)
			if i < 6 {
				assert.True(t, result)
			} else {
				assert.False(t, result)
			}

			// Let's make sure the loop picks the first element into the channel,
			// so the capacity could be used in full
			if i == 0 {
				assert.Eventually(t, func() bool {
					return pq.Size() == 0
				}, 5*time.Second, 10*time.Millisecond)
			}
		}
		assert.Equal(t, 5, pq.Size())
		stopStorage(pq.(*persistentQueue))
	}
}

func TestPersistentQueue_Close(t *testing.T) {
	wq := createTestQueue(t, 1001, 100, func(item Request) {})
	traces := newTraces(1, 10)
	req := newFakeTracesRequest(traces)

	for i := 0; i < 1000; i++ {
		wq.Produce(req)
	}
	// This will close the queue very quickly, consumers might not be able to consume anything and should finish gracefully
	assert.NotPanics(t, func() {
		wq.Stop()
	})
	// The additional stop should not panic
	assert.NotPanics(t, func() {
		wq.Stop()
	})
}

// Verify storage closes after queue consumers. If not in this order, successfully consumed items won't be updated in storage
func TestPersistentQueue_Close_StorageCloseAfterConsumers(t *testing.T) {
	wq := createTestQueue(t, 1001, 1, func(item Request) {})
	traces := newTraces(1, 10)

	lastRequestProcessedTime := time.Now()
	req := newFakeTracesRequest(traces)
	req.processingFinishedCallback = func() {
		lastRequestProcessedTime = time.Now()
	}

	fnBefore := stopStorage
	stopStorageTime := time.Now()
	stopStorage = func(queue *persistentQueue) {
		stopStorageTime = time.Now()
		queue.storage.stop()
	}

	for i := 0; i < 1000; i++ {
		wq.Produce(req)
	}
	assert.NotPanics(t, func() {
		wq.Stop()
	})
	assert.True(t, stopStorageTime.After(lastRequestProcessedTime), "storage stop time should be after last request processed time")
	stopStorage = fnBefore
}

func TestPersistentQueue_ConsumersProducers(t *testing.T) {
	cases := []struct {
		numMessagesProduced int
		numConsumers        int
	}{
		{
			numMessagesProduced: 1,
			numConsumers:        1,
		},
		{
			numMessagesProduced: 100,
			numConsumers:        1,
		},
		{
			numMessagesProduced: 100,
			numConsumers:        3,
		},
		{
			numMessagesProduced: 1,
			numConsumers:        100,
		},
		{
			numMessagesProduced: 100,
			numConsumers:        100,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("#messages: %d #consumers: %d", c.numMessagesProduced, c.numConsumers), func(t *testing.T) {
			traces := newTraces(1, 10)
			req := newFakeTracesRequest(traces)

			numMessagesConsumed := &atomic.Int32{}
			tq := createTestQueue(t, 1000, c.numConsumers, func(item Request) {
				if item != nil {
					numMessagesConsumed.Add(int32(1))
				}
			})

			for i := 0; i < c.numMessagesProduced; i++ {
				tq.Produce(req)
			}

			assert.Eventually(t, func() bool {
				return c.numMessagesProduced == int(numMessagesConsumed.Load())
			}, 5*time.Second, 10*time.Millisecond)
		})
	}
}

func newTraces(numTraces int, numSpans int) ptrace.Traces {
	traces := ptrace.NewTraces()
	batch := traces.ResourceSpans().AppendEmpty()
	batch.Resource().Attributes().PutStr("resource-attr", "some-resource")
	batch.Resource().Attributes().PutInt("num-traces", int64(numTraces))
	batch.Resource().Attributes().PutInt("num-spans", int64(numSpans))

	for i := 0; i < numTraces; i++ {
		traceID := pcommon.TraceID([16]byte{1, 2, 3, byte(i)})
		ils := batch.ScopeSpans().AppendEmpty()
		for j := 0; j < numSpans; j++ {
			span := ils.Spans().AppendEmpty()
			span.SetTraceID(traceID)
			span.SetSpanID([8]byte{1, 2, 3, byte(j)})
			span.SetName("should-not-be-changed")
			span.Attributes().PutInt("int-attribute", int64(j))
			span.Attributes().PutStr("str-attribute-1", "foobar")
			span.Attributes().PutStr("str-attribute-2", "fdslafjasdk12312312jkl")
			span.Attributes().PutStr("str-attribute-3", "AbcDefGeKKjkfdsafasdfsdasdf")
			span.Attributes().PutStr("str-attribute-4", "xxxxxx")
			span.Attributes().PutStr("str-attribute-5", "abcdef")
		}
	}

	return traces
}

func TestToStorageClient(t *testing.T) {
	getStorageClientError := errors.New("unable to create storage client")
	testCases := []struct {
		desc           string
		storage        storage.Extension
		numStorages    int
		storageIndex   int
		expectedError  error
		getClientError error
	}{
		{
			desc:          "obtain storage extension by name",
			numStorages:   2,
			storageIndex:  0,
			expectedError: nil,
		},
		{
			desc:          "fail on not existing storage extension",
			numStorages:   2,
			storageIndex:  100,
			expectedError: errNoStorageClient,
		},
		{
			desc:          "invalid extension type",
			numStorages:   2,
			storageIndex:  100,
			expectedError: errNoStorageClient,
		},
		{
			desc:           "fail on error getting storage client from extension",
			numStorages:    1,
			storageIndex:   0,
			expectedError:  getStorageClientError,
			getClientError: getStorageClientError,
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			storageID := component.NewIDWithName("file_storage", strconv.Itoa(tC.storageIndex))

			var extensions = map[component.ID]component.Component{}
			for i := 0; i < tC.numStorages; i++ {
				extensions[component.NewIDWithName("file_storage",
					strconv.Itoa(i))] = &mockStorageExtension{getClientError: tC.getClientError}
			}
			host := &mockHost{ext: extensions}
			ownerID := component.NewID("foo_exporter")

			// execute
			client, err := toStorageClient(context.Background(), storageID, host, ownerID, component.DataTypeTraces)

			// verify
			if tC.expectedError != nil {
				assert.ErrorIs(t, err, tC.expectedError)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestInvalidStorageExtensionType(t *testing.T) {
	storageID := component.NewIDWithName("extension", "extension")

	// make a test extension
	factory := extensiontest.NewNopFactory()
	extConfig := factory.CreateDefaultConfig()
	settings := extensiontest.NewNopCreateSettings()
	extension, err := factory.CreateExtension(context.Background(), settings, extConfig)
	assert.NoError(t, err)
	var extensions = map[component.ID]component.Component{
		storageID: extension,
	}
	host := &mockHost{ext: extensions}
	ownerID := component.NewID("foo_exporter")

	// execute
	client, err := toStorageClient(context.Background(), storageID, host, ownerID, component.DataTypeTraces)

	// we should get an error about the extension type
	assert.ErrorIs(t, err, errWrongExtensionType)
	assert.Nil(t, client)
}
