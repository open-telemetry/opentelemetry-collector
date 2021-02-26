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
	"errors"
	"sync"
	"sync/atomic"
)

var errTimeout = errors.New("shutdown timed out while waiting for operation to complete")
var errDuplicateShutdown = errors.New("shutdown attempted more than once")
var errAlreadyShutdown = errors.New("trying to operate when already shutdown")

// Helper for receivers to implement Shutdown() that ensures they are no longer
// producing data.
type Helper struct {
	shutdownFlag     int64 // 0=false, 1=true
	cond             *sync.Cond
	activeOperations int64
}

func NewHelper() Helper {
	return Helper{cond: sync.NewCond(&sync.Mutex{})}
}

// Shutdown() waits for the completion of any ongoing operations indicated via
// BeginOperation/EndOperation calls and then returns. Shutdown() waits up to the
// deadline specified in the ctx. If the deadline is exceeded Shutdown() returns with
// an error.
// Calling Shutdown() after it was already called will immediately return an error.
// Once Shutdown() is called any subsequent calls to BeginOperation() will return an error.
func (h *Helper) Shutdown(ctx context.Context) error {
	// Indicate that we are shutdown.
	if !atomic.CompareAndSwapInt64(&h.shutdownFlag, 0, 1) {
		// Didn't swap because we are already shutdown.
		return errDuplicateShutdown
	}

	// Wait until all active processing is completed.
	doneWaitingForProcessing := make(chan struct{})
	go func() {
		// Wait until there are no active processes.
		h.cond.L.Lock()
		for atomic.LoadInt64(&h.activeOperations) > 0 {
			h.cond.Wait()
		}
		h.cond.L.Unlock()

		// Signal to outer func that we are done waiting.
		close(doneWaitingForProcessing)
	}()

	select {
	case <-ctx.Done():
		return errTimeout
	case <-doneWaitingForProcessing:
	}
	return nil
}

func (h *Helper) isShutdown() bool {
	return atomic.LoadInt64(&h.shutdownFlag) != 0
}

// BeginOperation must be called to indicate the start of an operation that
// must be protected. If Shutdown() function is called while the operation is in
// the Shutdown() function will block until the operation is indicated to be complete
// by calling EndOperation().
// Will return an error if Shutdown() is already called.
func (h *Helper) BeginOperation() error {
	// Increment the counter of active operations.
	atomic.AddInt64(&h.activeOperations, 1)

	// Note that we must check shutdown flag after (not before) we increment the operation
	// counter. If we check it before we may let Shutdown() finish while we also allow to
	// enter processing, which is wrong.
	if h.isShutdown() {
		h.EndOperation()
		return errAlreadyShutdown
	}

	return nil
}

// EndOperation must be called to indicate the end of an  operation. See
// BeginOperation for help.
func (h *Helper) EndOperation() {
	// Decrement the counter that indicates how many active operations exist currently.
	if atomic.AddInt64(&h.activeOperations, -1) > 0 {
		// Optimize for the case when activeOperations > 0. This is important for
		// the cases when there is significant concurrency in calling BeginOperation/
		// EndOperation. For concurrency of 8 and very high contention this optimization
		// reduces waiting time by 3x (see BenchmarkOperationsConcurrent).
		// Since there are no other active operations we can safely return, there is no
		// need to try to Signal to Shutdown().
		return
	}

	if !h.isShutdown() {
		// The Shutdown() did not yet start, there is no need to signal via cond.
		// The Shutdown() will check activeOperations and will see that is 0.
		// This optimization primarily helps the non-concurrent case and gives
		// about 3x reduction in waiting time for non-concurrent case.
		return
	}

	// Shutdown() has started and may be waiting for activeOperations to become 0.
	// We need to signal and wake up the condition checking loop in Shutdown().

	// Acquire lock to ensure we don't end up with "lost wakeup" problem (calling Signal()
	// while the other party is not in Wait(), in which case Shutdown() will be stuck).
	h.cond.L.Lock()
	h.cond.Signal()
	h.cond.L.Unlock()
}
