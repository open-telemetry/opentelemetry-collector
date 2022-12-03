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

package featuregate // import "go.opentelemetry.io/collector/featuregate"

import (
	"fmt"
	"sync"
)

var reg = NewRegistry()

// GetRegistry returns the global Registry.
func GetRegistry() *Registry {
	return reg
}

type Registry struct {
	mu    sync.RWMutex
	gates map[string]Gate
}

// NewRegistry returns a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{gates: make(map[string]Gate)}
}

// RegistryOption allows to configure additional information about a Gate during registration.
type RegistryOption interface {
	apply(g *Gate)
}

type registerOption struct {
	applyFunc func(g *Gate)
}

func (ro registerOption) apply(g *Gate) {
	ro.applyFunc(g)
}

// WithRegisterDescription adds description for the Gate.
func WithRegisterDescription(description string) RegistryOption {
	return registerOption{
		applyFunc: func(g *Gate) {
			g.description = description
		},
	}
}

// WithRegisterReferenceURL adds an URL that has all the contextual information about the Gate.
func WithRegisterReferenceURL(url string) RegistryOption {
	return registerOption{
		applyFunc: func(g *Gate) {
			g.referenceURL = url
		},
	}
}

// WithRegisterRemovalVersion is used when the Gate is considered StageStable,
// to inform users that referencing the gate is no longer needed.
func WithRegisterRemovalVersion(version string) RegistryOption {
	return registerOption{
		applyFunc: func(g *Gate) {
			g.removalVersion = version
		},
	}
}

// Apply a configuration in the form of a map of Gate identifiers to boolean values.
// Sets only those values provided in the map, other gate values are not changed.
func (r *Registry) Apply(cfg map[string]bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, val := range cfg {
		g, ok := r.gates[id]
		if !ok {
			return fmt.Errorf("feature gate %s is unregistered", id)
		}
		if g.stage == StageStable {
			return fmt.Errorf("feature gate %s is stable, can not be modified", id)
		}
		g.enabled = val
		r.gates[g.id] = g
	}
	return nil
}

// IsEnabled returns true if a registered feature gate is enabled and false otherwise.
func (r *Registry) IsEnabled(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.gates[id]
	return ok && g.enabled
}

// MustRegisterID like RegisterID but panics if an invalid ID or gate options are provided.
func (r *Registry) MustRegisterID(id string, stage Stage, opts ...RegistryOption) {
	if err := r.RegisterID(id, stage, opts...); err != nil {
		panic(err)
	}
}

func (r *Registry) RegisterID(id string, stage Stage, opts ...RegistryOption) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.gates[id]; ok {
		return fmt.Errorf("attempted to add pre-existing gate %q", id)
	}
	g := Gate{
		id:    id,
		stage: stage,
	}
	for _, opt := range opts {
		opt.apply(&g)
	}
	switch g.stage {
	case StageAlpha:
		g.enabled = false
	case StageBeta, StageStable:
		g.enabled = true
	default:
		return fmt.Errorf("unknown stage value %q for gate %q", stage, id)
	}
	if g.stage == StageStable && g.removalVersion == "" {
		return fmt.Errorf("no removal version set for stable gate %q", id)
	}
	r.gates[id] = g
	return nil
}

// List returns a slice of copies of all registered Gates.
func (r *Registry) List() []Gate {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ret := make([]Gate, len(r.gates))
	i := 0
	for _, gate := range r.gates {
		ret[i] = gate
		i++
	}

	return ret
}
