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

func TestStatusIsError(t *testing.T) {
	for _, tc := range []struct {
		status  Status
		isError bool
	}{
		{
			status:  StatusStarting,
			isError: false,
		},
		{
			status:  StatusOK,
			isError: false,
		},
		{
			status:  StatusRecoverableError,
			isError: true,
		},
		{
			status:  StatusPermanentError,
			isError: true,
		},
		{
			status:  StatusFatalError,
			isError: true,
		},
		{
			status:  StatusStopping,
			isError: false,
		},
		{
			status:  StatusStopped,
			isError: false,
		},
	} {
		name := fmt.Sprintf("StatusIsError(%s) is %t", tc.status, tc.isError)
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.isError, StatusIsError(tc.status))
		})
	}
}

func TestAggregateStatusEvent(t *testing.T) {
	// maxTime is used to make sure we select the event with the latest timestamp
	maxTime := time.Unix(1<<63-62135596801, 999999999)
	// latest sets the timestamp for an event to maxTime
	latest := func(ev *StatusEvent) *StatusEvent {
		ev.timestamp = maxTime
		return ev
	}

	for _, tc := range []struct {
		name           string
		statusMap      map[*InstanceID]*StatusEvent
		expectedStatus *StatusEvent
	}{
		{
			name: "FatalError - existing event",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: latest(NewFatalErrorEvent(assert.AnError)),
				{}: NewStatusEvent(StatusRecoverableError),
			},
			expectedStatus: &StatusEvent{
				status:    StatusFatalError,
				timestamp: maxTime,
				err:       assert.AnError,
			},
		},
		{
			name: "FatalError - synthetic event",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: NewFatalErrorEvent(assert.AnError),
				{}: latest(NewStatusEvent(StatusRecoverableError)),
			},
			expectedStatus: &StatusEvent{
				status:    StatusFatalError,
				timestamp: maxTime,
				err:       assert.AnError,
			},
		},
		{
			name: "PermanentError - existing event",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: latest(NewPermanentErrorEvent(assert.AnError)),
				{}: NewStatusEvent(StatusRecoverableError),
			},
			expectedStatus: &StatusEvent{
				status:    StatusPermanentError,
				timestamp: maxTime,
				err:       assert.AnError,
			},
		},
		{
			name: "PermanentError - synthetic event",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: NewPermanentErrorEvent(assert.AnError),
				{}: latest(NewStatusEvent(StatusRecoverableError)),
			},
			expectedStatus: &StatusEvent{
				status:    StatusPermanentError,
				timestamp: maxTime,
				err:       assert.AnError,
			},
		},
		{
			name: "Stopping - existing event",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: NewStatusEvent(StatusRecoverableError),
				{}: latest(NewStatusEvent(StatusStopping)),
			},
			expectedStatus: &StatusEvent{
				status:    StatusStopping,
				timestamp: maxTime,
			},
		},
		{
			name: "Stopping - synthetic event",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: NewStatusEvent(StatusRecoverableError),
				{}: latest(NewStatusEvent(StatusStopped)),
			},
			expectedStatus: &StatusEvent{
				status:    StatusStopping,
				timestamp: maxTime,
			},
		},
		{
			name: "Stopped - existing event",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStopped),
				{}: latest(NewStatusEvent(StatusStopped)),
				{}: NewStatusEvent(StatusStopped),
			},
			expectedStatus: &StatusEvent{
				status:    StatusStopped,
				timestamp: maxTime,
			},
		},
		{
			name: "RecoverableError - existing event",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: NewStatusEvent(StatusOK),
				{}: latest(NewRecoverableErrorEvent(assert.AnError)),
			},
			expectedStatus: &StatusEvent{
				status:    StatusRecoverableError,
				timestamp: maxTime,
				err:       assert.AnError,
			},
		},
		{
			name: "Starting - synthetic event",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusStarting),
				{}: latest(NewStatusEvent(StatusOK)),
			},
			expectedStatus: &StatusEvent{
				status:    StatusStarting,
				timestamp: maxTime,
			},
		},
		{
			name: "OK - existing event",
			statusMap: map[*InstanceID]*StatusEvent{
				{}: NewStatusEvent(StatusOK),
				{}: latest(NewStatusEvent(StatusOK)),
				{}: NewStatusEvent(StatusOK),
			},
			expectedStatus: &StatusEvent{
				status:    StatusOK,
				timestamp: maxTime,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedStatus, AggregateStatusEvent(tc.statusMap))
		})
	}
}
