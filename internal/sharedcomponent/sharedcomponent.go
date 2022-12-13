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

// Package sharedcomponent exposes util functionality for receivers and exporters
// that need to share state between different signal types instances such as net.Listener or os.File.
package sharedcomponent // import "go.opentelemetry.io/collector/internal/sharedcomponent"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
)

// SharedComponents a map that keeps reference of all created instances for a given configuration,
// and ensures that the shared state is started and stopped only once.
type SharedComponents[K comparable, V component.Component] struct {
	comps map[K]*SharedComponent[V]
}

// NewSharedComponents returns a new empty SharedComponents.
func NewSharedComponents[K comparable, V component.Component]() *SharedComponents[K, V] {
	return &SharedComponents[K, V]{
		comps: make(map[K]*SharedComponent[V]),
	}
}

// GetOrAdd returns the already created instance if exists, otherwise creates a new instance
// and adds it to the map of references.
func (scs *SharedComponents[K, V]) GetOrAdd(key K, create func() (V, error)) (*SharedComponent[V], error) {
	if c, ok := scs.comps[key]; ok {
		return c, nil
	}
	comp, err := create()
	if err != nil {
		return nil, err
	}
	newComp := &SharedComponent[V]{
		component: comp,
		removeFunc: func() {
			delete(scs.comps, key)
		},
	}
	scs.comps[key] = newComp
	return newComp, nil
}

// SharedComponent ensures that the wrapped component is started and stopped only once.
// When stopped it is removed from the SharedComponents map.
type SharedComponent[V component.Component] struct {
	component V

	startOnce  sync.Once
	stopOnce   sync.Once
	removeFunc func()
}

// Unwrap returns the original component.
func (r *SharedComponent[V]) Unwrap() V {
	return r.component
}

// Start implements component.Component.
func (r *SharedComponent[V]) Start(ctx context.Context, host component.Host) error {
	var err error
	r.startOnce.Do(func() {
		err = r.component.Start(ctx, host)
	})
	return err
}

// Shutdown implements component.Component.
func (r *SharedComponent[V]) Shutdown(ctx context.Context) error {
	var err error
	r.stopOnce.Do(func() {
		err = r.component.Shutdown(ctx)
		r.removeFunc()
	})
	return err
}
