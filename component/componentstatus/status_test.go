// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package componentstatus

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
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
			require.NoError(t, ev.Err())
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

func Test_ReportStatus(t *testing.T) {
	t.Run("Reporter implemented", func(t *testing.T) {
		r := &reporter{}
		ReportStatus(r, NewEvent(StatusOK))
		require.True(t, r.reportStatusCalled)
	})

	t.Run("Reporter not implemented", func(t *testing.T) {
		h := &host{}
		ReportStatus(h, NewEvent(StatusOK))
		require.False(t, h.reportStatusCalled)
	})
}

var _ = (component.Host)(nil)
var _ = (Reporter)(nil)

type reporter struct {
	reportStatusCalled bool
}

func (r *reporter) GetExtensions() map[component.ID]component.Component {
	return nil
}

func (r *reporter) Report(_ *Event) {
	r.reportStatusCalled = true
}

var _ = (component.Host)(nil)

type host struct {
	reportStatusCalled bool
}

func (h *host) GetExtensions() map[component.ID]component.Component {
	return nil
}
