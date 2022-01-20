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
	"go.opentelemetry.io/collector/component/componenthelper"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/internal/internalinterface"
)

// NewNopExtensionCreateSettings returns a new nop settings for Create*Extension functions.
func NewNopExtensionCreateSettings() component.ExtensionCreateSettings {
	return component.ExtensionCreateSettings{
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type nopExtensionConfig struct {
	config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// nopExtensionFactory is factory for nopExtension.
type nopExtensionFactory struct {
	internalinterface.BaseInternal
}

var nopExtensionFactoryInstance = &nopExtensionFactory{}

// NewNopExtensionFactory returns a component.ExtensionFactory that constructs nop extensions.
func NewNopExtensionFactory() component.ExtensionFactory {
	return nopExtensionFactoryInstance
}

// Type gets the type of the Extension config created by this factory.
func (f *nopExtensionFactory) Type() config.Type {
	return "nop"
}

// CreateDefaultConfig creates the default configuration for the Extension.
func (f *nopExtensionFactory) CreateDefaultConfig() config.Extension {
	return &nopExtensionConfig{
		ExtensionSettings: config.NewExtensionSettings(config.NewComponentID("nop")),
	}
}

// CreateExtension implements component.ExtensionFactory interface.
func (f *nopExtensionFactory) CreateExtension(
	_ context.Context,
	_ component.ExtensionCreateSettings,
	_ config.Extension,
) (component.Extension, error) {
	return nopExtensionInstance, nil
}

var nopExtensionInstance = &nopExtension{}

// nopExtension stores consumed traces and metrics for testing purposes.
type nopExtension struct {
	componenthelper.StartFunc
	componenthelper.ShutdownFunc
}
