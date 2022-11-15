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

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

// NewStatusWatcherExtensionCreateSettings returns a new nop settings for Create*Extension functions.
func NewStatusWatcherExtensionCreateSettings() component.ExtensionCreateSettings {
	return component.ExtensionCreateSettings{
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type statusWatcherExtensionConfig struct {
	config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// NewStatusWatcherExtensionFactory returns a component.ExtensionFactory that constructs nop extensions.
func NewStatusWatcherExtensionFactory(
	onStatusChanged func(source component.StatusSource, event *component.StatusEvent),
) component.ExtensionFactory {
	return component.NewExtensionFactory(
		"statuswatcher",
		func() component.ExtensionConfig {
			return &statusWatcherExtensionConfig{
				ExtensionSettings: config.NewExtensionSettings(component.NewID("statuswatcher")),
			}
		},
		func(context.Context, component.ExtensionCreateSettings, component.ExtensionConfig) (component.Extension, error) {
			return &statusWatcherExtension{onStatusChanged: onStatusChanged}, nil
		},
		component.StabilityLevelStable)
}

// statusWatcherExtension stores consumed traces and metrics for testing purposes.
type statusWatcherExtension struct {
	nopComponent
	onStatusChanged func(source component.StatusSource, event *component.StatusEvent)
}

func (e statusWatcherExtension) ComponentStatusChanged(source component.StatusSource, event *component.StatusEvent) {
	e.onStatusChanged(source, event)
}
