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

package component // import "go.opentelemetry.io/collector/component/status"

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/config"
)

// StatusEventType represents an enumeration of status event types
type StatusEventType int

const (
	// StatusOK indicates the producer of a status event is functioning normally
	StatusOK StatusEventType = iota
	// StatusRecoverableError is an error that can be retried, potentially with a successful outcome
	StatusRecoverableError
	// StatusPermanentError is an error that will be always returned if its source receives the same inputs
	StatusPermanentError
	// StatusFatalError is an error that cannot be recovered from and will cause early termination of the collector
	StatusFatalError
)

// StatusEvent is a status event produced by a component to communicate its status to registered listeners.
// An event can signal that a component is working normally (i.e. Type: component.OK), or that it
// is in an error state (i.e. Type: component.RecoverableError). An error status may optionally
// include an error object to provide additional insight to registered listeners.
type StatusEvent struct {
	eventType     StatusEventType
	timestamp     time.Time
	componentKind Kind
	componentID   config.ComponentID
	err           error
}

// Type returns the event type
func (ev *StatusEvent) Type() StatusEventType {
	return ev.eventType
}

// Timestamp returns the time of the event in nanos
func (ev *StatusEvent) Timestamp() time.Time {
	return ev.timestamp
}

// ComponentKind returns the kind of the component that generated the StatusEvent
func (ev *StatusEvent) ComponentKind() Kind {
	return ev.componentKind
}

// ComponentID returns the ID of the component that generated the StatusEvent
func (ev *StatusEvent) ComponentID() config.ComponentID {
	return ev.componentID
}

// Err returns the error associated with the StatusEvent
func (ev *StatusEvent) Err() error {
	return ev.err
}

// StatusEventOption applies options to a StatusEvent
type StatusEventOption func(*StatusEvent) error

// WithComponentID sets the ComponentID that generated the StatusEvent
func WithComponentID(componentID config.ComponentID) StatusEventOption {
	return func(o *StatusEvent) error {
		o.componentID = componentID
		return nil
	}
}

// WithComponentKind sets the component kind that generated the StatusEvent
func WithComponentKind(componentKind Kind) StatusEventOption {
	return func(o *StatusEvent) error {
		o.componentKind = componentKind
		return nil
	}
}

// WithTimestamp sets the timestamp, expected in nanos, for a StatusEvent
func WithTimestamp(timestamp time.Time) StatusEventOption {
	return func(o *StatusEvent) error {
		o.timestamp = timestamp
		return nil
	}
}

// WithError assigns an error object to a Event. It is optional and should only be applied
// to a Event of type: RecoverableError, PermanentError, or FatalError.
func WithError(err error) StatusEventOption {
	return func(o *StatusEvent) error {
		if o.eventType == StatusOK {
			return errors.New("event with component.OK cannot have an error")
		}
		o.err = err
		return nil
	}
}

// NewStatusEvent creates and returns a StatusEvent with default and / or the provided options. Will
// return an error if an error is provided for a non-error event type (e.g. component.OK)
func NewStatusEvent(eventType StatusEventType, options ...StatusEventOption) (*StatusEvent, error) {
	ev := StatusEvent{
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

// StatusEventFunc is a callback function that receives StatusEvents
type StatusEventFunc func(event *StatusEvent) error

// PipelineFunc is a function to be called when the collector pipeline changes states
type PipelineFunc func() error

// UnregisterFunc is a function to be called to unregister a component that has previously
// registered to listen to status notifications
type StatusUnregisterFunc func() error

var noopStatusEventFunc = func(event *StatusEvent) error { return nil }

var noopPipelineFunc = func() error { return nil }

// StatusListener is a struct that manages handlers to status and pipeline events
type StatusListener struct {
	statusEventHandler      StatusEventFunc
	pipelineReadyHandler    PipelineFunc
	pipelineNotReadyHandler PipelineFunc
}

// StatusEventHandler delegates to the underlying handler registered to the Listener
func (l *StatusListener) StatusEventHandler(event *StatusEvent) error {
	return l.statusEventHandler(event)
}

// PipelineReadyHandler delegates to the underlying handler registered to the Listener
func (l *StatusListener) PipelineReadyHandler() error {
	return l.pipelineReadyHandler()
}

// PipelineNotReadyHandler delegates to the underlying handler registered to the Listener
func (l *StatusListener) PipelineNotReadyHandler() error {
	return l.pipelineNotReadyHandler()
}

// StatusListenerOption applies options to a status listener
type StatusListenerOption func(*StatusListener)

// WithStatusEventHandler allows you to configure callback for status events
func WithStatusEventHandler(handler StatusEventFunc) StatusListenerOption {
	return func(o *StatusListener) {
		o.statusEventHandler = handler
	}
}

// WithPipelineReadyReayHandler allows you configure a callback to be executed when the pipeline
// state changes to "ready"
func WithPipelineReadyHandler(handler PipelineFunc) StatusListenerOption {
	return func(o *StatusListener) {
		o.pipelineReadyHandler = handler
	}
}

// WithPipelineReadyReayHandler allows you configure a callback to be executed when the pipeline
// state changes to "not ready"
func WithPipelineNotReadyHandler(handler PipelineFunc) StatusListenerOption {
	return func(o *StatusListener) {
		o.pipelineNotReadyHandler = handler
	}
}

func NewStatusListener(options ...StatusListenerOption) *StatusListener {
	l := &StatusListener{
		statusEventHandler:      noopStatusEventFunc,
		pipelineReadyHandler:    noopPipelineFunc,
		pipelineNotReadyHandler: noopPipelineFunc,
	}

	for _, opt := range options {
		opt(l)
	}

	return l
}
