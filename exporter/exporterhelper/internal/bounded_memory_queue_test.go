// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component/componenttest"
)

type stringRequest struct {
	Request
	str string
}

func newStringRequest(str string) Request {
	return stringRequest{str: str}
}

// In this test we run a queue with capacity 1 and we want to test the overflow behavior.
func TestBoundedQueue(t *testing.T) {
	q := NewBoundedMemoryQueue(1)

	assert.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))

	// Offer two items. The first one should be accepted.
	assert.True(t, q.Offer(newStringRequest("a")))
	assert.Equal(t, 1, q.Size())
	// the second should be rejected since the queue is full
	assert.False(t, q.Offer(newStringRequest("b")))
	assert.Equal(t, 1, q.Size())

	pollAndCheck(t, q, "a")
	assert.Equal(t, 0, q.Size())

	// Add more items and poll
	for _, item := range []string{"d", "e", "f"} {
		assert.True(t, q.Offer(newStringRequest(item)))
		pollAndCheck(t, q, item)
	}

	assert.NoError(t, q.Shutdown(context.Background()))
	assert.False(t, q.Offer(newStringRequest("x")))
	_, ok := <-q.Poll()
	assert.False(t, ok)
}

func pollAndCheck(t *testing.T, q Queue, result string) {
	newReq, ok := <-q.Poll()
	assert.True(t, ok)
	assert.Equal(t, result, newReq.(stringRequest).str)
}

// In this test we run a queue with many items when the queue is stopped,
// the remaining items should be processable.
func TestShutdownWhileNotEmpty(t *testing.T) {
	q := NewBoundedMemoryQueue(10)

	assert.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))

	q.Offer(newStringRequest("a"))
	q.Offer(newStringRequest("b"))
	q.Offer(newStringRequest("c"))
	q.Offer(newStringRequest("d"))
	q.Offer(newStringRequest("e"))
	q.Offer(newStringRequest("f"))
	q.Offer(newStringRequest("g"))
	q.Offer(newStringRequest("h"))
	q.Offer(newStringRequest("i"))
	q.Offer(newStringRequest("j"))

	assert.NoError(t, q.Shutdown(context.Background()))
	assert.False(t, q.Offer(newStringRequest("x")))

	for _, item := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		pollAndCheck(t, q, item)
	}
	assert.Equal(t, 0, q.Size())
}

func TestZeroSize(t *testing.T) {
	q := NewBoundedMemoryQueue(0)

	err := q.Start(context.Background(), componenttest.NewNopHost())
	assert.NoError(t, err)

	assert.False(t, q.Offer(newStringRequest("a"))) // in process
	assert.NoError(t, q.Shutdown(context.Background()))
}

func Benchmark_QueueUsage_10000_1_50000(b *testing.B) {
	queueUsage(b, 10000, 1, 50000)
}

func Benchmark_QueueUsage_10000_2_50000(b *testing.B) {
	queueUsage(b, 10000, 2, 50000)
}
func Benchmark_QueueUsage_10000_5_50000(b *testing.B) {
	queueUsage(b, 10000, 5, 50000)
}
func Benchmark_QueueUsage_10000_10_50000(b *testing.B) {
	queueUsage(b, 10000, 10, 50000)
}

func Benchmark_QueueUsage_50000_1_50000(b *testing.B) {
	queueUsage(b, 50000, 1, 50000)
}

func Benchmark_QueueUsage_50000_2_50000(b *testing.B) {
	queueUsage(b, 50000, 2, 50000)
}
func Benchmark_QueueUsage_50000_5_50000(b *testing.B) {
	queueUsage(b, 50000, 5, 50000)
}
func Benchmark_QueueUsage_50000_10_50000(b *testing.B) {
	queueUsage(b, 50000, 10, 50000)
}

func Benchmark_QueueUsage_10000_1_250000(b *testing.B) {
	queueUsage(b, 10000, 1, 250000)
}

func Benchmark_QueueUsage_10000_2_250000(b *testing.B) {
	queueUsage(b, 10000, 2, 250000)
}
func Benchmark_QueueUsage_10000_5_250000(b *testing.B) {
	queueUsage(b, 10000, 5, 250000)
}
func Benchmark_QueueUsage_10000_10_250000(b *testing.B) {
	queueUsage(b, 10000, 10, 250000)
}

func queueUsage(b *testing.B, capacity int, numConsumers int, numberOfItems int) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := NewBoundedMemoryQueue(capacity)
		require.NoError(b, q.Start(context.Background(), componenttest.NewNopHost()))
		stopWG := sync.WaitGroup{}
		startConsumers(q, numConsumers, &stopWG, func(request Request) {
			time.Sleep(1 * time.Millisecond)
		})
		for j := 0; j < numberOfItems; j++ {
			q.Offer(newStringRequest(fmt.Sprintf("%d", j)))
		}
		require.NoError(b, q.Shutdown(context.Background()))
		stopWG.Wait()
	}
}

func startConsumers(queue Queue, numConsumers int, stopWG *sync.WaitGroup, consume func(request Request)) {
	var startWG sync.WaitGroup
	for i := 0; i < numConsumers; i++ {
		stopWG.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			defer stopWG.Done()
			for item := range queue.Poll() {
				consume(item)
			}
		}()
	}
	startWG.Wait()
}
