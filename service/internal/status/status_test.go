// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestStatusFSM(t *testing.T) {
	for _, tc := range []struct {
		name               string
		reportedStatuses   []component.Status
		expectedStatuses   []component.Status
		expectedErrorCount int
	}{
		{
			name: "successful startup and shutdown",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
		},
		{
			name: "component recovered",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusRecoverableError,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusRecoverableError,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
		},
		{
			name: "repeated events are errors",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusRecoverableError,
				component.StatusRecoverableError,
				component.StatusRecoverableError,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusRecoverableError,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
			expectedErrorCount: 2,
		},
		{
			name: "PermanentError is terminal",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusPermanentError,
				component.StatusOK,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusPermanentError,
			},
			expectedErrorCount: 1,
		},
		{
			name: "FatalError is terminal",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusFatalError,
				component.StatusOK,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusFatalError,
			},
			expectedErrorCount: 1,
		},
		{
			name: "Stopped is terminal",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
				component.StatusOK,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
			expectedErrorCount: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var receivedStatuses []component.Status
			fsm := newFSM(
				func(ev *component.StatusEvent) {
					receivedStatuses = append(receivedStatuses, ev.Status())
				},
			)

			errorCount := 0
			for _, status := range tc.reportedStatuses {
				if err := fsm.transition(component.NewStatusEvent(status)); err != nil {
					errorCount++
					require.ErrorIs(t, err, errInvalidStateTransition)
				}
			}

			require.Equal(t, tc.expectedErrorCount, errorCount)
			require.Equal(t, tc.expectedStatuses, receivedStatuses)
		})
	}
}

func TestValidSeqsToStopped(t *testing.T) {
	events := []*component.StatusEvent{
		component.NewStatusEvent(component.StatusStarting),
		component.NewStatusEvent(component.StatusOK),
		component.NewStatusEvent(component.StatusRecoverableError),
		component.NewStatusEvent(component.StatusPermanentError),
		component.NewStatusEvent(component.StatusFatalError),
	}

	for _, ev := range events {
		name := fmt.Sprintf("transition from: %s to: %s invalid", ev.Status(), component.StatusStopped)
		t.Run(name, func(t *testing.T) {
			fsm := newFSM(func(*component.StatusEvent) {})
			if ev.Status() != component.StatusStarting {
				require.NoError(t, fsm.transition(component.NewStatusEvent(component.StatusStarting)))
			}
			require.NoError(t, fsm.transition(ev))
			// skipping to stopped is not allowed
			err := fsm.transition(component.NewStatusEvent(component.StatusStopped))
			require.ErrorIs(t, err, errInvalidStateTransition)

			// stopping -> stopped is allowed for non-fatal, non-permanent errors
			err = fsm.transition(component.NewStatusEvent(component.StatusStopping))
			if ev.Status() == component.StatusPermanentError || ev.Status() == component.StatusFatalError {
				require.ErrorIs(t, err, errInvalidStateTransition)
			} else {
				require.NoError(t, err)
				require.NoError(t, fsm.transition(component.NewStatusEvent(component.StatusStopped)))
			}
		})
	}

}

func TestStatusFuncs(t *testing.T) {
	id1 := &component.InstanceID{}
	id2 := &component.InstanceID{}

	actualStatuses := make(map[*component.InstanceID][]component.Status)
	statusFunc := func(id *component.InstanceID, ev *component.StatusEvent) {
		actualStatuses[id] = append(actualStatuses[id], ev.Status())
	}

	statuses1 := []component.Status{
		component.StatusStarting,
		component.StatusOK,
		component.StatusStopping,
		component.StatusStopped,
	}

	statuses2 := []component.Status{
		component.StatusStarting,
		component.StatusOK,
		component.StatusRecoverableError,
		component.StatusOK,
		component.StatusStopping,
		component.StatusStopped,
	}

	expectedStatuses := map[*component.InstanceID][]component.Status{
		id1: statuses1,
		id2: statuses2,
	}

	rep := NewReporter(statusFunc,
		func(err error) {
			require.NoError(t, err)
		})
	comp1Func := NewReportStatusFunc(id1, rep.ReportStatus)
	comp2Func := NewReportStatusFunc(id2, rep.ReportStatus)
	rep.Ready()

	for _, st := range statuses1 {
		comp1Func(component.NewStatusEvent(st))
	}

	for _, st := range statuses2 {
		comp2Func(component.NewStatusEvent(st))
	}

	require.Equal(t, expectedStatuses, actualStatuses)
}

func TestStatusFuncsConcurrent(t *testing.T) {
	ids := []*component.InstanceID{{}, {}, {}, {}}
	count := 0
	statusFunc := func(*component.InstanceID, *component.StatusEvent) {
		count++
	}
	rep := NewReporter(statusFunc,
		func(err error) {
			require.NoError(t, err)
		})
	rep.Ready()

	wg := sync.WaitGroup{}
	wg.Add(len(ids))

	for _, id := range ids {
		id := id
		go func() {
			compFn := NewReportStatusFunc(id, rep.ReportStatus)
			compFn(component.NewStatusEvent(component.StatusStarting))
			for i := 0; i < 1000; i++ {
				compFn(component.NewStatusEvent(component.StatusRecoverableError))
				compFn(component.NewStatusEvent(component.StatusOK))
			}
			wg.Done()
		}()
	}

	wg.Wait()
	require.Equal(t, 8004, count)
}

func TestReporterReady(t *testing.T) {
	statusFunc := func(*component.InstanceID, *component.StatusEvent) {}
	var err error
	rep := NewReporter(statusFunc,
		func(e error) {
			err = e
		})
	id := &component.InstanceID{}

	rep.ReportStatus(id, component.NewStatusEvent(component.StatusStarting))
	require.ErrorIs(t, err, ErrStatusNotReady)
	rep.Ready()

	err = nil
	rep.ReportStatus(id, component.NewStatusEvent(component.StatusStarting))
	require.NoError(t, err)
}

func TestReportComponentOKIfStarting(t *testing.T) {
	for _, tc := range []struct {
		name             string
		initialStatuses  []component.Status
		expectedStatuses []component.Status
	}{
		{
			name: "matching condition: StatusStarting",
			initialStatuses: []component.Status{
				component.StatusStarting,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
			},
		},
		{
			name: "non-matching condition StatusOK",
			initialStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
			},
		},
		{
			name: "non-matching condition RecoverableError",
			initialStatuses: []component.Status{
				component.StatusStarting,
				component.StatusRecoverableError,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusRecoverableError,
			},
		},
		{
			name: "non-matching condition PermanentError",
			initialStatuses: []component.Status{
				component.StatusStarting,
				component.StatusPermanentError,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusPermanentError,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var receivedStatuses []component.Status

			rep := NewReporter(
				func(_ *component.InstanceID, ev *component.StatusEvent) {
					receivedStatuses = append(receivedStatuses, ev.Status())
				},
				func(err error) {
					require.NoError(t, err)
				},
			)
			rep.Ready()

			id := &component.InstanceID{}
			for _, status := range tc.initialStatuses {
				rep.ReportStatus(id, component.NewStatusEvent(status))
			}

			rep.ReportOKIfStarting(id)

			require.Equal(t, tc.expectedStatuses, receivedStatuses)
		})
	}
}
