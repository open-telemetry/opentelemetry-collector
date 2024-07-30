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

