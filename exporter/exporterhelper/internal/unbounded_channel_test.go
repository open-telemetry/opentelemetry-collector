// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUnboundedChannel(t *testing.T) {
	in, out := newUnboundedChannel[int]()
	assert.NotNil(t, in)
	assert.NotNil(t, out)

	in <- 1
	in <- 2

	assert.Equal(t, 1, <-out)
	in <- 3
	assert.Equal(t, 2, <-out)
	assert.Equal(t, 3, <-out)
	in <- 4
	assert.Equal(t, 4, <-out)
}
