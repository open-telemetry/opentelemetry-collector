// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestQueueController_FullCapacity(t *testing.T) {
	start := make(chan struct{})
	done := make(chan struct{})
	qc := NewQueueController[fakeReq](NewBoundedMemoryQueue[fakeReq](5), NewRequestsCapacityLimiter[fakeReq](5), 1,
		func(context.Context, fakeReq) {
			<-start
			<-done
		})
	require.NoError(t, qc.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 0, qc.Size())

	req := fakeReq{itemsCount: 5}

	// Let the first request through.
	assert.NoError(t, qc.Offer(context.Background(), req))
	start <- struct{}{}
	close(start)

	assert.Eventually(t, func() bool {
		return qc.Size() == 0
	}, 1*time.Second, 10*time.Millisecond)

	for i := 0; i < 10; i++ {
		result := qc.Offer(context.Background(), req)
		if i < 5 {
			assert.NoError(t, result)
		} else {
			assert.ErrorIs(t, result, ErrQueueIsFull)
		}
	}
	assert.Eventually(t, func() bool {
		return qc.Size() == 5
	}, 1*time.Second, 10*time.Millisecond)
	close(done)

	assert.Eventually(t, func() bool {
		return qc.Size() == 0
	}, 1*time.Second, 10*time.Millisecond)
}
