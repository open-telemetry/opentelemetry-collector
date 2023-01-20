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

	"go.uber.org/atomic"
)

var globalRegistry = NewRegistry()

// Deprecated: [v0.70.0] use GlobalRegistry.
var GetRegistry = GlobalRegistry

// GlobalRegistry returns the global Registry.
func GlobalRegistry() *Registry {
	return globalRegistry
}

type Registry struct {
	gates sync.Map
}

// NewRegistry returns a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// RegistryOption allows to configure additional information about a Gate during registration.
type RegistryOption interface {
	apply(g *Gate)
}

type registerOptionFunc func(g *Gate)

func (ro registerOptionFunc) apply(g *Gate) {
	ro(g)
}

// WithRegisterDescription adds description for the Gate.
func WithRegisterDescription(description string) RegistryOption {
	return registerOptionFunc(func(g *Gate) {
		g.description = description
	})
}

// WithRegisterReferenceURL adds an URL that has all the contextual information about the Gate.
func WithRegisterReferenceURL(url string) RegistryOption {
	return registerOptionFunc(func(g *Gate) {
		g.referenceURL = url
	})
}

// WithRegisterRemovalVersion is used when the Gate is considered StageStable,
// to inform users that referencing the gate is no longer needed.
func WithRegisterRemovalVersion(version string) RegistryOption {
	return registerOptionFunc(func(g *Gate) {
		g.removalVersion = version
	})
}

// Apply a configuration in the form of a map of Gate identifiers to boolean values.
// Sets only those values provided in the map, other gate values are not changed.
func (r *Registry) Apply(cfg map[string]bool) error {
	for id, val := range cfg {
		v, ok := r.gates.Load(id)
		if !ok {
			return fmt.Errorf("feature gate %s is unregistered", id)
		}
		g := v.(*Gate)
		if g.stage == StageStable {
			return fmt.Errorf("feature gate %s is stable, can not be modified", id)
		}
		g.enabled.Store(val)
	}
	return nil
}

// IsEnabled returns true if a registered feature gate is enabled and false otherwise.
func (r *Registry) IsEnabled(id string) bool {
	v, ok := r.gates.Load(id)
	if !ok {
		return false
	}
	g := v.(*Gate)
	return g.enabled.Load()
}

// MustRegisterID like RegisterID but panics if an invalid ID or gate options are provided.
func (r *Registry) MustRegisterID(id string, stage Stage, opts ...RegistryOption) {
	if err := r.RegisterID(id, stage, opts...); err != nil {
		panic(err)
	}
}

func (r *Registry) RegisterID(id string, stage Stage, opts ...RegistryOption) error {
	g := &Gate{
		id:    id,
		stage: stage,
	}
	for _, opt := range opts {
		opt.apply(g)
	}
	switch g.stage {
	case StageAlpha:
		g.enabled = atomic.NewBool(false)
	case StageBeta, StageStable:
		g.enabled = atomic.NewBool(true)
	default:
		return fmt.Errorf("unknown stage value %q for gate %q", stage, id)
	}
	if g.stage == StageStable && g.removalVersion == "" {
		return fmt.Errorf("no removal version set for stable gate %q", id)
	}
	if _, loaded := r.gates.LoadOrStore(id, g); loaded {
		return fmt.Errorf("attempted to add pre-existing gate %q", id)
	}
	return nil
}

// List returns a slice of copies of all registered Gates.
func (r *Registry) List() []Gate {
	var ret []Gate
	r.gates.Range(func(key, value any) bool {
		ret = append(ret, *(value.(*Gate)))
		return true
	})
	return ret
}
