// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

import (
	"errors"
)

type Status int32

const (
	StatusOK Status = iota
	StatusError
)

// StatusSource component that reports a status about itself.
// The implementation of this interface must be comparable to be useful as a map key.
type StatusSource interface {
	ID() ID
}

type StatusEvent struct {
	status Status
	err    error
}

func (ev *StatusEvent) Status() Status {
	return ev.status
}

// Err returns the error associated with the ComponentEvent.
func (ev *StatusEvent) Err() error {
	return ev.err
}

// StatusEventOption applies options to a StatusEvent.
type StatusEventOption func(*StatusEvent) error

// WithError sets the error object of the Event. It is optional
// and should only be applied to an Event of type ComponentError.
func WithError(err error) StatusEventOption {
	return func(o *StatusEvent) error {
		if o.status == StatusOK {
			return errors.New("event with ComponentOK cannot have an error")
		}
		o.err = err
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
	ComponentStatusChanged(source StatusSource, event *StatusEvent)
}
