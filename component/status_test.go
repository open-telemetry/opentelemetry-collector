// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package component

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatusEvent(t *testing.T) {
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
			ev := NewStatusEvent(status)
			require.Equal(t, status, ev.Status())
			require.Nil(t, ev.Err())
			require.False(t, ev.Timestamp().IsZero())
		})
	}
}

func TestStatusEventsWithError(t *testing.T) {
	statusConstructorMap := map[Status]func(error) *StatusEvent{
		StatusRecoverableError: NewRecoverableErrorEvent,
		StatusPermanentError:   NewPermanentErrorEvent,
		StatusFatalError:       NewFatalErrorEvent,
	}

	for status, newEvent := range statusConstructorMap {
		t.Run(fmt.Sprintf("error status constructor for: %s", status), func(t *testing.T) {
			ev := newEvent(assert.AnError)
			require.Equal(t, status, ev.Status())
			require.Equal(t, assert.AnError, ev.Err())
			require.False(t, ev.Timestamp().IsZero())
		})
	}
}

func TestAggregateStatus(t *testing.T) {
	for _, tc := range []struct {
		name           string
		statusMap      map[*InstanceID]*StatusEvent
		expectedStatus Status
	}{
		{
			name: "aggregate status with fatal is FatalError",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: NewStatusEvent(StatusFatalError),
				{}: NewStatusEvent(StatusRecoverableError),
			},
			expectedStatus: StatusFatalError,
		},
		{
			name: "aggregate status with permanent is PermanentError",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: NewStatusEvent(StatusPermanentError),
				{}: NewStatusEvent(StatusRecoverableError),
			},
			expectedStatus: StatusPermanentError,
		},
		{
			name: "aggregate status with stopping is Stopping",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: NewStatusEvent(StatusRecoverableError),
				{}: NewStatusEvent(StatusStopping),
			},
			expectedStatus: StatusStopping,
		},
		{
			name: "aggregate status with stopped and non-stopped is Stopping",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: NewStatusEvent(StatusRecoverableError),
				{}: NewStatusEvent(StatusStopped),
			},
			expectedStatus: StatusStopping,
		},
		{
			name: "aggregate status with all stopped is Stopped",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStopped),
				{}: NewStatusEvent(StatusStopped),
				{}: NewStatusEvent(StatusStopped),
			},
			expectedStatus: StatusStopped,
		},
		{
			name: "aggregate status with recoverable is RecoverableError",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: NewStatusEvent(StatusRecoverableError),
			},
			expectedStatus: StatusRecoverableError,
		},
		{
			name: "aggregate status with starting is Starting",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
			},
			expectedStatus: StatusStarting,
		},
		{
			name: "aggregate status with all ok is OK",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusOK),
				{}: NewStatusEvent(StatusOK),
				{}: NewStatusEvent(StatusOK),
			},
			expectedStatus: StatusOK,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedStatus, AggregateStatus(tc.statusMap))
		})
	}
}
