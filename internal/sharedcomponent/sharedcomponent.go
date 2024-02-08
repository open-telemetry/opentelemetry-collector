// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package sharedcomponent exposes functionality for components
// to register against a shared key, such as a configuration object, in order to be reused across signal types.
// This is particularly useful when the component relies on a shared resource such as os.File or http.Server.
package sharedcomponent // import "go.opentelemetry.io/collector/internal/sharedcomponent"

import (
	"context"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/collector/component"
)

func NewMap[K comparable, V component.Component]() *Map[K, V] {
	return &Map[K, V]{
		components: map[K]*Component[V]{},
	}
}

// Map keeps reference of all created instances for a given shared key such as a component configuration.
type Map[K comparable, V component.Component] struct {
	lock       sync.Mutex
	components map[K]*Component[V]
}

// LoadOrStore returns the already created instance if exists, otherwise creates a new instance
// and adds it to the map of references.
func (m *Map[K, V]) LoadOrStore(key K, create func() (V, error)) (*Component[V], error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if c, ok := m.components[key]; ok {
		return c, nil
	}
	comp, err := create()
	if err != nil {
		return nil, err
	}

	newComp := &Component[V]{
		component: comp,
		removeFunc: func() {
			m.lock.Lock()
			defer m.lock.Unlock()
			delete(m.components, key)
		},
	}
	m.components[key] = newComp
	return newComp, nil
}

// Component ensures that the wrapped component is started and stopped only once.
// When stopped it is removed from the Map.
type Component[V component.Component] struct {
	component V

	startCounter atomic.Int32
	startErr     error
	startOnce    sync.Once
	stopOnce     sync.Once
	removeFunc   func()
}

// Unwrap returns the original component.
func (c *Component[V]) Unwrap() V {
	return c.component
}

// Start starts the underlying component if it never started before.
func (c *Component[V]) Start(ctx context.Context, host component.Host) error {
	c.startCounter.Add(1)
	c.startOnce.Do(func() {
		c.startErr = c.component.Start(ctx, host)
	})
	return c.startErr
}

// Shutdown shuts down the underlying component.
func (c *Component[V]) Shutdown(ctx context.Context) error {
	if c.startCounter.Add(-1) <= 0 {
		var err error
		c.stopOnce.Do(func() {
			c.removeFunc()
			err = c.component.Shutdown(ctx)
		})
		return err
	}

	return nil
}
