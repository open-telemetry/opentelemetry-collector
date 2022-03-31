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

// Deprecated: [v0.48.0] will be removed.
type Option func(*baseComponent)

// Deprecated: [v0.48.0] embed component.StartFunc directly.
func WithStart(startFunc component.StartFunc) Option {
	return func(o *baseComponent) {
		o.StartFunc = startFunc
	}
}

// Deprecated: [v0.48.0] embed component.ShutdownFunc directly.
func WithShutdown(shutdownFunc component.ShutdownFunc) Option {
	return func(o *baseComponent) {
		o.ShutdownFunc = shutdownFunc
	}
}

type baseComponent struct {
	component.StartFunc
	component.ShutdownFunc
}

// Deprecated: [v0.48.0] embed component.StartFunc and component.ShutdownFunc directly.
func New(options ...Option) component.Component {
	bc := &baseComponent{}

	for _, op := range options {
		op(bc)
	}

	return bc
}
