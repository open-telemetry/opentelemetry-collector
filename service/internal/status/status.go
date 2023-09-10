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

var errInvalidStateTransition = errors.New("invalid state transition")

// fsm is a finite state machine that models transitions for component status
type fsm struct {
	current      *component.StatusEvent
	transitions  map[component.Status]map[component.Status]struct{}
	onTransition onTransitionFunc
}

// Transition will attempt to execute a state transition. If successful, it calls the onTransitionFunc
// with a StatusEvent representing the new state. Returns an error if the arguments result in an
// invalid status, or if the state transition is not valid.
func (m *fsm) Transition(status component.Status, options ...component.StatusEventOption) error {
	if _, ok := m.transitions[m.current.Status()][status]; !ok {
		return fmt.Errorf(
			"cannot transition from %s to %s: %w",
			m.current.Status(),
			status,
			errInvalidStateTransition,
		)
	}

	ev, err := component.NewStatusEvent(status, options...)
	if err != nil {
		return err
	}

	m.current = ev
	m.onTransition(ev)

	return nil
}

// newFSM creates a state machine with all valid transitions for component.Status.
// The initial state is set to component.StatusNone.
func newFSM(onTransition onTransitionFunc) *fsm {
	initial, _ := component.NewStatusEvent(component.StatusNone)
	return &fsm{
		current:      initial,
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
				component.StatusStopped:          {},
			},
			component.StatusOK: {
				component.StatusRecoverableError: {},
				component.StatusPermanentError:   {},
				component.StatusFatalError:       {},
				component.StatusStopping:         {},
				component.StatusStopped:          {},
			},
			component.StatusRecoverableError: {
				component.StatusOK:             {},
				component.StatusPermanentError: {},
				component.StatusFatalError:     {},
				component.StatusStopping:       {},
				component.StatusStopped:        {},
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

type InitFunc func()
type readyFunc func() bool

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

type NotifyStatusFunc func(*component.InstanceID, *component.StatusEvent)
type ServiceStatusFunc func(id *component.InstanceID, status component.Status, opts ...component.StatusEventOption) error

var errStatusNotReady = errors.New("report component status is not ready until service start")

// NewServiceStatusFunc returns a function to be used as ReportComponentStatus for
// servicetelemetry.Settings, which differs from component.TelemetrySettings in that
// the service version does not correspond to a specific component, and thus needs
// the a component.InstanceID as a parameter.
func NewServiceStatusFunc(notifyStatusChange NotifyStatusFunc) (InitFunc, ServiceStatusFunc) {
	init, isReady := initAndReadyFuncs()
	mu := sync.Mutex{}
	fsmMap := make(map[*component.InstanceID]*fsm)
	return init,
		func(id *component.InstanceID, status component.Status, opts ...component.StatusEventOption) error {
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
			return fsm.Transition(status, opts...)
		}

}

// NewComponentStatusFunc returns a function to be used as ReportComponentStatus for
// component.TelemetrySettings, which differs from servicetelemetry.Settings in that
// the component version is tied to specific component instance.
func NewComponentStatusFunc(id *component.InstanceID, srvStatus ServiceStatusFunc) component.StatusFunc {
	return func(status component.Status, opts ...component.StatusEventOption) error {
		return srvStatus(id, status, opts...)
	}
}
