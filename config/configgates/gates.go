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

package configgates

import (
	"fmt"
	"sync"
)

// Gates is the interface used to query the status of feature gates. It can be
// implemented by a read-only copy of the feature gate registry.
type Gates interface {
	IsEnabled(id string) bool
}

// Gate represents an individual feature that may be enabled or disabled based
// on the lifecycle state of the feature and CLI flags specified by the user.
type Gate struct {
	Id          string
	Description string
	Enabled     bool
}

// Registry holds information about available feature gates and their enablement status.
//
// Components that wish to register a feature gate should do so using the global Registry
// available from GetRegistry() in an init() function to ensure that the gate is registered
// as soon as possible and thus available to be managed by configuration and CLI flags.
type Registry struct {
	sync.RWMutex
	gates map[string]*Gate
}

var registry = &Registry{gates: make(map[string]*Gate)}

// GetRegistry returns the global Registry that should be used by components that are
// responsible for registering feature gates and processing configuration.
func GetRegistry() *Registry {
	return registry
}

// IsEnabled returns true if a registered Gate is enabled and false otherwise.
func (r *Registry) IsEnabled(id string) bool {
	r.RLock()
	defer r.RUnlock()

	g, ok := r.gates[id]
	return ok && g.Enabled
}

// Set updates a registered Gate's Enabled attribute. If no Gate is registered with the
// provided ID an error is returned.
func (r *Registry) Set(id string, enabled bool) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.gates[id]; !ok {
		return fmt.Errorf("attempted to set non-existent gate %q", id)
	}

	r.gates[id].Enabled = enabled
	return nil
}

// Add a Gate to the Registry. An error is returned if a Gate with the same ID is already registered.
func (r *Registry) Add(g *Gate) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.gates[g.Id]; ok {
		return fmt.Errorf("attempted to add pre-existing gate %q", g.Id)
	}

	r.gates[g.Id] = g
	return nil
}

// List returns a slice of copies of the current state of all registered Gates.
func (r *Registry) List() []Gate {
	r.RLock()
	defer r.RUnlock()

	ret := make([]Gate, len(r.gates))
	i := 0
	for _, gate := range r.gates {
		ret[i] = *gate
		i++
	}

	return ret
}

// Frozen returns a read-only Gates implementation with a snapshot of the current
// state of the Registry.
func (r *Registry) Frozen() Gates {
	r.RLock()
	defer r.RUnlock()

	ret := make(map[string]Gate, len(r.gates))
	for _, gate := range r.gates {
		ret[gate.Id] = *gate
	}

	return &frozenRegistry{gates: ret}
}

type frozenRegistry struct {
	gates map[string]Gate
}

// IsEnabled returns true if a registered feature gate is enabled and false otherwise.
func (r frozenRegistry) IsEnabled(id string) bool {
	g, ok := r.gates[id]
	return ok && g.Enabled
}
