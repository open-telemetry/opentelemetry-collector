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

package status // import "go.opentelemetry.io/collector/component/status"

import (
	"time"

	"go.opentelemetry.io/collector/config"
)

// EventType represents an enumeration of status event types
type EventType int

const (
	// OK indicates the producer of a status event is functioning normally
	OK EventType = iota
	// RecoverableError is an error that can be retried, potentially with a successful outcome
	RecoverableError
	// PermanentError is an error that will be always returned if its source receives the same inputs
	PermanentError
	// FatalError is an error that cannot be recovered from and will cause early termination of the collector
	FatalError
)

// Event is a status event produced by a component to communicate its status to registered listeners.
// An event can signal that a component is working normally (i.e. Type: status.OK), or that it
// is in an error state (i.e. Type: status.RecoverableError). An error status may optionally
// include an error object to provide additional insight to registered listeners.
type Event struct {
	eventType   EventType
	timestamp   int64
	componentID config.ComponentID
	err         error
}

// Type returns the event type
func (ev *Event) Type() EventType {
	return ev.eventType
}

// Timestamp returns the time of the event in nanos
func (ev *Event) Timestamp() int64 {
	return ev.timestamp
}

// ComponentID returns the ID of the component that generated the status.Event
func (ev *Event) ComponentID() config.ComponentID {
	return ev.componentID
}

// Err returns the error associated with the status.Event
func (ev *Event) Err() error {
	return ev.err
}

// EventOption applies options to a status.Event
type EventOption func(*Event)

// WithComponentID sets the ComponentID that generated the status.Event
func WithComponentID(componentID config.ComponentID) EventOption {
	return func(o *Event) {
		o.componentID = componentID
	}
}

// WithTimestamp sets the timestamp, expected in nanos, for a status.Event
func WithTimestamp(nanos int64) EventOption {
	return func(o *Event) {
		o.timestamp = nanos
	}
}

// WithError assigns an error object to a Event. It is optional and should only be applied
// to a Event with type status.EventError.
func WithError(err error) EventOption {
	return func(o *Event) {
		o.err = err
	}
}

// NewEvent creates and returns a status.Event with default and / or the provided options
func NewEvent(eventType EventType, options ...EventOption) *Event {
	ev := Event{
		eventType: eventType,
	}

	for _, opt := range options {
		opt(&ev)
	}

	if ev.timestamp == 0 {
		ev.timestamp = time.Now().UnixNano()
	}

	return &ev
}

// EventFunc is a callback function that receives status.Events
type EventFunc func(event *Event) error

// PipelineFunc is a function to be called when the collector pipeline changes states
type PipelineFunc func() error

// UnregisterFunc is a function to be called to unregister a component that has previously
// registered to listen to status notifications
type UnregisterFunc func() error

var noopStatusEventFunc = func(event *Event) error { return nil }

var noopPipelineFunc = func() error { return nil }

// Listener is a struct that manages handlers to status and pipeline events
type Listener struct {
	statusEventHandler      EventFunc
	pipelineReadyHandler    PipelineFunc
	pipelineNotReadyHandler PipelineFunc
}

// StatusEventHandler delegates to the underlying handler registered to the Listener
func (l *Listener) StatusEventHandler(event *Event) error {
	return l.statusEventHandler(event)
}

// PipelineReadyHandler delegates to the underlying handler registered to the Listener
func (l *Listener) PipelineReadyHandler() error {
	return l.pipelineReadyHandler()
}

// PipelineNotReadyHandler delegates to the underlying handler registered to the Listener
func (l *Listener) PipelineNotReadyHandler() error {
	return l.pipelineNotReadyHandler()
}

// ListenerOption applies options to a status listener
type ListenerOption func(*Listener)

// WithStatusEventHandler allows you to configure callback for status events
func WithStatusEventHandler(handler EventFunc) ListenerOption {
	return func(o *Listener) {
		o.statusEventHandler = handler
	}
}

// WithPipelineReadyReayHandler allows you configure a callback to be executed when the pipeline
// state changes to "ready"
func WithPipelineReadyHandler(handler PipelineFunc) ListenerOption {
	return func(o *Listener) {
		o.pipelineReadyHandler = handler
	}
}

// WithPipelineReadyReayHandler allows you configure a callback to be executed when the pipeline
// state changes to "not ready"
func WithPipelineNotReadyHandler(handler PipelineFunc) ListenerOption {
	return func(o *Listener) {
		o.pipelineNotReadyHandler = handler
	}
}

func NewListener(options ...ListenerOption) *Listener {
	l := &Listener{
		statusEventHandler:      noopStatusEventFunc,
		pipelineReadyHandler:    noopPipelineFunc,
		pipelineNotReadyHandler: noopPipelineFunc,
	}

	for _, opt := range options {
		opt(l)
	}

	return l
}
