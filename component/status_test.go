// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package component

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusEventWithoutError(t *testing.T) {
	statuses := []Status{
		StatusStarting,
		StatusOK,
		StatusRecoverableError,
		StatusPermanentError,
		StatusFatalError,
		StatusStopping,
		StatusStopped,
	}

	for _, status := range statuses {
		t.Run(fmt.Sprintf("%s without error", status), func(t *testing.T) {
			ev, err := NewStatusEvent(status)
			require.NoError(t, err)
			require.Equal(t, status, ev.Status())
			require.Nil(t, ev.Err())
			require.False(t, ev.Timestamp().IsZero())
		})
	}
}

func TestStatusEventWithError(t *testing.T) {
	statuses := []Status{
		StatusRecoverableError,
		StatusRecoverableError,
		StatusFatalError,
	}

	for _, status := range statuses {
		t.Run(fmt.Sprintf("error status: %s with error", status), func(t *testing.T) {
			ev, err := NewStatusEvent(status, WithError(assert.AnError))
			require.NoError(t, err)
			require.Equal(t, status, ev.Status())
			require.Equal(t, assert.AnError, ev.Err())
			require.False(t, ev.Timestamp().IsZero())
		})
	}
}

func TestNonErrorStatusWithError(t *testing.T) {
	statuses := []Status{
		StatusStarting,
		StatusOK,
		StatusStopping,
		StatusStopped,
	}

	for _, status := range statuses {
		t.Run(fmt.Sprintf("non error status: %s with error", status), func(t *testing.T) {
			ev, err := NewStatusEvent(status, WithError(assert.AnError))
			require.Error(t, err)
			require.ErrorIs(t, err, errStatusEventInvalidArgument)
			require.Nil(t, ev)
		})
	}
}

func TestStatusEventWithTimestamp(t *testing.T) {
	ts := time.Now()
	ev, err := NewStatusEvent(StatusOK, WithTimestamp(ts))
	require.NoError(t, err)
	require.Equal(t, StatusOK, ev.Status())
	require.Nil(t, ev.Err())
	require.Equal(t, ts, ev.Timestamp())
}
