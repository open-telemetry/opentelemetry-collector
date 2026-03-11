// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	var count atomic.Int64
	statusFunc := func(*componentstatus.InstanceID, *componentstatus.Event) {
		count.Add(1)
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
			wg.Done()
		}()
	}

	wg.Wait()
	require.Equal(t, int64(8004), count.Load())
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

func TestReporterDeadlockOnAsyncFatalError(t *testing.T) {
	blockCh := make(chan error)

	id1 := &componentstatus.InstanceID{} // Component A: starts its server asynchronously
	id2 := &componentstatus.InstanceID{} // Component B: next component started by StartAll

	onStatusChangeCalled := make(chan struct{})

	rep := NewReporter(
		func(_ *componentstatus.InstanceID, ev *componentstatus.Event) {
			if ev.Status() == componentstatus.StatusFatalError {
				close(onStatusChangeCalled)
				// replicating deadlock condition.
				blockCh <- ev.Err()
			}
		},
		func(error) {},
	)

	var wg sync.WaitGroup
	t.Cleanup(func() {
		go func() { <-blockCh }()
		wg.Wait()
	})

	// StartAll: Component A begins starting.
	rep.ReportStatus(id1, componentstatus.NewEvent(componentstatus.StatusStarting))

	wg.Go(func() {
		rep.ReportStatus(id1, componentstatus.NewFatalErrorEvent(errors.New("port already in use")))
	})

	select {
	case <-onStatusChangeCalled:
		// The FatalError goroutine now holds mutex lock and is blocked on blockCh.
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for FatalError reporter to enter onStatusChange")
	}

	startAllDone := make(chan struct{})
	wg.Go(func() {
		rep.ReportStatus(id2, componentstatus.NewEvent(componentstatus.StatusStarting))
		close(startAllDone)
	})

	select {
	case <-startAllDone:
		// ReportStatus completed successfully.
	case <-time.After(time.Second):
		t.Fatal("deadlock: ReportStatus for second component blocked because onStatusChange held r.mu")
	}
}
