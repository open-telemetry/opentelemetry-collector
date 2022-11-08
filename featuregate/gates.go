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

// Stage represents the Gate's lifecycle and what is the expected state of it.
type Stage int8

const (
	// Alpha is used when creating a new feature and the Gate must be explicitly enabled
	// by the operator.
	//
	// The Gate will be disabled by default.
	Alpha Stage = iota
	// Beta is used when the feature flag is well tested and is enabled by default,
	// but can be disabled by a Gate.
	//
	// The Gate will be enabled by default.
	Beta
	// Stable is used when feature is permanently enabled and can not be disabled by a Gate.
	// This value is used to provide feedback to the user that the gate will be removed in the next version.
	//
	// The Gate will be enabled by default and will return an error if modified.
	Stable
)

func (s Stage) String() string {
	switch s {
	case Alpha:
		return "Alpha"
	case Beta:
		return "Beta"
	case Stable:
		return "Stable"
	}
	return "unknown"
}

// Gate is an immutable object that is owned by the `Registry`
// to represents an individual feature that may be enabled or disabled based
// on the lifecycle state of the feature and CLI flags specified by the user.
type Gate struct {
	// Deprecated: [v0.64.0] Use GetID() instead to read,
	//				use `Registry.RegisterID` to set value.
	ID string
	// Deprecated: [v0.64.0] use GetDescription to read,
	// 				use `WithRegisterDescription` to set using `Registry.RegisterID`.
	Description string
	// Deprecated: [v0.64.0] use `IsEnabled(id)` to read,
	//              use `Registry.Apply` to set.
	Enabled        bool
	stage          Stage
	referenceURL   string
	removalVersion string
}

// RegistryOption allows for configuration additional information
// about a gate that can be exposed throughout the application
type RegistryOption func(g *Gate)

// WithRegisterDescription adds the description to the provided `Gate“.
func WithRegisterDescription(description string) RegistryOption {
	return func(g *Gate) {
		g.Description = description
	}
}

// WithRegisterReferenceURL adds an URL that has
// all the contextual information about the `Gate`.
func WithRegisterReferenceURL(url string) RegistryOption {
	return func(g *Gate) {
		g.referenceURL = url
	}
}

// WithRegisterRemovalVersion is used when the `Gate` is considered `Stable`,
// to inform users that referencing the gate is no longer needed.
func WithRegisterRemovalVersion(version string) RegistryOption {
	return func(g *Gate) {
		g.removalVersion = version
	}
}

func (g *Gate) GetID() string {
	return g.ID
}

func (g *Gate) IsEnabled() bool {
	return g.Enabled
}

func (g *Gate) GetDescription() string {
	return g.Description
}

func (g *Gate) Stage() Stage {
	return g.stage
}

func (g *Gate) ReferenceURL() string {
	return g.referenceURL
}

func (g *Gate) RemovalVersion() string {
	return g.removalVersion
}

var reg = NewRegistry()

// GetRegistry returns the global Registry.
func GetRegistry() *Registry {
	return reg
}

// NewRegistry returns a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{gates: make(map[string]Gate)}
}

type Registry struct {
	mu    sync.RWMutex
	gates map[string]Gate
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
		if g.stage == Stable {
			return fmt.Errorf("feature gate %s is stable, can not be modified", id)
		}
		g.Enabled = val
		r.gates[g.ID] = g
	}
	return nil
}

// IsEnabled returns true if a registered feature gate is enabled and false otherwise.
func (r *Registry) IsEnabled(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.gates[id]
	return ok && g.Enabled
}

// MustRegister like Register but panics if a Gate with the same ID is already registered.
// Deprecated: [v0.64.0] Use MustRegisterID instead.
func (r *Registry) MustRegister(g Gate) {
	if err := r.Register(g); err != nil {
		panic(err)
	}
}

// Register registers a Gate. May only be called in an init() function.
// Deprecated: [v0.64.0] Use RegisterID instead.
func (r *Registry) Register(g Gate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.gates[g.ID]; ok {
		return fmt.Errorf("attempted to add pre-existing gate %q", g.ID)
	}
	r.gates[g.ID] = g
	return nil
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
		ID:    id,
		stage: stage,
	}
	for _, opt := range opts {
		opt(&g)
	}
	switch g.stage {
	case Alpha:
		g.Enabled = false
	case Beta, Stable:
		g.Enabled = true
	default:
		return fmt.Errorf("unknown stage value %q for gate %q", stage, id)
	}
	if g.stage == Stable && g.removalVersion == "" {
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
