// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package componentstatus is an experimental module that defines how components should
// report health statues, how collector hosts should facilitate component status reporting,
// and how extensions should watch for new component statuses.
//
// This package is currently under development and is exempt from the Collector SIG's
// breaking change policy.
package componentstatus // import "go.opentelemetry.io/collector/component/componentstatus"

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Reporter is an extra interface for `component.Host` implementations.
// A Reporter defines how to report a `componentstatus.Event`.
type Reporter interface {
	// Report allows a component to report runtime changes in status. The service
	// will automatically report status for a component during startup and shutdown. Components can
	// use this method to report status after start and before shutdown. For more details about
	// component status reporting see: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-status.md
	Report(*Event)
}

// Watcher is an extra interface for Extension hosted by the OpenTelemetry
// Collector that is to be implemented by extensions interested in changes to component
// status.
//
// TODO: consider moving this interface to a new package/module like `extension/statuswatcher`
// https://github.com/open-telemetry/opentelemetry-collector/issues/10764
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

// Event contains a status, a timestamp, optional error information, and additional attributes
type Event struct {
	// attributes provides additional context or metadata for the event.
	attributes pcommon.Map

	status Status
	err    error
	// TODO: consider if a timestamp is necessary in the default Event struct or is needed only for the healthcheckv2 extension
	// https://github.com/open-telemetry/opentelemetry-collector/issues/10763
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

// Attributes returns the attributes (pcommon.Map) associated with the Event.
func (ev *Event) Attributes() pcommon.Map {
	return ev.attributes
}

// Timestamp returns the timestamp associated with the Event
func (ev *Event) Timestamp() time.Time {
	return ev.timestamp
}

// EventBuilderOption is a sealed interface wrapping options for [NewEvent].
type EventBuilderOption interface {
	applyOption(*Event)
}

type eventOptionFunc func(*Event)

func (f eventOptionFunc) applyOption(event *Event) {
	f(event)
}

// NewEvent creates and returns an Event with the specified options and sets the timestamp
// time.Now(). To set an error on the event for an error status use the
// WithError with one of the dedicated status (e.g. StatusRecoverableError, StatusPermanentError, StatusFatalError)
func NewEvent(st Status, opts ...EventBuilderOption) *Event {
	event := &Event{
		timestamp:  time.Now(),
		attributes: pcommon.NewMap(),
		status:     st,
	}

	for _, opt := range opts {
		opt.applyOption(event)
	}

	return event
}

// WithAttributes sets the Event attributes.
func WithAttributes(attributes pcommon.Map) EventBuilderOption {
	return eventOptionFunc(func(e *Event) {
		e.attributes = attributes
	})
}

// WithError sets the Event error.
func WithError(err error) EventBuilderOption {
	return eventOptionFunc(func(e *Event) {
		e.err = err
	})
}

// NewRecoverableErrorEvent wraps a transient error
// passed as argument as a Event with a status StatusRecoverableError
// and a timestamp set to time.Now().
func NewRecoverableErrorEvent(err error) *Event {
	return NewEvent(StatusRecoverableError, WithError(err))
}

// NewPermanentErrorEvent wraps an error requiring human intervention to fix
// passed as argument as a Event with a status StatusPermanentError
// and a timestamp set to time.Now().
func NewPermanentErrorEvent(err error) *Event {
	return NewEvent(StatusPermanentError, WithError(err))
}

// NewFatalErrorEvent wraps the fatal runtime error passed as argument as a Event
// with a status StatusFatalError and a timestamp set to time.Now().
func NewFatalErrorEvent(err error) *Event {
	return NewEvent(StatusFatalError, WithError(err))
}

// StatusIsError returns true for error statuses (e.g. StatusRecoverableError,
// StatusPermanentError, or StatusFatalError)
func StatusIsError(status Status) bool {
	return status == StatusRecoverableError ||
		status == StatusPermanentError ||
		status == StatusFatalError
}

// ReportStatus is a helper function that handles checking if the component.Host has implemented Reporter.
// If it has, the Event is reported. Otherwise, nothing happens.
func ReportStatus(host component.Host, e *Event) {
	statusReporter, ok := host.(Reporter)
	if ok {
		statusReporter.Report(e)
	}
}
