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

package featuregate

import (
	"fmt"
)

// Gates is the interface used to query the status of feature gates. It can be
// implemented by a read-only copy of the feature gate registry.
type Gates interface {
	// IsEnabled returns true if a registered feature gate is enabled and false otherwise.
	IsEnabled(id string) bool

	// List returns a slice of copies of all registered Gates.
	List() []Gate
}

// Gate represents an individual feature that may be enabled or disabled based
// on the lifecycle state of the feature and CLI flags specified by the user.
type Gate struct {
	ID          string
	Description string
	Enabled     bool
}

var reg = registry{gates: make(map[string]Gate)}

// Register a Gate. May only be called in an init() function. Not thread-safe.
// Will panic() if a Gate with the same ID is already registered.
func Register(g Gate) {
	if err := reg.add(g); err != nil {
		panic(err)
	}
}

// Apply a configuration in the form of a map of Gate identifiers to boolean values.
// Returns a copy of the global registry with the Gate Enabled properties set appropriately.
func Apply(cfg map[string]bool) Gates {
	return reg.apply(cfg)
}

type registry struct {
	gates map[string]Gate
}

func (r registry) apply(cfg map[string]bool) Gates {
	ret := registry{gates: make(map[string]Gate, len(reg.gates))}
	for _, gate := range r.gates {
		g := gate
		if state, ok := cfg[g.ID]; ok {
			g.Enabled = state
		}
		ret.gates[g.ID] = g
	}

	return ret
}

func (r registry) add(g Gate) error {
	if _, ok := r.gates[g.ID]; ok {
		return fmt.Errorf("attempted to add pre-existing gate %q", g.ID)
	}

	r.gates[g.ID] = g
	return nil
}

func (r registry) IsEnabled(id string) bool {
	g, ok := r.gates[id]
	return ok && g.Enabled
}

func (r registry) List() []Gate {
	ret := make([]Gate, len(r.gates))
	i := 0
	for _, gate := range r.gates {
		ret[i] = gate
		i++
	}

	return ret
}
