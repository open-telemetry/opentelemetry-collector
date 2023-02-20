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

package internal

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func createTestQueue(extension storage.Extension, capacity int) *persistentQueue {
	logger := zap.NewNop()

	client, err := extension.GetClient(context.Background(), component.KindReceiver, component.ID{}, "")
	if err != nil {
		panic(err)
	}

	wq := NewPersistentQueue(context.Background(), "foo", component.DataTypeTraces, capacity, logger, client, newFakeTracesRequestUnmarshalerFunc())
	return wq.(*persistentQueue)
}

func TestPersistentQueue_Capacity(t *testing.T) {
	path := t.TempDir()

	for i := 0; i < 100; i++ {
		ext := createStorageExtension(path)
		t.Cleanup(func() { require.NoError(t, ext.Shutdown(context.Background())) })

		wq := createTestQueue(ext, 5)
		assert.Equal(t, 0, wq.Size())

		traces := newTraces(1, 10)
		req := newFakeTracesRequest(traces)

		for i := 0; i < 10; i++ {
			result := wq.Produce(req)
			if i < 6 {
				assert.True(t, result)
			} else {
				assert.False(t, result)
			}

			// Let's make sure the loop picks the first element into the channel,
			// so the capacity could be used in full
			if i == 0 {
				assert.Eventually(t, func() bool {
					return wq.Size() == 0
				}, 5*time.Second, 10*time.Millisecond)
			}
		}
		assert.Equal(t, 5, wq.Size())
	}
}

func TestPersistentQueue_Close(t *testing.T) {
	path := t.TempDir()

	ext := createStorageExtension(path)
	t.Cleanup(func() { assert.NoError(t, ext.Shutdown(context.Background())) })

	wq := createTestQueue(ext, 1001)
	traces := newTraces(1, 10)
	req := newFakeTracesRequest(traces)

	wq.StartConsumers(100, func(item Request) {})

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
	path := t.TempDir()

	ext := createStorageExtension(path)
	t.Cleanup(func() { assert.NoError(t, ext.Shutdown(context.Background())) })

	wq := createTestQueue(ext, 1001)
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

	wq.StartConsumers(1, func(item Request) {})

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
			path := t.TempDir()

			traces := newTraces(1, 10)
			req := newFakeTracesRequest(traces)

			ext := createStorageExtension(path)
			tq := createTestQueue(ext, 5000)

			defer tq.Stop()
			t.Cleanup(func() { assert.NoError(t, ext.Shutdown(context.Background())) })

			numMessagesConsumed := &atomic.Int32{}
			tq.StartConsumers(c.numConsumers, func(item Request) {
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
