// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lifecycle // import "go.opentelemetry.io/collector/internal/lifecycle"

import "go.uber.org/atomic"

// Lifecycle represents the current
// running state of the component
// that is intended to be concurrency safe.
type State interface {
	// Start atomically updates the state and returns true if
	// the previous state was not running.
	Start() (started bool)

	// Stop atomically updates the state and returns true if
	// the previous state was not shutdown.
	Stop() (stopped bool)
}

type state struct {
	current *atomic.Int64
}

const (
	shutdown = iota
	running
)

var (
	_ State = (*state)(nil)
)

func NewState() State {
	return &state{current: atomic.NewInt64(shutdown)}
}

func (s *state) Start() (started bool) {
	return s.current.Swap(running) == shutdown
}

func (s *state) Stop() (stopped bool) {
	return s.current.Swap(shutdown) == running
}
