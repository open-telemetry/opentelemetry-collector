// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package status // import "go.opentelemetry.io/collector/service/internal/status"

import (
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component/componentstatus"
)

// onTransitionFunc receives a componentstatus.Event on a successful state transition
type onTransitionFunc func(*componentstatus.Event)

// errInvalidStateTransition is returned for invalid state transitions
var errInvalidStateTransition = errors.New("invalid state transition")

// fsm is a finite state machine that models transitions for component status
type fsm struct {
	current      *componentstatus.Event
	transitions  map[componentstatus.Status]map[componentstatus.Status]struct{}
	onTransition onTransitionFunc
}

// transition will attempt to execute a state transition. If it's successful, it calls the
// onTransitionFunc with a Event representing the new state. Returns an error if the arguments
// result in an invalid status, or if the state transition is not valid.
func (m *fsm) transition(ev *componentstatus.Event) error {
	if _, ok := m.transitions[m.current.Status()][ev.Status()]; !ok {
		return fmt.Errorf(
			"cannot transition from %s to %s: %w",
			m.current.Status(),
			ev.Status(),
			errInvalidStateTransition,
		)
	}
	m.current = ev
	m.onTransition(ev)
	return nil
}

// newFSM creates a state machine with all valid transitions for componentstatus.Status.
// The initial state is set to componentstatus.StatusNone.
func newFSM(onTransition onTransitionFunc) *fsm {
	return &fsm{
		current:      componentstatus.NewEvent(componentstatus.StatusNone),
		onTransition: onTransition,
		transitions: map[componentstatus.Status]map[componentstatus.Status]struct{}{
			componentstatus.StatusNone: {
				componentstatus.StatusStarting: {},
			},
			componentstatus.StatusStarting: {
				componentstatus.StatusOK:               {},
				componentstatus.StatusRecoverableError: {},
				componentstatus.StatusPermanentError:   {},
				componentstatus.StatusFatalError:       {},
				componentstatus.StatusStopping:         {},
			},
			componentstatus.StatusOK: {
				componentstatus.StatusRecoverableError: {},
				componentstatus.StatusPermanentError:   {},
				componentstatus.StatusFatalError:       {},
				componentstatus.StatusStopping:         {},
			},
			componentstatus.StatusRecoverableError: {
				componentstatus.StatusOK:             {},
				componentstatus.StatusPermanentError: {},
				componentstatus.StatusFatalError:     {},
				componentstatus.StatusStopping:       {},
			},
			componentstatus.StatusPermanentError: {
				componentstatus.StatusStopping: {},
			},
			componentstatus.StatusFatalError: {},
			componentstatus.StatusStopping: {
				componentstatus.StatusRecoverableError: {},
				componentstatus.StatusPermanentError:   {},
				componentstatus.StatusFatalError:       {},
				componentstatus.StatusStopped:          {},
			},
			componentstatus.StatusStopped: {},
		},
	}
}

// NotifyStatusFunc is the receiver of status events after successful state transitions
type NotifyStatusFunc func(*componentstatus.InstanceID, *componentstatus.Event)

// InvalidTransitionFunc is the receiver of invalid transition errors
type InvalidTransitionFunc func(error)

// ServiceStatusFunc is the expected type of ReportStatus
type ServiceStatusFunc func(*componentstatus.InstanceID, *componentstatus.Event)

// ErrStatusNotReady is returned when trying to report status before service start
var ErrStatusNotReady = errors.New("report component status is not ready until service start")

// Reporter handles component status reporting
type Reporter interface {
	ReportStatus(id *componentstatus.InstanceID, ev *componentstatus.Event)
	ReportOKIfStarting(id *componentstatus.InstanceID)
}

type reporter struct {
	mu                  sync.Mutex
	fsmMap              map[*componentstatus.InstanceID]*fsm
	onStatusChange      NotifyStatusFunc
	onInvalidTransition InvalidTransitionFunc
}

// NewReporter returns a reporter that will invoke the NotifyStatusFunc when a component's status
// has changed.
func NewReporter(onStatusChange NotifyStatusFunc, onInvalidTransition InvalidTransitionFunc) Reporter {
	return &reporter{
		fsmMap:              make(map[*componentstatus.InstanceID]*fsm),
		onStatusChange:      onStatusChange,
		onInvalidTransition: onInvalidTransition,
	}
}

// ReportStatus reports status for the given InstanceID
func (r *reporter) ReportStatus(
	id *componentstatus.InstanceID,
	ev *componentstatus.Event,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.componentFSM(id).transition(ev); err != nil {
		r.onInvalidTransition(err)
	}
}

func (r *reporter) ReportOKIfStarting(id *componentstatus.InstanceID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fsm := r.componentFSM(id)
	if fsm.current.Status() == componentstatus.StatusStarting {
		if err := fsm.transition(componentstatus.NewEvent(componentstatus.StatusOK)); err != nil {
			r.onInvalidTransition(err)
		}
	}
}

// Note: a lock must be acquired before calling this method.
func (r *reporter) componentFSM(id *componentstatus.InstanceID) *fsm {
	fsm, ok := r.fsmMap[id]
	if !ok {
		fsm = newFSM(func(ev *componentstatus.Event) { r.onStatusChange(id, ev) })
		r.fsmMap[id] = fsm
	}
	return fsm
}

// NewReportStatusFunc returns a function to be used as ReportStatus for componentstatus.TelemetrySettings
func NewReportStatusFunc(
	id *componentstatus.InstanceID,
	srvStatus ServiceStatusFunc,
) func(*componentstatus.Event) {
	return func(ev *componentstatus.Event) {
		srvStatus(id, ev)
	}
}
