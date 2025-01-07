// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSizedCapacityChannel(t *testing.T) {
	q := newSizedChannel[int](7, nil, 0)
	require.NoError(t, q.push(1, 1, nil))
	assert.Equal(t, 1, q.Size())
	assert.Equal(t, 7, q.Capacity())

	// failed callback should not allow the element to be added
	require.Error(t, q.push(2, 2, func() error { return errors.New("failed") }))
	assert.Equal(t, 1, q.Size())

	require.NoError(t, q.push(3, 3, nil))
	assert.Equal(t, 4, q.Size())

	// should not be able to send to the full queue
	require.Error(t, q.push(4, 4, nil))
	assert.Equal(t, 4, q.Size())

	el, ok := q.pop(func(el int) int64 { return int64(el) })
	assert.Equal(t, 1, el)
	assert.True(t, ok)
	assert.Equal(t, 3, q.Size())

	el, ok = q.pop(func(el int) int64 { return int64(el) })
	assert.Equal(t, 3, el)
	assert.True(t, ok)
	assert.Equal(t, 0, q.Size())

	q.shutdown()
	el, ok = q.pop(func(el int) int64 { return int64(el) })
	assert.False(t, ok)
	assert.Equal(t, 0, el)
}

func TestSizedCapacityChannel_Offer_sizedNotFullButChannelFull(t *testing.T) {
	q := newSizedChannel[int](1, nil, 0)
	require.NoError(t, q.push(1, 1, nil))

	q.used.Store(0)
	err := q.push(1, 1, nil)
	require.Error(t, err)
	assert.Equal(t, ErrQueueIsFull, err)
}
