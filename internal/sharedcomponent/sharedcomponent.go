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
		components: map[K]*componentEntry[V]{},
	}
}

// Map keeps reference of all created instances for a given shared key such as a component configuration.
type Map[K comparable, V component.Component] struct {
	lock       sync.Mutex
	components map[K]*componentEntry[V]
}

// LoadOrStore looks up the already created instance if exists, otherwise creates a new instance
// and adds it to the map of references. It returns the wrapped shared component, the instance of
// the component and a possible error on creation.
func (m *Map[K, V]) LoadOrStore(key K, create func() (V, error)) (component.Component, V, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if c, ok := m.components[key]; ok {
		return c, c.component, nil
	}
	comp, err := create()
	if err != nil {
		return comp, comp, err
	}

	newComp := &componentEntry[V]{
		component: comp,
		removeFunc: func() {
			m.lock.Lock()
			defer m.lock.Unlock()
			delete(m.components, key)
		},
	}
	m.components[key] = newComp
	return newComp, comp, nil
}

type componentEntry[V component.Component] struct {
	component V

	startCounter atomic.Int32
	startErr     error
	startOnce    sync.Once
	stopOnce     sync.Once
	removeFunc   func()
}

// Start starts the underlying component if it never started before.
func (c *componentEntry[V]) Start(ctx context.Context, host component.Host) error {
	c.startCounter.Add(1)
	c.startOnce.Do(func() {
		c.startErr = c.component.Start(ctx, host)
	})
	return c.startErr
}

// Shutdown shuts down the underlying component.
func (c *componentEntry[V]) Shutdown(ctx context.Context) error {
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
