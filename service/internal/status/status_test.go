// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
)

func TestStatusFSM(t *testing.T) {
	for _, tc := range []struct {
		name               string
		reportedStatuses   []componentstatus.Status
		expectedStatuses   []componentstatus.Status
		expectedErrorCount int
	}{
		{
			name: "successful startup and shutdown",
			reportedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusStopped,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusStopped,
			},
		},
		{
			name: "component recovered",
			reportedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusRecoverableError,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusStopped,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusRecoverableError,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusStopped,
			},
		},
		{
			name: "repeated events are errors",
			reportedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusRecoverableError,
				componentstatus.StatusRecoverableError,
				componentstatus.StatusRecoverableError,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusStopped,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusRecoverableError,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusStopped,
			},
			expectedErrorCount: 2,
		},
		{
			name: "PermanentError is terminal",
			reportedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusPermanentError,
				componentstatus.StatusOK,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusPermanentError,
			},
			expectedErrorCount: 1,
		},
		{
			name: "FatalError is terminal",
			reportedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusFatalError,
				componentstatus.StatusOK,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusFatalError,
			},
			expectedErrorCount: 1,
		},
		{
			name: "Stopped is terminal",
			reportedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusStopped,
				componentstatus.StatusOK,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusStopped,
			},
			expectedErrorCount: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var receivedStatuses []componentstatus.Status
			fsm := newFSM(
				func(ev *componentstatus.StatusEvent) {
					receivedStatuses = append(receivedStatuses, ev.Status())
				},
			)

			errorCount := 0
			for _, status := range tc.reportedStatuses {
				if err := fsm.transition(componentstatus.NewStatusEvent(status)); err != nil {
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
	events := []*componentstatus.StatusEvent{
		componentstatus.NewStatusEvent(componentstatus.StatusStarting),
		componentstatus.NewStatusEvent(componentstatus.StatusOK),
		componentstatus.NewStatusEvent(componentstatus.StatusRecoverableError),
		componentstatus.NewStatusEvent(componentstatus.StatusPermanentError),
		componentstatus.NewStatusEvent(componentstatus.StatusFatalError),
	}

	for _, ev := range events {
		name := fmt.Sprintf("transition from: %s to: %s invalid", ev.Status(), componentstatus.StatusStopped)
		t.Run(name, func(t *testing.T) {
			fsm := newFSM(func(*componentstatus.StatusEvent) {})
			if ev.Status() != componentstatus.StatusStarting {
				require.NoError(t, fsm.transition(componentstatus.NewStatusEvent(componentstatus.StatusStarting)))
			}
			require.NoError(t, fsm.transition(ev))
			// skipping to stopped is not allowed
			err := fsm.transition(componentstatus.NewStatusEvent(componentstatus.StatusStopped))
			require.ErrorIs(t, err, errInvalidStateTransition)

			// stopping -> stopped is allowed for non-fatal, non-permanent errors
			err = fsm.transition(componentstatus.NewStatusEvent(componentstatus.StatusStopping))
			if ev.Status() == componentstatus.StatusPermanentError || ev.Status() == componentstatus.StatusFatalError {
				require.ErrorIs(t, err, errInvalidStateTransition)
			} else {
				require.NoError(t, err)
				require.NoError(t, fsm.transition(componentstatus.NewStatusEvent(componentstatus.StatusStopped)))
			}
		})
	}

}

func TestStatusFuncs(t *testing.T) {
	id1 := &component.InstanceID{}
	id2 := &component.InstanceID{}

	actualStatuses := make(map[*component.InstanceID][]componentstatus.Status)
	statusFunc := func(id *component.InstanceID, ev *componentstatus.StatusEvent) {
		actualStatuses[id] = append(actualStatuses[id], ev.Status())
	}

	statuses1 := []componentstatus.Status{
		componentstatus.StatusStarting,
		componentstatus.StatusOK,
		componentstatus.StatusStopping,
		componentstatus.StatusStopped,
	}

	statuses2 := []componentstatus.Status{
		componentstatus.StatusStarting,
		componentstatus.StatusOK,
		componentstatus.StatusRecoverableError,
		componentstatus.StatusOK,
		componentstatus.StatusStopping,
		componentstatus.StatusStopped,
	}

	expectedStatuses := map[*component.InstanceID][]componentstatus.Status{
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
		comp1Func(componentstatus.NewStatusEvent(st))
	}

	for _, st := range statuses2 {
		comp2Func(componentstatus.NewStatusEvent(st))
	}

	require.Equal(t, expectedStatuses, actualStatuses)
}

func TestStatusFuncsConcurrent(t *testing.T) {
	ids := []*component.InstanceID{{}, {}, {}, {}}
	count := 0
	statusFunc := func(*component.InstanceID, *componentstatus.StatusEvent) {
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
			compFn(componentstatus.NewStatusEvent(componentstatus.StatusStarting))
			for i := 0; i < 1000; i++ {
				compFn(componentstatus.NewStatusEvent(componentstatus.StatusRecoverableError))
				compFn(componentstatus.NewStatusEvent(componentstatus.StatusOK))
			}
			wg.Done()
		}()
	}

	wg.Wait()
	require.Equal(t, 8004, count)
}

func TestReporterReady(t *testing.T) {
	statusFunc := func(*component.InstanceID, *componentstatus.StatusEvent) {}
	var err error
	rep := NewReporter(statusFunc,
		func(e error) {
			err = e
		})
	id := &component.InstanceID{}

	rep.ReportStatus(id, componentstatus.NewStatusEvent(componentstatus.StatusStarting))
	require.ErrorIs(t, err, ErrStatusNotReady)
	rep.Ready()

	err = nil
	rep.ReportStatus(id, componentstatus.NewStatusEvent(componentstatus.StatusStarting))
	require.NoError(t, err)
}

func TestReportComponentOKIfStarting(t *testing.T) {
	for _, tc := range []struct {
		name             string
		initialStatuses  []componentstatus.Status
		expectedStatuses []componentstatus.Status
	}{
		{
			name: "matching condition: StatusStarting",
			initialStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
			},
		},
		{
			name: "non-matching condition StatusOK",
			initialStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
			},
		},
		{
			name: "non-matching condition RecoverableError",
			initialStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusRecoverableError,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusRecoverableError,
			},
		},
		{
			name: "non-matching condition PermanentError",
			initialStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusPermanentError,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusPermanentError,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var receivedStatuses []componentstatus.Status

			rep := NewReporter(
				func(_ *component.InstanceID, ev *componentstatus.StatusEvent) {
					receivedStatuses = append(receivedStatuses, ev.Status())
				},
				func(err error) {
					require.NoError(t, err)
				},
			)
			rep.Ready()

			id := &component.InstanceID{}
			for _, status := range tc.initialStatuses {
				rep.ReportStatus(id, componentstatus.NewStatusEvent(status))
			}

			rep.ReportOKIfStarting(id)

			require.Equal(t, tc.expectedStatuses, receivedStatuses)
		})
	}
}
