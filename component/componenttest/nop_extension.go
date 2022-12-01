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
	"go.opentelemetry.io/collector/extension"
)

// Deprecated: [v0.67.0] use extensiontest.NewNopCreateSettings.
func NewNopExtensionCreateSettings() extension.CreateSettings {
	return extension.CreateSettings{
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type nopExtensionConfig struct {
	config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// Deprecated: [v0.67.0] use extensiontest.NewNopFactory.
func NewNopExtensionFactory() extension.Factory {
	return extension.NewFactory(
		"nop",
		func() component.Config {
			return &nopExtensionConfig{
				ExtensionSettings: config.NewExtensionSettings(component.NewID("nop")),
			}
		},
		func(context.Context, component.ExtensionCreateSettings, component.Config) (component.Extension, error) {
			return nopExtensionInstance, nil
		},
		component.StabilityLevelStable)
}

var nopExtensionInstance = &nopExtension{}

// nopExtension stores consumed traces and metrics for testing purposes.
type nopExtension struct {
	nopComponent
}
