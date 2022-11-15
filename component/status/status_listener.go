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

// Listener is a struct that holds component and pipeline status events handlers.
type Listener struct {
	componentEventHandler ComponentEventFunc
	pipelineStatusHandler PipelineStatusFunc
}

// ComponentEventHandler delegates to the underlying handler registered to the Listener.
func (l *Listener) ComponentEventHandler(event *ComponentEvent) error {
	return l.componentEventHandler(event)
}

// PipelineStatusHandler delegates to the underlying handler registered to the Listener.
func (l *Listener) PipelineStatusHandler(status PipelineReadiness) error {
	return l.pipelineStatusHandler(status)
}

// ListenerOption applies options to a Listener.
type ListenerOption func(*Listener)

// WithComponentEventHandler allows you to supply a callback to be executed on
// component status events.
func WithComponentEventHandler(handler ComponentEventFunc) ListenerOption {
	return func(o *Listener) {
		o.componentEventHandler = handler
	}
}

// WithPipelineStatusHandler allows to supply a callback to be executed when the pipeline
// status changes.
func WithPipelineStatusHandler(handler PipelineStatusFunc) ListenerOption {
	return func(o *Listener) {
		o.pipelineStatusHandler = handler
	}
}

func NewStatusListener(options ...ListenerOption) *Listener {
	l := &Listener{
		componentEventHandler: noopComponentEventFunc,
		pipelineStatusHandler: noopPipelineStatusFunc,
	}

	for _, opt := range options {
		opt(l)
	}

	return l
}
