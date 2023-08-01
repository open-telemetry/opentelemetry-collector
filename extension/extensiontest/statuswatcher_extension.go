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

// NewStatusWatcherExtensionFactory returns a component.ExtensionFactory that constructs nop extensions.
func NewStatusWatcherExtensionFactory(
	onStatusChanged func(source component.StatusSource, event *component.StatusEvent),
) extension.Factory {
	return extension.NewFactory(
		"statuswatcher",
		func() component.Config {
			return &struct{}{}
		},
		func(context.Context, extension.CreateSettings, component.Config) (component.Component, error) {
			return &statusWatcherExtension{onStatusChanged: onStatusChanged}, nil
		},
		component.StabilityLevelStable)
}

// statusWatcherExtension stores consumed traces and metrics for testing purposes.
type statusWatcherExtension struct {
	component.StartFunc
	component.ShutdownFunc
	onStatusChanged func(source component.StatusSource, event *component.StatusEvent)
}

func (e statusWatcherExtension) ComponentStatusChanged(source component.StatusSource, event *component.StatusEvent) {
	e.onStatusChanged(source, event)
}
