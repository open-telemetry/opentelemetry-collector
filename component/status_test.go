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
