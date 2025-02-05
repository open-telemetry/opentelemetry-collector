// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestDisabledPassErrorBack(t *testing.T) {
	myErr := errors.New("test error")
	q := newDisabledQueue[int64](func(_ context.Context, _ int64, done Done) {
		done.OnDone(myErr)
	})
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	require.ErrorIs(t, q.Offer(context.Background(), int64(1)), myErr)
	require.NoError(t, q.Shutdown(context.Background()))
}

func TestDisabledCancelIncomingRequest(t *testing.T) {
	wg := sync.WaitGroup{}
	stop := make(chan struct{})
	q := newDisabledQueue[int64](func(_ context.Context, _ int64, done Done) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-stop
			done.OnDone(nil)
		}()
	})
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-time.After(time.Second)
		cancel()
	}()
	require.ErrorIs(t, q.Offer(ctx, int64(1)), context.Canceled)
	close(stop)
	require.NoError(t, q.Shutdown(context.Background()))
	wg.Wait()
}

func TestDisabledSizeAndCapacity(t *testing.T) {
	wg := sync.WaitGroup{}
	stop := make(chan struct{})
	q := newDisabledQueue[int64](func(_ context.Context, _ int64, done Done) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-stop
			done.OnDone(nil)
		}()
	})
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	assert.EqualValues(t, 0, q.Size())
	assert.EqualValues(t, 0, q.Capacity())
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, q.Offer(context.Background(), int64(1)))
	}()
	assert.Eventually(t, func() bool { return q.Size() == 1 }, 1*time.Second, 10*time.Millisecond)
	assert.EqualValues(t, 0, q.Capacity())
	close(stop)
	require.NoError(t, q.Shutdown(context.Background()))
	wg.Wait()
}

func TestDisabledQueueMultiThread(t *testing.T) {
	buf := newBuffer()
	buf.start()
	q := newDisabledQueue[int64](buf.consume)
	require.NoError(t, q.Start(context.Background(), componenttest.NewNopHost()))
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10_000; j++ {
				assert.NoError(t, q.Offer(context.Background(), int64(j)))
			}
		}()
	}
	wg.Wait()
	require.NoError(t, q.Shutdown(context.Background()))
	buf.shutdown()
	assert.Equal(t, int64(10*10_000), buf.consumed())
}

func BenchmarkDisabledQueueOffer(b *testing.B) {
	consumed := &atomic.Int64{}
	q := newDisabledQueue[int64](func(_ context.Context, _ int64, done Done) {
		consumed.Add(1)
		done.OnDone(nil)
	})
	require.NoError(b, q.Start(context.Background(), componenttest.NewNopHost()))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		require.NoError(b, q.Offer(context.Background(), int64(i)))
	}
	require.NoError(b, q.Shutdown(context.Background()))
	assert.Equal(b, int64(b.N), consumed.Load())
}

const flushNum = 5

type buffer struct {
	ch    chan Done
	nr    *atomic.Int64
	wg    sync.WaitGroup
	dones []Done
}

func newBuffer() *buffer {
	buf := &buffer{
		ch:    make(chan Done, 10),
		nr:    &atomic.Int64{},
		dones: make([]Done, 0, flushNum),
	}
	return buf
}

func (buf *buffer) consume(_ context.Context, _ int64, done Done) {
	buf.ch <- done
}

func (buf *buffer) start() {
	buf.wg.Add(1)
	go func() {
		defer buf.wg.Done()
		buf.dones = make([]Done, 0, flushNum)
		for {
			select {
			case done, ok := <-buf.ch:
				if !ok {
					return
				}
				buf.dones = append(buf.dones, done)
				if len(buf.dones) == flushNum {
					buf.flush()
				}
			case <-time.After(10 * time.Millisecond):
				buf.flush()
			}
		}
	}()
}

func (buf *buffer) shutdown() {
	close(buf.ch)
	buf.wg.Wait()
}

func (buf *buffer) flush() {
	if len(buf.dones) == 0 {
		return
	}
	buf.nr.Add(int64(len(buf.dones)))
	for _, done := range buf.dones {
		done.OnDone(nil)
	}
	buf.dones = buf.dones[:0]
}

func (buf *buffer) consumed() int64 {
	return buf.nr.Load()
}
