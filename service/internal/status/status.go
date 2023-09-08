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

// Event will attempt to execute a state transition. If successful, it calls the onTransitionFunc
// with a StatusEvent representing the new state. Returns an error if the arguments result in an
// invalid status, or if the state transition is not valid.
func (m *fsm) Event(status component.Status, options ...component.StatusEventOption) error {
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

type NotifyStatusFunc func(*component.InstanceID, *component.StatusEvent)
type ServiceStatusFunc func(id *component.InstanceID, status component.Status, opts ...component.StatusEventOption)

// NewServiceStatusFunc returns a function to be used as ReportComponentStatus for
// servicetelemetry.Settings, which differs from component.TelemetrySettings in that
// the service version does not correspond to a specific component, and thus needs
// the a component.InstanceID as a parameter.
func NewServiceStatusFunc(notifyStatusChange NotifyStatusFunc) ServiceStatusFunc {
	var fsmMap sync.Map
	return func(id *component.InstanceID, status component.Status, opts ...component.StatusEventOption) {
		f, ok := fsmMap.Load(id)
		if !ok {
			f = newFSM(func(ev *component.StatusEvent) {
				notifyStatusChange(id, ev)
			})
			if val, loaded := fsmMap.LoadOrStore(id, f); loaded {
				f = val
			}
		}
		_ = f.(*fsm).Event(status, opts...)
	}
}

// NewComponentStatusFunc returns a function to be used as ReportComponentStatus for
// component.TelemetrySettings, which differs from servicetelemetry.Settings in that
// the component version is tied to specific component instance.
func NewComponentStatusFunc(id *component.InstanceID, srvStatus ServiceStatusFunc) component.StatusFunc {
	return func(status component.Status, opts ...component.StatusEventOption) {
		srvStatus(id, status, opts...)
	}
}
