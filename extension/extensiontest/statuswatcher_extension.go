// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensiontest // import "go.opentelemetry.io/collector/extension/extensiontest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension"
)

// NewStatusWatcherExtensionCreateSettings returns a new nop settings for Create*Extension functions.
func NewStatusWatcherExtensionCreateSettings() extension.CreateSettings {
	return extension.CreateSettings{
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewStatusWatcherExtensionFactory returns a component.ExtensionFactory to construct a status watcher extension.
func NewStatusWatcherExtensionFactory(
	onStatusChanged func(source *component.InstanceID, event *component.StatusEvent),
) extension.Factory {
	return extension.NewFactory(
		component.MustNewType("statuswatcher"),
		func() component.Config {
			return &struct{}{}
		},
		func(context.Context, extension.CreateSettings, component.Config) (component.Component, error) {
			return &statusWatcherExtension{onStatusChanged: onStatusChanged}, nil
		},
		component.StabilityLevelStable)
}

// statusWatcherExtension receives status events reported via component status reporting for testing
// purposes.
type statusWatcherExtension struct {
	component.StartFunc
	component.ShutdownFunc
	onStatusChanged func(source *component.InstanceID, event *component.StatusEvent)
}

func (e statusWatcherExtension) ComponentStatusChanged(source *component.InstanceID, event *component.StatusEvent) {
	e.onStatusChanged(source, event)
}
