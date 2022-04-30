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
	// EventOK indicates the producer of a status event is functioning normally
	EventOK EventType = iota
	// EventError indicates the producer of a status event is in an error state
	EventError
)

// StatusEvent is an event produced by a component to communicate its status to registered listeners.
// An event can signal that a component is working normally (i.e. Type: status.EventOK), or that it
// is in an error state (i.e. Type: status.EventError). An error status may optionally include an
// error object to provide additonal insight to registered listeners.
type StatusEvent struct {
	Type        EventType
	Timestamp   int64
	ComponentID config.ComponentID
	Error       error
}

// StatusEventOption applies options to a StatusEvent
type StatusEventOption func(*StatusEvent)

// WithError assigns an error object to a StatusEvent. It is optional and should only be applied
// to a StatusEvent with type status.EventError.
func WithError(err error) StatusEventOption {
	return func(o *StatusEvent) {
		o.Error = err
	}
}

// WithTimestamp sets the timestamp, expected in nanos, for a StatusEvent
func WithTimestamp(nanos int64) StatusEventOption {
	return func(o *StatusEvent) {
		o.Timestamp = nanos
	}
}

// NewStatusEvent creates and returns a StatusEvent with default and / or the provided options
func NewStatusEvent(eventType EventType, componentID config.ComponentID, options ...StatusEventOption) StatusEvent {
	ev := StatusEvent{
		Type:        eventType,
		ComponentID: componentID,
	}

	for _, opt := range options {
		opt(&ev)
	}

	if ev.Timestamp == 0 {
		ev.Timestamp = time.Now().UnixNano()
	}

	return ev
}

// StatusEventFunc is a callback function that receives StatusEvents
type StatusEventFunc func(event StatusEvent) error

// PipelineEventFunc is a function to be called when the collector pipeline changes states
type PipelineEventFunc func() error

// UnregisterFunc is a function to be called to unregister a component that has previously
// registered to listen to status notifications
type UnregisterFunc func() error

var noopStatusEventFunc = func(event StatusEvent) error { return nil }

var noopPipelineFunc = func() error { return nil }

type listener struct {
	statusEventHandler      StatusEventFunc
	pipelineReadyHandler    PipelineEventFunc
	pipelineNotReadyHandler PipelineEventFunc
}

// ListenerOption applies options to a status listener
type ListenerOption func(*listener)

// WithStatusEventHandler allows you to configure callback for status events
func WithStatusEventHandler(handler StatusEventFunc) ListenerOption {
	return func(o *listener) {
		o.statusEventHandler = handler
	}
}

// WithPipelineReadyReayHandler allows you configure a callback to be executed when the pipeline
// state changes to "ready"
func WithPipelineReadyHandler(handler PipelineEventFunc) ListenerOption {
	return func(o *listener) {
		o.pipelineReadyHandler = handler
	}
}

// WithPipelineReadyReayHandler allows you configure a callback to be executed when the pipeline
// state changes to "not ready"
func WithPipelineNotReadyHandler(handler PipelineEventFunc) ListenerOption {
	return func(o *listener) {
		o.pipelineNotReadyHandler = handler
	}
}

func newListener(options ...ListenerOption) *listener {
	l := &listener{
		statusEventHandler:      noopStatusEventFunc,
		pipelineReadyHandler:    noopPipelineFunc,
		pipelineNotReadyHandler: noopPipelineFunc,
	}

	for _, opt := range options {
		opt(l)
	}

	return l
}
