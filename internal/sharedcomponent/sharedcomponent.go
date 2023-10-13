// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
func (scs *SharedComponents[K, V]) GetOrAdd(key K, create func() (V, error), telemetrySettings *component.TelemetrySettings) (*SharedComponent[V], error) {
	if c, ok := scs.comps[key]; ok {
		// If we haven't already seen this telemetry settings, this shared component represents
		// another instance. Wrap ReportComponentStatus to report for all instances this shared
		// component represents.
		if _, ok := c.seenSettings[telemetrySettings]; !ok {
			c.seenSettings[telemetrySettings] = struct{}{}
			prev := c.telemetry.ReportComponentStatus
			c.telemetry.ReportComponentStatus = func(ev *component.StatusEvent) error {
				if err := telemetrySettings.ReportComponentStatus(ev); err != nil {
					return err
				}
				return prev(ev)
			}
		}
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
		telemetry: telemetrySettings,
		seenSettings: map[*component.TelemetrySettings]struct{}{
			telemetrySettings: {},
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

	telemetry    *component.TelemetrySettings
	seenSettings map[*component.TelemetrySettings]struct{}
}

// Unwrap returns the original component.
func (r *SharedComponent[V]) Unwrap() V {
	return r.component
}

// Start implements component.Component.
func (r *SharedComponent[V]) Start(ctx context.Context, host component.Host) error {
	var err error
	r.startOnce.Do(func() {
		// It's important that status for a sharedcomponent is reported through its
		// telemetrysettings to keep status in sync and avoid race conditions. This logic duplicates
		// and takes priority over the automated status reporting that happens in graph, making the
		// status reporting in graph a no-op.
		_ = r.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusStarting))
		if err = r.component.Start(ctx, host); err != nil {
			_ = r.telemetry.ReportComponentStatus(component.NewPermanentErrorEvent(err))
		}
	})
	return err
}

// Shutdown implements component.Component.
func (r *SharedComponent[V]) Shutdown(ctx context.Context) error {
	var err error
	r.stopOnce.Do(func() {
		// It's important that status for a sharedcomponent is reported through its
		// telemetrysettings to keep status in sync and avoid race conditions. This logic duplicates
		// and takes priority over the automated status reporting that happens in graph, making the
		// the status reporting in graph a no-op.
		_ = r.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusStopping))
		err = r.component.Shutdown(ctx)
		if err != nil {
			_ = r.telemetry.ReportComponentStatus(component.NewPermanentErrorEvent(err))
		} else {
			_ = r.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusStopped))
		}
		r.removeFunc()
	})
	return err
}
