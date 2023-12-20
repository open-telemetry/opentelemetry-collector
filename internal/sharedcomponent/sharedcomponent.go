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

// Map keeps reference of all created instances for a given shared key such as a component configuration.
type Map[K comparable, V component.Component] map[K]*SharedComponent[V]

// LoadOrStore returns the already created instance if exists, otherwise creates a new instance
// and adds it to the map of references.
func (scs Map[K, V]) LoadOrStore(key K, create func() (V, error), telemetrySettings *component.TelemetrySettings) (*SharedComponent[V], error) {
	if c, ok := scs[key]; ok {
		// If we haven't already seen this telemetry settings, this shared component represents
		// another instance. Wrap ReportComponentStatus to report for all instances this shared
		// component represents.
		if _, ok := c.seenSettings[telemetrySettings]; !ok {
			c.seenSettings[telemetrySettings] = struct{}{}
			prev := c.telemetry.ReportComponentStatus
			c.telemetry.ReportComponentStatus = func(ev *component.StatusEvent) error {
				err := telemetrySettings.ReportComponentStatus(ev)
				if prevErr := prev(ev); prevErr != nil {
					err = prevErr
				}
				return err
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
			delete(scs, key)
		},
		telemetry: telemetrySettings,
		seenSettings: map[*component.TelemetrySettings]struct{}{
			telemetrySettings: {},
		},
		activeCount: &atomic.Int32{},
	}
	scs[key] = newComp
	return newComp, nil
}

// SharedComponent ensures that the wrapped component is started and stopped only once.
// When stopped it is removed from the Map.
type SharedComponent[V component.Component] struct {
	component V

	activeCount *atomic.Int32 // a counter keeping track of the number of active uses of the component
	startOnce   sync.Once
	stopOnce    sync.Once
	removeFunc  func()

	telemetry    *component.TelemetrySettings
	seenSettings map[*component.TelemetrySettings]struct{}
}

// Unwrap returns the original component.
func (r *SharedComponent[V]) Unwrap() V {
	return r.component
}

// Start starts the underlying component if it never started before. Each call to Start is counted as an active usage.
// Shutdown will shut down the underlying component if called as many times as Start is called.
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
	r.activeCount.Add(1)
	return err
}

// Shutdown shuts down the underlying component if all known usages, measured by the number of times
// Start was called, are accounted for.
func (r *SharedComponent[V]) Shutdown(ctx context.Context) error {
	if r.activeCount.Add(-1) > 0 {
		return nil
	}

	var err error
	r.stopOnce.Do(func() {
		// It's important that status for a sharedcomponent is reported through its
		// telemetrysettings to keep status in sync and avoid race conditions. This logic duplicates
		// and takes priority over the automated status reporting that happens in graph, making the
		// status reporting in graph a no-op.
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
