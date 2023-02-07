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
	"sort"
	"sync"

	"go.uber.org/atomic"
)

var globalRegistry = NewRegistry()

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

// Deprecated: [v0.71.0] use RegisterOption.
type RegistryOption = RegisterOption

// RegisterOption allows to configure additional information about a Gate during registration.
type RegisterOption interface {
	apply(g *Gate)
}

type registerOptionFunc func(g *Gate)

func (ro registerOptionFunc) apply(g *Gate) {
	ro(g)
}

// WithRegisterDescription adds description for the Gate.
func WithRegisterDescription(description string) RegisterOption {
	return registerOptionFunc(func(g *Gate) {
		g.description = description
	})
}

// WithRegisterReferenceURL adds a URL that has all the contextual information about the Gate.
func WithRegisterReferenceURL(url string) RegisterOption {
	return registerOptionFunc(func(g *Gate) {
		g.referenceURL = url
	})
}

// WithRegisterRemovalVersion is used when the Gate is considered StageStable,
// to inform users that referencing the gate is no longer needed.
func WithRegisterRemovalVersion(version string) RegisterOption {
	return registerOptionFunc(func(g *Gate) {
		g.removalVersion = version
	})
}

// Deprecated: [v0.71.0] use Set.
func (r *Registry) Apply(cfg map[string]bool) error {
	for id, enabled := range cfg {
		if err := r.Set(id, enabled); err != nil {
			return err
		}
	}
	return nil
}

// Deprecated: [v0.71.0] check the enable status on the returned Gate from Register or MustRegister.
func (r *Registry) IsEnabled(id string) bool {
	v, ok := r.gates.Load(id)
	if !ok {
		return false
	}
	g := v.(*Gate)
	return g.IsEnabled()
}

// MustRegister like Register but panics if an invalid ID or gate options are provided.
func (r *Registry) MustRegister(id string, stage Stage, opts ...RegisterOption) *Gate {
	g, err := r.Register(id, stage, opts...)
	if err != nil {
		panic(err)
	}
	return g
}

// Register a Gate and return it. The returned Gate can be used to check if is enabled or not.
func (r *Registry) Register(id string, stage Stage, opts ...RegisterOption) (*Gate, error) {
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
		return nil, fmt.Errorf("unknown stage value %q for gate %q", stage, id)
	}
	if g.stage == StageStable && g.removalVersion == "" {
		return nil, fmt.Errorf("no removal version set for stable gate %q", id)
	}
	if _, loaded := r.gates.LoadOrStore(id, g); loaded {
		return nil, fmt.Errorf("attempted to add pre-existing gate %q", id)
	}
	return g, nil
}

// Deprecated: [v0.71.0] use MustRegister.
func (r *Registry) MustRegisterID(id string, stage Stage, opts ...RegisterOption) {
	r.MustRegister(id, stage, opts...)
}

// Deprecated: [v0.71.0] use Register.
func (r *Registry) RegisterID(id string, stage Stage, opts ...RegisterOption) error {
	_, err := r.Register(id, stage, opts...)
	return err
}

// Set the enabled valued for a Gate identified by the given id.
func (r *Registry) Set(id string, enabled bool) error {
	v, ok := r.gates.Load(id)
	if !ok {
		return fmt.Errorf("no such feature gate -%v", id)
	}
	g := v.(*Gate)
	if g.stage == StageStable {
		return fmt.Errorf("feature gate %s is stable, can not be modified", id)
	}
	g.enabled.Store(enabled)
	return nil
}

// VisitAll visits all the gates in lexicographical order, calling fn for each.
func (r *Registry) VisitAll(fn func(*Gate)) {
	var gates []*Gate
	r.gates.Range(func(key, value any) bool {
		gates = append(gates, value.(*Gate))
		return true
	})
	sort.Slice(gates, func(i, j int) bool {
		return gates[i].ID() < gates[j].ID()
	})
	for i := range gates {
		fn(gates[i])
	}
}

// Deprecated: [v0.71.0] use VisitAll.
func (r *Registry) List() []Gate {
	var ret []Gate
	r.VisitAll(func(gate *Gate) {
		ret = append(ret, *gate)
	})
	return ret
}
