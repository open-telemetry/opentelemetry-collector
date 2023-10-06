// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package status // import "go.opentelemetry.io/collector/service/internal/status"

import (
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
)

// onTransitionFunc receives a component.StatusEvent on a successful state transition
type onTransitionFunc func(*component.StatusEvent)

// errInvalidStateTransition is returned for invalid state transitions
var errInvalidStateTransition = errors.New("invalid state transition")

// fsm is a finite state machine that models transitions for component status
type fsm struct {
	current      *component.StatusEvent
	transitions  map[component.Status]map[component.Status]struct{}
	onTransition onTransitionFunc
}

// transition will attempt to execute a state transition. If it's successful, it calls the
// onTransitionFunc with a StatusEvent representing the new state. Returns an error if the arguments
// result in an invalid status, or if the state transition is not valid.
func (m *fsm) transition(ev *component.StatusEvent) error {
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

// newFSM creates a state machine with all valid transitions for component.Status.
// The initial state is set to component.StatusNone.
func newFSM(onTransition onTransitionFunc) *fsm {
	return &fsm{
		current:      component.NewStatusEvent(component.StatusNone),
		onTransition: onTransition,
		transitions: map[component.Status]map[component.Status]struct{}{
			component.StatusNone: {
				component.StatusStarting: {},
			},
			component.StatusStarting: {
				component.StatusOK:               {},
				component.StatusRecoverableError: {},
				component.StatusPermanentError:   {},
				component.StatusFatalError:       {},
				component.StatusStopping:         {},
			},
			component.StatusOK: {
				component.StatusRecoverableError: {},
				component.StatusPermanentError:   {},
				component.StatusFatalError:       {},
				component.StatusStopping:         {},
			},
			component.StatusRecoverableError: {
				component.StatusOK:             {},
				component.StatusPermanentError: {},
				component.StatusFatalError:     {},
				component.StatusStopping:       {},
			},
			component.StatusPermanentError: {},
			component.StatusFatalError:     {},
			component.StatusStopping: {
				component.StatusRecoverableError: {},
				component.StatusPermanentError:   {},
				component.StatusFatalError:       {},
				component.StatusStopped:          {},
			},
			component.StatusStopped: {},
		},
	}
}

// InitFunc can be used to toggle a ready flag to true
type InitFunc func()

// readFunc can be used to check the value of a ready flag
type readyFunc func() bool

// initAndReadyFuncs returns a pair of functions to set and check a boolean ready flag
func initAndReadyFuncs() (InitFunc, readyFunc) {
	mu := sync.RWMutex{}
	isReady := false

	init := func() {
		mu.Lock()
		defer mu.Unlock()
		isReady = true
	}

	ready := func() bool {
		mu.RLock()
		defer mu.RUnlock()
		return isReady
	}

	return init, ready
}

// NotifyStatusFunc is the receiver of status events after successful state transitions
type NotifyStatusFunc func(*component.InstanceID, *component.StatusEvent)

// ServiceStatusFunc is the expected type of ReportComponentStatus for servicetelemetry.Settings
type ServiceStatusFunc func(*component.InstanceID, *component.StatusEvent) error

// errStatusNotReady is returned when trying to report status before service start
var errStatusNotReady = errors.New("report component status is not ready until service start")

// NewServiceStatusFunc returns a function to be used as ReportComponentStatus for
// servicetelemetry.Settings, which differs from component.TelemetrySettings in that
// the service version does not correspond to a specific component, and thus needs
// the a component.InstanceID as a parameter.
func NewServiceStatusFunc(notifyStatusChange NotifyStatusFunc) (InitFunc, ServiceStatusFunc) {
	init, isReady := initAndReadyFuncs()
	// mu synchronizes access to the fsmMap and the underlying fsm during a state transition
	mu := sync.Mutex{}
	fsmMap := make(map[*component.InstanceID]*fsm)
	return init,
		func(id *component.InstanceID, ev *component.StatusEvent) error {
			if !isReady() {
				return errStatusNotReady
			}
			mu.Lock()
			defer mu.Unlock()
			fsm, ok := fsmMap[id]
			if !ok {
				fsm = newFSM(func(ev *component.StatusEvent) {
					notifyStatusChange(id, ev)
				})
				fsmMap[id] = fsm
			}
			return fsm.transition(ev)
		}

}

// NewComponentStatusFunc returns a function to be used as ReportComponentStatus for
// component.TelemetrySettings, which differs from servicetelemetry.Settings in that
// the component version is tied to specific component instance.
func NewComponentStatusFunc(id *component.InstanceID, srvStatus ServiceStatusFunc) component.StatusFunc {
	return func(ev *component.StatusEvent) error {
		return srvStatus(id, ev)
	}
}
