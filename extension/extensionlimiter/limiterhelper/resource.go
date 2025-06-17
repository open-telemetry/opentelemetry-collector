// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"container/list"
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

var (
	// TODO: was grpccodes.ResourceExhausted
	ErrResourceWaitLimit = errors.New("too much waiting data")

	// TODO: was grpccodes.InvalidArgument
	ErrResourceSizeLimit = errors.New("request is too large")
)

// NewResourceLimiter returns an implementation of the
// resource-limiter extension based on a LIFO queue.
// See this [article](https://medium.com/swlh/fifo-considered-harmful-793b76f98374)
// explaining why LIFO is preferred here.
func NewResourceLimiter(admitLimit, waitLimit uint64) extensionlimiter.ResourceLimiter {
	return &boundedQueue{
		limitAdmit: admitLimit,
		limitWait:  waitLimit,
	}
}

var _ extensionlimiter.ResourceLimiter = &boundedQueue{}

type boundedQueue struct {
	limitAdmit uint64
	limitWait  uint64

	// lock protects currentAdmitted, currentWaiting, and waiters
	lock            sync.Mutex
	currentAdmitted uint64
	currentWaiting  uint64
	waiters         *list.List // of *waiter
}

// waiter is an item in the BoundedQueue waiters list.
type waiter struct {
	notify notification
	value  int
}

func (bq *boundedQueue) ReserveResource(ctx context.Context, value int) (extensionlimiter.ResourceReservation, error) {
	if uint64(value) > bq.limitAdmit {
		return nil, ErrResourceSizeLimit
	}

	bq.lock.Lock()
	defer bq.lock.Unlock()

	if bq.currentAdmitted+uint64(value) <= bq.limitAdmit {
		// the fast success path.
		bq.currentAdmitted += uint64(value)
		return struct {
			extensionlimiter.DelayFunc
			extensionlimiter.ReleaseFunc
		}{
			nil, // No delay
			func() {
				// There was never a waiter in this
				// case, just release and admit waiters.
				bq.lock.Lock()
				defer bq.lock.Unlock()

				bq.releaseLocked(value)
			},
		}, nil
	}

	// since we were unable to admit, check if we can wait.
	if bq.currentWaiting+uint64(value) > bq.limitWait {
		return nil, ErrResourceWaitLimit
	}

	// otherwise we need to wait
	element := bq.addWaiterLocked(value)
	waiter := element.Value.(*waiter)

	return struct {
		extensionlimiter.DelayFunc
		extensionlimiter.ReleaseFunc
	}{
		func() <-chan struct{} {
			// The caller waits for this notification
			// to use the resource.
			return waiter.notify.channel()
		},
		func() {
			// Called when the caller finishes.
			bq.lock.Lock()
			defer bq.lock.Unlock()

			if waiter.notify.hasBeen() {
				// We were also admitted, which can happen
				// concurrently with cancellation. Make sure
				// to release since no one else will do it.
				bq.releaseLocked(value)
			} else {
				// Remove ourselves from the list of waiters
				// so that we can't be admitted in the future.
				bq.removeWaiterLocked(value, element)
				bq.admitWaitersLocked()
			}
		},
	}, nil
}

func (bq *boundedQueue) admitWaitersLocked() {
	for bq.waiters.Len() != 0 {
		// Ensure there is enough room to admit the next waiter.
		element := bq.waiters.Back()
		waiter := element.Value.(*waiter)
		if bq.currentAdmitted+uint64(waiter.value) > bq.limitAdmit {
			// Returning means continuing to wait for the
			// most recent arrival to get service by another release.
			return
		}

		// Release the next waiter and tell it that it has been admitted.
		bq.removeWaiterLocked(waiter.value, element)
		bq.currentAdmitted += uint64(waiter.value)

		waiter.notify.notice()
	}
}

func (bq *boundedQueue) addWaiterLocked(value int) *list.Element {
	bq.currentWaiting += uint64(value)
	return bq.waiters.PushBack(&waiter{
		value:  value,
		notify: newNotification(),
	})
}

func (bq *boundedQueue) removeWaiterLocked(value int, element *list.Element) {
	bq.currentWaiting -= uint64(value)
	bq.waiters.Remove(element)
}

func (bq *boundedQueue) releaseLocked(value int) {
	bq.currentAdmitted -= uint64(value)
	bq.admitWaitersLocked()
}

// BlockingResourceLimiter wraps for ResourceLimiter extension in a
// blocking interface which considers the context deadline while
// waiting for the resource.
type BlockingResourceLimiter struct {
	limiter extensionlimiter.ResourceLimiter
}

// NewBlockingResourceLimiter returns a blocking wrapper for
// ResourceLimiter extensions.
func NewBlockingResourceLimiter(limiter extensionlimiter.ResourceLimiter) BlockingResourceLimiter {
	return BlockingResourceLimiter{
		limiter: limiter,
	}
}

// WaitFor blocks the caller until the requested value is allowed by
// the limiter.
func (b BlockingResourceLimiter) WaitFor(ctx context.Context, value int) (extensionlimiter.ReleaseFunc, error) {
	rsv, err := b.limiter.ReserveResource(ctx, value)
	if err != nil {
		return func() {}, err
	}
	select {
	case <-ctx.Done():
		rsv.Release()
		return func() {}, context.Cause(ctx)
	case <-rsv.Delay():
		return rsv.Release, nil
	}
}
