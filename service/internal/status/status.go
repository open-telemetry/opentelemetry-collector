// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package status // import "go.opentelemetry.io/collector/service/internal/status"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/servicehost"
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

// newStatusFSM creates a state machine with all valid transitions for component.Status.
// It sets the initial state to component.StatusStarting and triggers the onTransitionFunc
// for the initial state.
func newStatusFSM(onTransition onTransitionFunc) *fsm {
	starting, _ := component.NewStatusEvent(component.StatusStarting)
	m := &fsm{
		current:      starting,
		onTransition: onTransition,
		transitions: map[component.Status]map[component.Status]struct{}{
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

	// fire initial starting event
	m.onTransition(starting)
	return m
}

// A Notifier emits status events
type Notifier interface {
	Event(status component.Status, options ...component.StatusEventOption) error
}

// NewNotifier returns a status.Notifier that reports component status through the given
// servicehost. The underlying implementation is a finite state machine.
func NewNotifier(host servicehost.Host, instanceID *component.InstanceID) Notifier {
	return newStatusFSM(
		func(ev *component.StatusEvent) {
			host.ReportComponentStatus(instanceID, ev)
		},
	)
}
