// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/receiver/internal"

import "go.opentelemetry.io/collector/component"

// Settings configures Receiver creators.
type Settings struct {
	// ID returns the ID of the component that will be created.
	ID component.ID

	component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes.
	BuildInfo component.BuildInfo

	receiverFactoryFunc func(componentType component.Type) Factory
}

func NewSettings(id component.ID, telemetrySettings component.TelemetrySettings, buildInfo component.BuildInfo, receiverFactoryFunc func(componentType component.Type) Factory) Settings {
	return Settings{
		ID:                  id,
		TelemetrySettings:   telemetrySettings,
		BuildInfo:           buildInfo,
		receiverFactoryFunc: receiverFactoryFunc,
	}
}
