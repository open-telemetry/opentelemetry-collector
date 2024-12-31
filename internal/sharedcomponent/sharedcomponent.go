// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package sharedcomponent exposes functionality for components
// to register against a shared key, such as a configuration object, in order to be reused across signal types.
// This is particularly useful when the component relies on a shared resource such as os.File or http.Server.
package sharedcomponent // import "go.opentelemetry.io/collector/internal/sharedcomponent"

import (
	"container/ring"
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
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

	startOnce  sync.Once
	stopOnce   sync.Once
	removeFunc func()

	hostWrapper *hostWrapper
}

// Unwrap returns the original component.
func (c *Component[V]) Unwrap() V {
	return c.component
}

// Start starts the underlying component if it never started before.
func (c *Component[V]) Start(ctx context.Context, host component.Host) error {
	if c.hostWrapper == nil {
		var err error
		c.startOnce.Do(func() {
			c.hostWrapper = &hostWrapper{
				host:           host,
				sources:        make([]componentstatus.Reporter, 0),
				previousEvents: ring.New(5),
			}
			statusReporter, isStatusReporter := host.(componentstatus.Reporter)
			if isStatusReporter {
				c.hostWrapper.addSource(statusReporter)
			}

			// It's important that status for a shared component is reported through its
			// telemetry settings to keep status in sync and avoid race conditions. This logic duplicates
			// and takes priority over the automated status reporting that happens in graph, making the
			// status reporting in graph a no-op.
			c.hostWrapper.Report(componentstatus.NewEvent(componentstatus.StatusStarting))
			if err = c.component.Start(ctx, c.hostWrapper); err != nil {
				c.hostWrapper.Report(componentstatus.NewPermanentErrorEvent(err))
			}
		})
		return err
	}
	statusReporter, isStatusReporter := host.(componentstatus.Reporter)
	if isStatusReporter {
		c.hostWrapper.addSource(statusReporter)
	}
	return nil
}

var (
	_ component.Host           = (*hostWrapper)(nil)
	_ componentstatus.Reporter = (*hostWrapper)(nil)
)

type hostWrapper struct {
	host           component.Host
	sources        []componentstatus.Reporter
	previousEvents *ring.Ring
	lock           sync.Mutex
}

func (h *hostWrapper) GetExtensions() map[component.ID]component.Component {
	return h.host.GetExtensions()
}

func (h *hostWrapper) Report(e *componentstatus.Event) {
	// Only remember an event if it will be emitted and it has not been sent already.
	h.lock.Lock()
	defer h.lock.Unlock()
	if len(h.sources) > 0 {
		h.previousEvents.Value = e
		h.previousEvents = h.previousEvents.Next()
	}
	for _, s := range h.sources {
		s.Report(e)
	}
}

func (h *hostWrapper) addSource(s componentstatus.Reporter) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.previousEvents.Do(func(a any) {
		if e, ok := a.(*componentstatus.Event); ok {
			s.Report(e)
		}
	})
	h.sources = append(h.sources, s)
}

// Shutdown shuts down the underlying component.
func (c *Component[V]) Shutdown(ctx context.Context) error {
	var err error
	c.stopOnce.Do(func() {
		// It's important that status for a shared component is reported through its
		// telemetry settings to keep status in sync and avoid race conditions. This logic duplicates
		// and takes priority over the automated status reporting that happens in graph, making the
		// status reporting in graph a no-op.
		if c.hostWrapper != nil {
			c.hostWrapper.Report(componentstatus.NewEvent(componentstatus.StatusStopping))
		}
		err = c.component.Shutdown(ctx)
		if c.hostWrapper != nil {
			if err != nil {
				c.hostWrapper.Report(componentstatus.NewPermanentErrorEvent(err))
			} else {
				c.hostWrapper.Report(componentstatus.NewEvent(componentstatus.StatusStopped))
			}
		}
		c.removeFunc()
	})
	return err
}
