// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestsCapacityLimiter(t *testing.T) {
	rl := newQueueCapacityLimiter[fakeReq](&RequestSizer[fakeReq]{}, 2)
	assert.Equal(t, 0, rl.Size())
	assert.Equal(t, 2, rl.Capacity())

	req := fakeReq{itemsCount: 5}

	assert.True(t, rl.claim(req))
	assert.Equal(t, 1, rl.Size())

	assert.True(t, rl.claim(req))
	assert.Equal(t, 2, rl.Size())

	assert.False(t, rl.claim(req))
	assert.Equal(t, 2, rl.Size())

	rl.release(req)
	assert.Equal(t, 1, rl.Size())
}

func TestItemsCapacityLimiter(t *testing.T) {
	rl := newQueueCapacityLimiter[fakeReq](&ItemsSizer[fakeReq]{}, 7)
	assert.Equal(t, 0, rl.Size())
	assert.Equal(t, 7, rl.Capacity())

	req := fakeReq{itemsCount: 3}

	assert.True(t, rl.claim(req))
	assert.Equal(t, 3, rl.Size())

	assert.True(t, rl.claim(req))
	assert.Equal(t, 6, rl.Size())

	assert.False(t, rl.claim(req))
	assert.Equal(t, 6, rl.Size())

	rl.release(req)
	assert.Equal(t, 3, rl.Size())
}

type fakeReq struct {
	itemsCount int
}

func (r fakeReq) ItemsCount() int {
	return r.itemsCount
}
