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

package shutdownhelper

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShutdownDuringCall(t *testing.T) {
	for i := 0; i < 100; i++ {
		h := NewHelper()

		var cnt int64
		ch := make(chan bool)
		var shutdownFinished int64
		go func() {
			for {
				// Protect the operation.
				if err := h.BeginOperation(); err != nil {
					assert.EqualError(t, err, errAlreadyShutdown.Error())
					// We were shutdown, stop processing.
					break
				}
				if atomic.LoadInt64(&shutdownFinished) == 1 {
					assert.Fail(t, "Shutdown() returned, but we entered processing")
				}

				// This is our operation that is protected: increment a counter.
				atomic.AddInt64(&cnt, 1)

				// Indicate we are done with the operation.
				h.EndOperation()
			}
			// Indicate that goroutine is one.
			close(ch)
		}()

		for atomic.LoadInt64(&cnt) == 0 {
			// Wait for cnt to be incremented in the goroutine. This ensures that the
			// goroutine is started.
		}

		// Now shutdown the helper.
		err := h.Shutdown(context.Background())
		atomic.StoreInt64(&shutdownFinished, 1)

		// Save the cnt value after shutdown.
		v := atomic.LoadInt64(&cnt)
		assert.NoError(t, err)

		// Wait until the goroutine finishes.
		<-ch

		// cnt should not be changed. This is the guarantee provided by Helper.
		assert.EqualValues(t, v, atomic.LoadInt64(&cnt))
	}
}

func TestShutdownDeadline(t *testing.T) {
	h := NewHelper()

	var cnt int64
	ch := make(chan bool)
	go func() {
		// Protect the operation.
		if err := h.BeginOperation(); err != nil {
			// We were shutdown, stop processing.
			return
		}

		defer h.EndOperation()

		// This is our operation that is protected: increment a counter.
		atomic.AddInt64(&cnt, 1)

		// Imitate long processing.
		<-ch

		// Indicate we are done.
		close(ch)
	}()

	for atomic.LoadInt64(&cnt) == 0 {
		// Wait for cnt to be incremented in the goroutine. This ensures that the
		// goroutine is started.
	}

	// Now shutdown the helper.
	ctx, cancelFn := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancelFn()
	err := h.Shutdown(ctx)

	// This should be timeout error.
	assert.EqualError(t, err, errTimeout.Error())

	// Tell the goroutine to exit.
	ch <- true

	// Wait for it to exit.
	<-ch
}

func TestDuplicateShutdown(t *testing.T) {
	h := NewHelper()

	err := h.Shutdown(context.Background())
	assert.NoError(t, err)

	err = h.Shutdown(context.Background())
	// This should be timeout error.
	assert.EqualError(t, err, errDuplicateShutdown.Error())
}

// This benchmark measures how much overhead the helper adds to the processing
// of each request when there is no contention.
//
// A typical result for this benchmark for reference:
// 	BenchmarkOperationsSingle-8       	100000000	        11.5 ns/op
func BenchmarkOperationsSingle(b *testing.B) {
	h := NewHelper()
	for i := 0; i < b.N; i++ {
		h.BeginOperation()
		h.EndOperation()
	}
}

// This benchmark measures how much overhead the helper adds to the processing
// of each request when we process concurrently. Note that this is the worst case
// when processing is nothing but calling BeginOperation/EndOperation. In realistic
// scenarios when there is also real work done the contention is going to be smaller.
// Typical output:
//	BenchmarkOperationsConcurrent/Concurrency=1-8         	100000000	        11.4 ns/op
//	BenchmarkOperationsConcurrent/Concurrency=2-8         	12334543	        97.0 ns/op
//	BenchmarkOperationsConcurrent/Concurrency=4-8         	 6855817	       173 ns/op
//	BenchmarkOperationsConcurrent/Concurrency=8-8         	 3543706	       339 ns/op
func BenchmarkOperationsConcurrent(b *testing.B) {
	h := NewHelper()

	for c := 1; c <= 8; c *= 2 {
		b.Run("Concurrency="+strconv.Itoa(c), func(b *testing.B) {
			wg := sync.WaitGroup{}
			for j := 0; j < c; j++ {
				wg.Add(1)
				go func() {
					for i := 0; i < b.N; i++ {
						h.BeginOperation()
						h.EndOperation()
					}
					wg.Done()
				}()
			}

			wg.Wait()
		})
	}
}
