// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package sharedcomponent exposes functionality for components
// to register against a shared key, such as a configuration object, in order to be reused across signal types.
// This is particularly useful when the component relies on a shared resource such as os.File or http.Server.
package sharedcomponent // import "go.opentelemetry.io/collector/internal/sharedcomponent"

import (
	"context"
	"slices"
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
func (m *Map[K, V]) LoadOrStore(key K, create func() (V, error), telemetrySettings *component.TelemetrySettings) (*Component[V], error) {
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
		telemetry: telemetrySettings,
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

	telemetry *component.TelemetrySettings

	hostWrapper *hostWrapper
}

// Unwrap returns the original component.
func (c *Component[V]) Unwrap() V {
	return c.component
}

// Start starts the underlying component if it never started before.
func (c *Component[V]) Start(ctx context.Context, host component.Host) error {
	if c.hostWrapper == nil {
		c.hostWrapper = &hostWrapper{
			host:           host,
			sources:        make([]componentstatus.StatusReporter, 0),
			previousEvents: make([]*componentstatus.StatusEvent, 0),
		}
	}

	statusReporter, isStatusReporter := host.(componentstatus.StatusReporter)
	if isStatusReporter {
		c.hostWrapper.addSource(statusReporter)
	}

	var err error
	c.startOnce.Do(func() {
		// It's important that status for a shared component is reported through its
		// telemetry settings to keep status in sync and avoid race conditions. This logic duplicates
		// and takes priority over the automated status reporting that happens in graph, making the
		// status reporting in graph a no-op.
		if c.hostWrapper != nil {
			c.hostWrapper.ReportStatus(componentstatus.NewStatusEvent(componentstatus.StatusStarting))
		}
		if err = c.component.Start(ctx, c.hostWrapper); err != nil {
			if c.hostWrapper != nil {
				c.hostWrapper.ReportStatus(componentstatus.NewPermanentErrorEvent(err))
			}
		}
	})

	return err
}

var _ component.Host = (*hostWrapper)(nil)
var _ componentstatus.StatusReporter = (*hostWrapper)(nil)

type hostWrapper struct {
	host           component.Host
	sources        []componentstatus.StatusReporter
	previousEvents []*componentstatus.StatusEvent
}

func (h *hostWrapper) GetFactory(kind component.Kind, componentType component.Type) component.Factory {
	return h.host.GetFactory(kind, componentType)
}

func (h *hostWrapper) GetExtensions() map[component.ID]component.Component {
	return h.host.GetExtensions()
}

func (h *hostWrapper) ReportStatus(e *componentstatus.StatusEvent) {
	if !slices.Contains(h.previousEvents, e) {
		h.previousEvents = append(h.previousEvents, e)
	}

	for _, s := range h.sources {
		s.ReportStatus(e)
	}
}

func (h *hostWrapper) addSource(s componentstatus.StatusReporter) {
	for _, e := range h.previousEvents {
		s.ReportStatus(e)
	}
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
			c.hostWrapper.ReportStatus(componentstatus.NewStatusEvent(componentstatus.StatusStopping))
		}
		err = c.component.Shutdown(ctx)
		if c.hostWrapper != nil {
			if err != nil {
				c.hostWrapper.ReportStatus(componentstatus.NewPermanentErrorEvent(err))
			} else {
				c.hostWrapper.ReportStatus(componentstatus.NewStatusEvent(componentstatus.StatusStopped))
			}
		}
		c.removeFunc()
	})
	return err
}
