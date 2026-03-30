// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package experr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShutdownErr(t *testing.T) {
	err := NewShutdownErr(errors.New("some error"))
	assert.Equal(t, "interrupted due to shutdown: some error", err.Error())
}

func TestIsShutdownErr(t *testing.T) {
	err := errors.New("testError")
	require.False(t, IsShutdownErr(err))
	err = NewShutdownErr(err)
	require.True(t, IsShutdownErr(err))
}

func TestNewDroppedItemsErr_WithReason(t *testing.T) {
	err := NewDroppedItemsErr(5, "incompatible temporality")
	assert.Equal(t, "dropped 5 items: incompatible temporality", err.Error())
}

func TestNewDroppedItemsErr_NoReason(t *testing.T) {
	err := NewDroppedItemsErr(3, "")
	assert.Equal(t, "dropped 3 items", err.Error())
}

func TestDroppedItemsFromErr(t *testing.T) {
	err := NewDroppedItemsErr(7, "some reason")
	d, ok := DroppedItemsFromErr(err)
	require.True(t, ok)
	assert.Equal(t, 7, d.Dropped)
	assert.Equal(t, "some reason", d.Reason)

	_, ok = DroppedItemsFromErr(errors.New("not a dropped items error"))
	assert.False(t, ok)

	_, ok = DroppedItemsFromErr(nil)
	assert.False(t, ok)
}

