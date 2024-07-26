// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package componentstatus

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
			ev := NewEvent(status)
			require.Equal(t, status, ev.Status())
			require.Nil(t, ev.Err())
			require.False(t, ev.Timestamp().IsZero())
		})
	}
}

func TestStatusEventsWithError(t *testing.T) {
	statusConstructorMap := map[Status]func(error) *Event{
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
		statusMap      map[*InstanceID]*Event
		expectedStatus Status
	}{
		{
			name: "aggregate status with fatal is FatalError",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: NewEvent(StatusFatalError),
				{}: NewEvent(StatusRecoverableError),
			},
			expectedStatus: StatusFatalError,
		},
		{
			name: "aggregate status with permanent is PermanentError",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: NewEvent(StatusPermanentError),
				{}: NewEvent(StatusRecoverableError),
			},
			expectedStatus: StatusPermanentError,
		},
		{
			name: "aggregate status with stopping is Stopping",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: NewEvent(StatusRecoverableError),
				{}: NewEvent(StatusStopping),
			},
			expectedStatus: StatusStopping,
		},
		{
			name: "aggregate status with stopped and non-stopped is Stopping",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: NewEvent(StatusRecoverableError),
				{}: NewEvent(StatusStopped),
			},
			expectedStatus: StatusStopping,
		},
		{
			name: "aggregate status with all stopped is Stopped",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStopped),
				{}: NewEvent(StatusStopped),
				{}: NewEvent(StatusStopped),
			},
			expectedStatus: StatusStopped,
		},
		{
			name: "aggregate status with recoverable is RecoverableError",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: NewEvent(StatusRecoverableError),
			},
			expectedStatus: StatusRecoverableError,
		},
		{
			name: "aggregate status with starting is Starting",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
			},
			expectedStatus: StatusStarting,
		},
		{
			name: "aggregate status with all ok is OK",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusOK),
				{}: NewEvent(StatusOK),
				{}: NewEvent(StatusOK),
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
	latest := func(ev *Event) *Event {
		ev.timestamp = maxTime
		return ev
	}

	for _, tc := range []struct {
		name           string
		statusMap      map[*InstanceID]*Event
		expectedStatus *Event
	}{
		{
			name: "FatalError - existing event",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: latest(NewFatalErrorEvent(assert.AnError)),
				{}: NewEvent(StatusRecoverableError),
			},
			expectedStatus: &Event{
				status:    StatusFatalError,
				timestamp: maxTime,
				err:       assert.AnError,
			},
		},
		{
			name: "FatalError - synthetic event",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: NewFatalErrorEvent(assert.AnError),
				{}: latest(NewEvent(StatusRecoverableError)),
			},
			expectedStatus: &Event{
				status:    StatusFatalError,
				timestamp: maxTime,
				err:       assert.AnError,
			},
		},
		{
			name: "PermanentError - existing event",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: latest(NewPermanentErrorEvent(assert.AnError)),
				{}: NewEvent(StatusRecoverableError),
			},
			expectedStatus: &Event{
				status:    StatusPermanentError,
				timestamp: maxTime,
				err:       assert.AnError,
			},
		},
		{
			name: "PermanentError - synthetic event",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: NewPermanentErrorEvent(assert.AnError),
				{}: latest(NewEvent(StatusRecoverableError)),
			},
			expectedStatus: &Event{
				status:    StatusPermanentError,
				timestamp: maxTime,
				err:       assert.AnError,
			},
		},
		{
			name: "Stopping - existing event",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: NewEvent(StatusRecoverableError),
				{}: latest(NewEvent(StatusStopping)),
			},
			expectedStatus: &Event{
				status:    StatusStopping,
				timestamp: maxTime,
			},
		},
		{
			name: "Stopping - synthetic event",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: NewEvent(StatusRecoverableError),
				{}: latest(NewEvent(StatusStopped)),
			},
			expectedStatus: &Event{
				status:    StatusStopping,
				timestamp: maxTime,
			},
		},
		{
			name: "Stopped - existing event",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStopped),
				{}: latest(NewEvent(StatusStopped)),
				{}: NewEvent(StatusStopped),
			},
			expectedStatus: &Event{
				status:    StatusStopped,
				timestamp: maxTime,
			},
		},
		{
			name: "RecoverableError - existing event",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: NewEvent(StatusOK),
				{}: latest(NewRecoverableErrorEvent(assert.AnError)),
			},
			expectedStatus: &Event{
				status:    StatusRecoverableError,
				timestamp: maxTime,
				err:       assert.AnError,
			},
		},
		{
			name: "Starting - synthetic event",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusStarting),
				{}: latest(NewEvent(StatusOK)),
			},
			expectedStatus: &Event{
				status:    StatusStarting,
				timestamp: maxTime,
			},
		},
		{
			name: "OK - existing event",
			statusMap: map[*InstanceID]*Event{
				{}: NewEvent(StatusOK),
				{}: latest(NewEvent(StatusOK)),
				{}: NewEvent(StatusOK),
			},
			expectedStatus: &Event{
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
