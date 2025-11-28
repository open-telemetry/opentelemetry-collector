// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package arc

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testAlpha   = 0.4
	testEpsilon = 1e-9
)

func TestRobustEWMA_Initial(t *testing.T) {
	e := newRobustEWMA(testAlpha)
	assert.False(t, e.initialized())
	assert.InDelta(t, testAlpha, e.alpha, testEpsilon)

	// First update initializes
	mean, absDev := e.update(10)
	assert.True(t, e.initialized())
	assert.InDelta(t, 10.0, mean, testEpsilon)
	assert.InDelta(t, 0.0, absDev, testEpsilon)
	assert.InDelta(t, 10.0, e.mean, testEpsilon)
	assert.InDelta(t, 0.0, e.absDev, testEpsilon)
}

func TestRobustEWMA_Updates(t *testing.T) {
	e := newRobustEWMA(testAlpha)

	// Init
	e.update(10)

	// Second update
	// mean = 0.4 * 20 + 0.6 * 10 = 8 + 6 = 14
	// resid = 20 - 10 = 10
	// absDev = 0.4 * 10 + 0.6 * 0 = 4
	mean, absDev := e.update(20)
	assert.InDelta(t, 14.0, mean, testEpsilon)
	assert.InDelta(t, 4.0, absDev, testEpsilon)
	assert.InDelta(t, 14.0, e.mean, testEpsilon)
	assert.InDelta(t, 4.0, e.absDev, testEpsilon)

	// Third update
	// prev = 14
	// mean = 0.4 * 12 + 0.6 * 14 = 4.8 + 8.4 = 13.2
	// resid = 12 - 14 = -2 -> 2 (abs)
	// absDev = 0.4 * 2 + 0.6 * 4 = 0.8 + 2.4 = 3.2
	mean, absDev = e.update(12)
	assert.InDelta(t, 13.2, mean, testEpsilon)
	assert.InDelta(t, 3.2, absDev, testEpsilon)
	assert.InDelta(t, 13.2, e.mean, testEpsilon)
	assert.InDelta(t, 3.2, e.absDev, testEpsilon)

	// Fourth update (check negative residual logic)
	// prev = 13.2
	// mean = 0.4 * 30 + 0.6 * 13.2 = 12 + 7.92 = 19.92
	// resid = 30 - 13.2 = 16.8
	// absDev = 0.4 * 16.8 + 0.6 * 3.2 = 6.72 + 1.92 = 8.64
	mean, absDev = e.update(30)
	assert.InDelta(t, 19.92, mean, testEpsilon)
	assert.InDelta(t, 8.64, absDev, testEpsilon)
	assert.InDelta(t, 19.92, e.mean, testEpsilon)
	assert.InDelta(t, 8.64, e.absDev, testEpsilon)
}

func TestRobustEWMA_NaN(t *testing.T) {
	e := newRobustEWMA(testAlpha)
	e.update(10)
	e.update(20)

	// Update with NaN
	e.update(math.NaN())

	// Should not propagate NaN
	assert.False(t, math.IsNaN(e.mean))
	assert.False(t, math.IsNaN(e.absDev))

	// Should not propagate Inf
	e.update(math.Inf(1))
	assert.False(t, math.IsInf(e.mean, 1))
	assert.False(t, math.IsInf(e.absDev, 1))
}
