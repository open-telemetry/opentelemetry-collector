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

package featuregate // import "go.opentelemetry.io/collector/service/featuregate"

import (
	"fmt"
	"sync"
)

// Gate represents an individual feature that may be enabled or disabled based
// on the lifecycle state of the feature and CLI flags specified by the user.
type Gate struct {
	ID          string
	Description string
	Enabled     bool
}

var reg = &Registry{gates: make(map[string]Gate)}

func GetRegistry() *Registry {
	return reg
}

func NewRegistry() Registry {
	return Registry{gates: make(map[string]Gate)}
}

// Register a Gate. May only be called in an init() function.
// Will panic() if a Gate with the same ID is already registered.
func (r *Registry) Register(g Gate) {
	if err := r.add(g); err != nil {
		panic(err)
	}
}

type Registry struct {
	sync.RWMutex
	gates map[string]Gate
}

// Apply a configuration in the form of a map of Gate identifiers to boolean values.
// Sets only those values provided in the map, other gate values are not changed.
func (r *Registry) Apply(cfg map[string]bool) {
	r.Lock()
	defer r.Unlock()
	for id, val := range cfg {
		if g, ok := r.gates[id]; ok {
			g.Enabled = val
			r.gates[g.ID] = g
		}
	}
}

func (r *Registry) add(g Gate) error {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.gates[g.ID]; ok {
		return fmt.Errorf("attempted to add pre-existing gate %q", g.ID)
	}

	r.gates[g.ID] = g
	return nil
}

// IsEnabled returns true if a registered feature gate is enabled and false otherwise.
func (r *Registry) IsEnabled(id string) bool {
	r.RLock()
	defer r.RUnlock()
	g, ok := r.gates[id]
	return ok && g.Enabled
}

// List returns a slice of copies of all registered Gates.
func (r *Registry) List() []Gate {
	r.RLock()
	defer r.RUnlock()
	ret := make([]Gate, len(r.gates))
	i := 0
	for _, gate := range r.gates {
		ret[i] = gate
		i++
	}

	return ret
}
