// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

import (
	"errors"
	"fmt"
	"time"
)

type Status int32

// Enumeration of possible component statuses
const (
	StatusStarting Status = iota
	StatusOK
	StatusRecoverableError
	StatusPermanentError
	StatusFatalError
	StatusStopping
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
	return "StatusUnknown"
}

// errorStatuses is a set of statuses that can have associated errors
var errorStatuses = map[Status]struct{}{
	StatusRecoverableError: {},
	StatusPermanentError:   {},
	StatusFatalError:       {},
}

// StatusEvent contains a status and timestamp, and can contain an error
type StatusEvent struct {
	status    Status
	err       error
	timestamp time.Time
}

// Status returns the Status (enum) associated with the StatusEvent
func (ev *StatusEvent) Status() Status {
	return ev.status
}

// Err returns the error associated with the StatusEvent.
func (ev *StatusEvent) Err() error {
	return ev.err
}

// Timestamp returns the timestamp associated with the StatusEvent
func (ev *StatusEvent) Timestamp() time.Time {
	return ev.timestamp
}

// StatusEventOption applies options to a StatusEvent.
type StatusEventOption func(*StatusEvent) error

// errStatusEventInvalidArgument indicates an invalid option was specified when creating a status
// event. This will happen when using WithError for a non-error status.
var errStatusEventInvalidArgument = errors.New("status event argument error")

// WithError sets the error object of the StatusEvent. It is optional
// and should only be applied to an Event of type ComponentError.
func WithError(err error) StatusEventOption {
	return func(o *StatusEvent) error {
		if _, ok := errorStatuses[o.status]; !ok {
			return fmt.Errorf(
				"event with %s cannot have an error: %w",
				o.status,
				errStatusEventInvalidArgument,
			)
		}
		o.err = err
		return nil
	}
}

// WithTimestamp is optional, when used it sets the timestamp of the StatusEvent.
func WithTimestamp(t time.Time) StatusEventOption {
	return func(o *StatusEvent) error {
		o.timestamp = t
		return nil
	}
}

// NewStatusEvent creates and returns a StatusEvent with default and provided
// options. Will return an error if an error is provided for a non-error event
// type (status.ComponentOK).
// If the timestamp is not provided will set it to time.Now().
func NewStatusEvent(status Status, options ...StatusEventOption) (*StatusEvent, error) {
	ev := StatusEvent{
		status: status,
	}

	for _, opt := range options {
		if err := opt(&ev); err != nil {
			return nil, err
		}
	}

	if ev.timestamp.IsZero() {
		ev.timestamp = time.Now()
	}

	return &ev, nil
}

// StatusWatcher is an extra interface for Extension hosted by the OpenTelemetry
// Collector that is to be implemented by extensions interested in changes to component
// status.
type StatusWatcher interface {
	// ComponentStatusChanged notifies about a change in the source component status.
	// Extensions that implement this interface must be ready that the ComponentStatusChanged
	// may be called  before, after or concurrently with Component.Shutdown() call.
	// The function may be called concurrently with itself.
	ComponentStatusChanged(source *InstanceID, event *StatusEvent)
}
