// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componentstatus"
)

func TestStatusFSM(t *testing.T) {
	for _, tt := range []struct {
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
			name: "repeated OK and RecoverableError events are valid",
			reportedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusOK,
				componentstatus.StatusRecoverableError,
				componentstatus.StatusRecoverableError,
				componentstatus.StatusOK,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusStopped,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusOK,
				componentstatus.StatusRecoverableError,
				componentstatus.StatusRecoverableError,
				componentstatus.StatusOK,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
				componentstatus.StatusStopped,
			},
		},
		{
			name: "PermanentError is stoppable",
			reportedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusPermanentError,
				componentstatus.StatusOK,
				componentstatus.StatusStopping,
			},
			expectedStatuses: []componentstatus.Status{
				componentstatus.StatusStarting,
				componentstatus.StatusOK,
				componentstatus.StatusPermanentError,
				componentstatus.StatusStopping,
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
		t.Run(tt.name, func(t *testing.T) {
			var receivedStatuses []componentstatus.Status
			fsm := newFSM(
				func(ev *componentstatus.Event) {
					receivedStatuses = append(receivedStatuses, ev.Status())
				},
			)

			errorCount := 0
			for _, status := range tt.reportedStatuses {
				if err := fsm.transition(componentstatus.NewEvent(status)); err != nil {
					errorCount++
					require.ErrorIs(t, err, errInvalidStateTransition)
				}
			}

			require.Equal(t, tt.expectedErrorCount, errorCount)
			require.Equal(t, tt.expectedStatuses, receivedStatuses)
		})
	}
}

func TestValidSeqsToStopped(t *testing.T) {
	events := []*componentstatus.Event{
		componentstatus.NewEvent(componentstatus.StatusStarting),
		componentstatus.NewEvent(componentstatus.StatusOK),
		componentstatus.NewEvent(componentstatus.StatusRecoverableError),
		componentstatus.NewEvent(componentstatus.StatusPermanentError),
		componentstatus.NewEvent(componentstatus.StatusFatalError),
	}

	for _, ev := range events {
		name := fmt.Sprintf("transition from: %s to: %s", ev.Status(), componentstatus.StatusStopped)
		t.Run(name, func(t *testing.T) {
			fsm := newFSM(func(*componentstatus.Event) {})
			if ev.Status() != componentstatus.StatusStarting {
				require.NoError(t, fsm.transition(componentstatus.NewEvent(componentstatus.StatusStarting)))
			}
			require.NoError(t, fsm.transition(ev))
			// skipping to stopped is not allowed
			err := fsm.transition(componentstatus.NewEvent(componentstatus.StatusStopped))
			require.ErrorIs(t, err, errInvalidStateTransition)

			// stopping -> stopped is allowed for non-fatal errors
			err = fsm.transition(componentstatus.NewEvent(componentstatus.StatusStopping))
			if ev.Status() == componentstatus.StatusFatalError {
				require.ErrorIs(t, err, errInvalidStateTransition)
			} else {
				require.NoError(t, err)
				require.NoError(t, fsm.transition(componentstatus.NewEvent(componentstatus.StatusStopped)))
			}
		})
	}
}

func TestStatusFuncs(t *testing.T) {
	id1 := &componentstatus.InstanceID{}
	id2 := &componentstatus.InstanceID{}

	actualStatuses := make(map[*componentstatus.InstanceID][]componentstatus.Status)
	statusFunc := func(id *componentstatus.InstanceID, ev *componentstatus.Event) {
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

	expectedStatuses := map[*componentstatus.InstanceID][]componentstatus.Status{
		id1: statuses1,
		id2: statuses2,
	}

	rep := NewReporter(statusFunc,
		func(err error) {
			require.NoError(t, err)
		})
	comp1Func := NewReportStatusFunc(id1, rep.ReportStatus)
	comp2Func := NewReportStatusFunc(id2, rep.ReportStatus)

	for _, st := range statuses1 {
		comp1Func(componentstatus.NewEvent(st))
	}

	for _, st := range statuses2 {
		comp2Func(componentstatus.NewEvent(st))
	}

	require.Equal(t, expectedStatuses, actualStatuses)
}

func TestStatusFuncsConcurrent(t *testing.T) {
	ids := []*componentstatus.InstanceID{{}, {}, {}, {}}
	count := 0
	statusFunc := func(*componentstatus.InstanceID, *componentstatus.Event) {
		count++
	}
	rep := NewReporter(statusFunc,
		func(err error) {
			require.NoError(t, err)
		})

	wg := sync.WaitGroup{}
	wg.Add(len(ids))

	for _, id := range ids {
		go func() {
			compFn := NewReportStatusFunc(id, rep.ReportStatus)
			compFn(componentstatus.NewEvent(componentstatus.StatusStarting))
			for range 1000 {
				compFn(componentstatus.NewEvent(componentstatus.StatusRecoverableError))
				compFn(componentstatus.NewEvent(componentstatus.StatusOK))
			}
			compFn(componentstatus.NewEvent(componentstatus.StatusStopping))
			compFn(componentstatus.NewEvent(componentstatus.StatusStopped))
			wg.Done()
		}()
	}

	wg.Wait()
	require.Equal(t, 8012, count)
	assert.Empty(t, rep.(*reporter).fsmMap, "stopped instances must not be retained")
}

func TestReportComponentOKIfStarting(t *testing.T) {
	for _, tt := range []struct {
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
		t.Run(tt.name, func(t *testing.T) {
			var receivedStatuses []componentstatus.Status

			rep := NewReporter(
				func(_ *componentstatus.InstanceID, ev *componentstatus.Event) {
					receivedStatuses = append(receivedStatuses, ev.Status())
				},
				func(err error) {
					require.NoError(t, err)
				},
			)

			id := &componentstatus.InstanceID{}
			for _, status := range tt.initialStatuses {
				rep.ReportStatus(id, componentstatus.NewEvent(status))
			}

			rep.ReportOKIfStarting(id)

			require.Equal(t, tt.expectedStatuses, receivedStatuses)
		})
	}
}

// TestReporterDropsStoppedInstances verifies that the reporter forgets an
// instance once it reports StatusStopped. Partial reloads replace component
// instances while the service keeps running, so a reporter that keeps every
// instance it has ever seen grows with each reload. A late event for a
// dropped instance is reported as an invalid transition and must not bring
// the entry back.
func TestReporterDropsStoppedInstances(t *testing.T) {
	var invalid []error
	rep := NewReporter(
		func(*componentstatus.InstanceID, *componentstatus.Event) {},
		func(err error) { invalid = append(invalid, err) },
	)
	r := rep.(*reporter)

	ids := []*componentstatus.InstanceID{{}, {}}
	for _, id := range ids {
		for _, st := range []componentstatus.Status{
			componentstatus.StatusStarting,
			componentstatus.StatusOK,
			componentstatus.StatusStopping,
		} {
			rep.ReportStatus(id, componentstatus.NewEvent(st))
		}
	}
	require.Len(t, r.fsmMap, 2)

	for _, id := range ids {
		rep.ReportStatus(id, componentstatus.NewEvent(componentstatus.StatusStopped))
	}
	assert.Empty(t, r.fsmMap)
	assert.Empty(t, invalid)

	rep.ReportStatus(ids[0], componentstatus.NewEvent(componentstatus.StatusOK))
	rep.ReportStatus(ids[0], componentstatus.NewRecoverableErrorEvent(errors.New("late")))
	assert.Empty(t, r.fsmMap)
	require.Len(t, invalid, 2)
	for _, err := range invalid {
		require.ErrorIs(t, err, errInvalidStateTransition)
	}
}

// TestReporterUnknownInstanceFirstEvent verifies that an event other than
// StatusStarting for an instance the reporter has never seen is an invalid
// transition that leaves no entry behind. A component shared between
// pipelines replays its status events to every pipeline it was started
// from, including one whose node has already been stopped and dropped by a
// partial reload; such a replay must not resurrect the dropped entry.
func TestReporterUnknownInstanceFirstEvent(t *testing.T) {
	var notified []componentstatus.Status
	var invalid []error
	rep := NewReporter(
		func(_ *componentstatus.InstanceID, ev *componentstatus.Event) {
			notified = append(notified, ev.Status())
		},
		func(err error) { invalid = append(invalid, err) },
	)
	r := rep.(*reporter)

	id := &componentstatus.InstanceID{}
	for _, st := range []componentstatus.Status{
		componentstatus.StatusOK,
		componentstatus.StatusRecoverableError,
		componentstatus.StatusStopping,
		componentstatus.StatusStopped,
	} {
		rep.ReportStatus(id, componentstatus.NewEvent(st))
	}
	assert.Empty(t, r.fsmMap)
	assert.Empty(t, notified)
	require.Len(t, invalid, 4)
	for _, err := range invalid {
		require.ErrorIs(t, err, errInvalidStateTransition)
	}

	// StatusStarting is the only valid first event and is what creates the entry.
	rep.ReportStatus(id, componentstatus.NewEvent(componentstatus.StatusStarting))
	assert.Len(t, r.fsmMap, 1)
	assert.Equal(t, []componentstatus.Status{componentstatus.StatusStarting}, notified)
}

func TestReportOKIfStartingUnknownInstance(t *testing.T) {
	var notified []componentstatus.Status
	rep := NewReporter(
		func(_ *componentstatus.InstanceID, ev *componentstatus.Event) {
			notified = append(notified, ev.Status())
		},
		func(err error) { require.NoError(t, err) },
	)

	rep.ReportOKIfStarting(&componentstatus.InstanceID{})
	assert.Empty(t, notified)
	assert.Empty(t, rep.(*reporter).fsmMap, "an unknown instance must not be registered")
}
