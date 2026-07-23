// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSizeCacheBytesOnly(t *testing.T) {
	sc := NewSizeCache()

	_, ok := sc.Get()
	assert.False(t, ok)

	sc.Set(100)
	size, ok := sc.Get()
	require.True(t, ok)
	assert.Equal(t, 100, size)

	sc.Invalidate()
	_, ok = sc.Get()
	assert.False(t, ok)

	sc.Set(0)
	size, ok = sc.Get()
	require.True(t, ok)
	assert.Equal(t, 0, size)
}
