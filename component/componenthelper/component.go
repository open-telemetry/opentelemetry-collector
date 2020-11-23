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

// ComponentSettings represents a settings struct to create components.
type ComponentSettings struct {
	Start
	Shutdown
}

// DefaultComponentSettings returns the default settings for a component. The Start and Shutdown are no-op.
func DefaultComponentSettings() *ComponentSettings {
	return &ComponentSettings{
		Start:    func(ctx context.Context, host component.Host) error { return nil },
		Shutdown: func(ctx context.Context) error { return nil },
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

// NewComponent returns a component.Component that calls the given Start and Shutdown.
func NewComponent(s *ComponentSettings) component.Component {
	return &baseComponent{
		start:    s.Start,
		shutdown: s.Shutdown,
	}
}
