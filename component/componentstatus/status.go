// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentstatus // import "go.opentelemetry.io/collector/component/componentstatus"

import (
	"time"
)

// Watcher is an extra interface for Extension hosted by the OpenTelemetry
// Collector that is to be implemented by extensions interested in changes to component
// status.
type Watcher interface {
	// ComponentStatusChanged notifies about a change in the source component status.
	// Extensions that implement this interface must be ready that the ComponentStatusChanged
	// may be called before, after or concurrently with calls to Component.Start() and Component.Shutdown().
	// The function may be called concurrently with itself.
	ComponentStatusChanged(source *InstanceID, event *Event)
}

type Status int32

// Enumeration of possible component statuses
const (
	// StatusNone indicates absence of component status.
	StatusNone Status = iota
	// StatusStarting indicates the component is starting.
	StatusStarting
	// StatusOK indicates the component is running without issues.
	StatusOK
	// StatusRecoverableError indicates that the component has experienced a transient error and may recover.
	StatusRecoverableError
	// StatusPermanentError indicates that the component has detected a condition at runtime that will need human intervention to fix. The collector will continue to run in a degraded mode.
	StatusPermanentError
	// StatusFatalError indicates that the collector has experienced a fatal runtime error and will shut down.
	StatusFatalError
	// StatusStopping indicates that the component is in the process of shutting down.
	StatusStopping
	// StatusStopped indicates that the component has completed shutdown.
	StatusStopped
)

// String returns a string representation of a Status
func (s Status) String() string {
	switch s {
	case StatusStarting:
		return "StatusStarting"
	case StatusOK:
		return "StatusOK"
	case StatusRecoverableError:
		return "StatusRecoverableError"
	case StatusPermanentError:
		return "StatusPermanentError"
	case StatusFatalError:
		return "StatusFatalError"
	case StatusStopping:
		return "StatusStopping"
	case StatusStopped:
		return "StatusStopped"
	}
	return "StatusNone"
}

// Event contains a status and timestamp, and can contain an error
type Event struct {
	status    Status
	err       error
	timestamp time.Time
}

// Status returns the Status (enum) associated with the Event
func (ev *Event) Status() Status {
	return ev.status
}

// Err returns the error associated with the Event.
func (ev *Event) Err() error {
	return ev.err
}

// Timestamp returns the timestamp associated with the Event
func (ev *Event) Timestamp() time.Time {
	return ev.timestamp
}

// NewEvent creates and returns a Event with the specified status and sets the timestamp
// time.Now(). To set an error on the event for an error status use one of the dedicated
// constructors (e.g. NewRecoverableErrorEvent, NewPermanentErrorEvent, NewFatalErrorEvent)
func NewEvent(status Status) *Event {
	return &Event{
		status:    status,
		timestamp: time.Now(),
	}
}

// NewRecoverableErrorEvent wraps a transient error
// passed as argument as a Event with a status StatusRecoverableError
// and a timestamp set to time.Now().
func NewRecoverableErrorEvent(err error) *Event {
	ev := NewEvent(StatusRecoverableError)
	ev.err = err
	return ev
}

// NewPermanentErrorEvent wraps an error requiring human intervention to fix
// passed as argument as a Event with a status StatusPermanentError
// and a timestamp set to time.Now().
func NewPermanentErrorEvent(err error) *Event {
	ev := NewEvent(StatusPermanentError)
	ev.err = err
	return ev
}

// NewFatalErrorEvent wraps the fatal runtime error passed as argument as a Event
// with a status StatusFatalError and a timestamp set to time.Now().
func NewFatalErrorEvent(err error) *Event {
	ev := NewEvent(StatusFatalError)
	ev.err = err
	return ev
}

// AggregateStatus will derive a status for the given input using the following rules in order:
//  1. If all instances have the same status, there is nothing to aggregate, return it.
//  2. If any instance encounters a fatal error, the component is in a Fatal Error state.
//  3. If any instance is in a Permanent Error state, the component status is Permanent Error.
//  4. If any instance is Stopping, the component is in a Stopping state.
//  5. An instance is Stopped, but not all instances are Stopped, we must be in the process of Stopping the component.
//  6. If any instance is in a Recoverable Error state, the component status is Recoverable Error.
//  7. By process of elimination, the only remaining state is starting.
func AggregateStatus[K comparable](eventMap map[K]*Event) Status {
	seen := make(map[Status]struct{})
	for _, ev := range eventMap {
		seen[ev.Status()] = struct{}{}
	}

	// All statuses are the same. Note, this will handle StatusOK and StatusStopped as these two
	// cases require all components be in the same state.
	if len(seen) == 1 {
		for st := range seen {
			return st
		}
	}

	// Handle mixed status cases
	if _, isFatal := seen[StatusFatalError]; isFatal {
		return StatusFatalError
	}

	if _, isPermanent := seen[StatusPermanentError]; isPermanent {
		return StatusPermanentError
	}

	if _, isStopping := seen[StatusStopping]; isStopping {
		return StatusStopping
	}

	if _, isStopped := seen[StatusStopped]; isStopped {
		return StatusStopping
	}

	if _, isRecoverable := seen[StatusRecoverableError]; isRecoverable {
		return StatusRecoverableError
	}

	// By process of elimination, this is the last possible status; no check necessary.
	return StatusStarting
}

// StatusIsError returns true for error statuses (e.g. StatusRecoverableError,
// StatusPermanentError, or StatusFatalError)
func StatusIsError(status Status) bool {
	return status == StatusRecoverableError ||
		status == StatusPermanentError ||
		status == StatusFatalError
}

// AggregateStatusEvent returns a status event where:
//   - The status is set to the aggregate status of the events in the eventMap
//   - The timestamp is set to the latest timestamp of the events in the eventMap
//   - For an error status, the event will have same error as the most current event of the same
//     error type from the eventMap
func AggregateStatusEvent[K comparable](eventMap map[K]*Event) *Event {
	var lastEvent, lastMatchingEvent *Event
	aggregateStatus := AggregateStatus[K](eventMap)

	for _, ev := range eventMap {
		if lastEvent == nil || lastEvent.timestamp.Before(ev.timestamp) {
			lastEvent = ev
		}
		if aggregateStatus == ev.Status() &&
			(lastMatchingEvent == nil || lastMatchingEvent.timestamp.Before(ev.timestamp)) {
			lastMatchingEvent = ev
		}
	}

	// the effective status matches an existing event
	if lastEvent.Status() == aggregateStatus {
		return lastEvent
	}

	// the effective status requires a synthetic event
	aggregateEvent := &Event{
		status:    aggregateStatus,
		timestamp: lastEvent.timestamp,
	}
	if StatusIsError(aggregateStatus) {
		aggregateEvent.err = lastMatchingEvent.err
	}

	return aggregateEvent
}
