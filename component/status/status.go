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

type EventType int

const (
	EventOK EventType = iota
	EventError
)

type StatusEvent struct {
	Type        EventType
	Timestamp   int64
	ComponentID config.ComponentID
	Error       error
}

type StatusEventOption func(*StatusEvent)

func WithError(err error) StatusEventOption {
	return func(o *StatusEvent) {
		o.Error = err
	}
}

func WithTimestamp(nanos int64) StatusEventOption {
	return func(o *StatusEvent) {
		o.Timestamp = nanos
	}
}

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

type StatusEventFunc func(event StatusEvent) error

type PipelineEventFunc func() error

type UnregisterFunc func() error

var noopStatusEventFunc = func(event StatusEvent) error { return nil }

var noopPipelineFunc = func() error { return nil }

type listener struct {
	statusEventHandler      StatusEventFunc
	pipelineReadyHandler    PipelineEventFunc
	pipelineNotReadyHandler PipelineEventFunc
}

type ListenerOption func(*listener)

func WithStatusEventHandler(handler StatusEventFunc) ListenerOption {
	return func(o *listener) {
		o.statusEventHandler = handler
	}
}

func WithPipelineReadyHandler(handler PipelineEventFunc) ListenerOption {
	return func(o *listener) {
		o.pipelineReadyHandler = handler
	}
}

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
