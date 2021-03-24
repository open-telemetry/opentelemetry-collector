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

package componenthelper

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

// Start specifies the function invoked when the exporter is being started.
type Start func(context.Context, component.Host) error

// Shutdown specifies the function invoked when the exporter is being shutdown.
type Shutdown func(context.Context) error

// baseSettings represents a settings struct to create components.
type baseSettings struct {
	Start
	Shutdown
}

// Option represents the possible options for New.
type Option func(*baseSettings)

// WithStart overrides the default Start function for a processor.
// The default shutdown function does nothing and always returns nil.
func WithStart(start Start) Option {
	return func(o *baseSettings) {
		o.Start = start
	}
}

// WithShutdown overrides the default Shutdown function for a processor.
// The default shutdown function does nothing and always returns nil.
func WithShutdown(shutdown Shutdown) Option {
	return func(o *baseSettings) {
		o.Shutdown = shutdown
	}
}

type baseComponent struct {
	start    Start
	shutdown Shutdown
}

// Start all senders and exporter and is invoked during service start.
func (be *baseComponent) Start(ctx context.Context, host component.Host) error {
	return be.start(ctx, host)
}

// Shutdown all senders and exporter and is invoked during service shutdown.
func (be *baseComponent) Shutdown(ctx context.Context) error {
	return be.shutdown(ctx)
}

// fromOptions returns the internal settings starting from the default and applying all options.
func fromOptions(options []Option) *baseSettings {
	opts := &baseSettings{
		Start:    func(ctx context.Context, host component.Host) error { return nil },
		Shutdown: func(ctx context.Context) error { return nil },
	}

	for _, op := range options {
		op(opts)
	}

	return opts
}

// New returns a component.Component configured with the provided Options.
func New(options ...Option) component.Component {
	bs := fromOptions(options)
	return &baseComponent{
		start:    bs.Start,
		shutdown: bs.Shutdown,
	}
}
