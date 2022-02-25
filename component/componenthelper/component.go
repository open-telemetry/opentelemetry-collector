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

package componenthelper // import "go.opentelemetry.io/collector/component/componenthelper"

import (
	"go.opentelemetry.io/collector/component"
)

// Deprecated: [v0.46.0] use component.StartFunc.
type StartFunc = component.StartFunc

// Deprecated: [v0.46.0] use component.ShutdownFunc.
type ShutdownFunc = component.ShutdownFunc

// Option represents the possible options for New.
type Option func(*baseComponent)

// WithStart overrides the default `Start` function for a component.Component.
// The default always returns nil.
func WithStart(startFunc component.StartFunc) Option {
	return func(o *baseComponent) {
		o.StartFunc = startFunc
	}
}

// WithShutdown overrides the default `Shutdown` function for a component.Component.
// The default always returns nil.
func WithShutdown(shutdownFunc component.ShutdownFunc) Option {
	return func(o *baseComponent) {
		o.ShutdownFunc = shutdownFunc
	}
}

type baseComponent struct {
	component.StartFunc
	component.ShutdownFunc
}

// New returns a component.Component configured with the provided options.
func New(options ...Option) component.Component {
	bc := &baseComponent{}

	for _, op := range options {
		op(bc)
	}

	return bc
}
