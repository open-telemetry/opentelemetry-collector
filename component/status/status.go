// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package status

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component/id"
)

// ComponentEventType represents an enumeration of component status event types.
type ComponentEventType int

const (
	// ComponentOK indicates the component is functioning normally.
	ComponentOK ComponentEventType = iota
	// ComponentError indicates the component is in erroneous state.
	ComponentError
)

// Source component that reports a status about itself.
// The implementation of this interface must be comparable to be useful as a map key.
type Source interface {
	ID() id.ID
}

// ComponentEvent is a status event produced by a component to communicate its status to
// registered listeners.
// An event can signal that a component is working normally (status.ComponentOK),
// or that it is in an error state (status.ComponentError). An error status may
// optionally include an error object to provide additional insight to
// registered listeners.
type ComponentEvent struct {
	eventType ComponentEventType
	timestamp time.Time
	source    Source
	err       error
}

// Type returns the event type.
func (ev *ComponentEvent) Type() ComponentEventType {
	return ev.eventType
}

// Source is the component that is reporting its status.
func (ev *ComponentEvent) Source() Source {
	return ev.source
}

// Timestamp returns the time of the event.
func (ev *ComponentEvent) Timestamp() time.Time {
	return ev.timestamp
}

// Err returns the error associated with the ComponentEvent.
func (ev *ComponentEvent) Err() error {
	return ev.err
}

// ComponentEventOption applies options to a ComponentEvent.
type ComponentEventOption func(*ComponentEvent) error

// WithSource sets the source component for a ComponentEvent.
func WithSource(source Source) ComponentEventOption {
	return func(o *ComponentEvent) error {
		o.source = source
		return nil
	}
}

// WithTimestamp sets the timestamp for a ComponentEvent.
func WithTimestamp(timestamp time.Time) ComponentEventOption {
	return func(o *ComponentEvent) error {
		o.timestamp = timestamp
		return nil
	}
}

// WithError sets the error object of the Event. It is optional
// and should only be applied to an Event of type ComponentError.
func WithError(err error) ComponentEventOption {
	return func(o *ComponentEvent) error {
		if o.eventType == ComponentOK {
			return errors.New("event with ComponentOK cannot have an error")
		}
		o.err = err
		return nil
	}
}

// NewComponentEvent creates and returns a ComponentEvent with default and provided
// options. Will return an error if an error is provided for a non-error event
// type (status.ComponentOK).
// If the timestamp is not provided will set it to time.Now().
func NewComponentEvent(
	eventType ComponentEventType, options ...ComponentEventOption,
) (*ComponentEvent, error) {
	ev := ComponentEvent{
		eventType: eventType,
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

// ComponentEventFunc is a callback function that receives ComponentEvent.
type ComponentEventFunc func(event *ComponentEvent) error

var noopComponentEventFunc = func(event *ComponentEvent) error { return nil }
