// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
)

const flushNum = 5

type buffer struct {
	ch    chan DoneCallback
	nr    *atomic.Int64
	wg    sync.WaitGroup
	dones []DoneCallback
}

func newBuffer() *buffer {
	buf := &buffer{
		ch:    make(chan DoneCallback, 10),
		nr:    &atomic.Int64{},
		dones: make([]DoneCallback, 0, flushNum),
	}
	return buf
}

func (buf *buffer) consume(_ context.Context, _ int64, done DoneCallback) {
	buf.ch <- done
}

func (buf *buffer) start() {
	buf.wg.Add(1)
	go func() {
		defer buf.wg.Done()
		buf.dones = make([]DoneCallback, 0, flushNum)
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
		done(nil)
	}
	buf.dones = buf.dones[:0]
}

func (buf *buffer) consumed() int64 {
	return buf.nr.Load()
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
				require.NoError(t, q.Offer(context.Background(), int64(j)))
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
	q := newDisabledQueue[int64](func(_ context.Context, _ int64, done DoneCallback) {
		consumed.Add(1)
		done(nil)
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
