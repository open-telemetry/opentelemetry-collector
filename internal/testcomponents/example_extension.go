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

package testcomponents

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenthelper"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/extension/extensionhelper"
)

// ExampleExtensionCfg is for testing purposes. We are defining an example config and factory
// for "exampleextension" extension type.
type ExampleExtensionCfg struct {
	config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting             string                   `mapstructure:"extra"`
	ExtraMapSetting          map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting         []string                 `mapstructure:"extra_list"`
}

const extType = "exampleextension"

// ExampleExtensionFactory is factory for ExampleExtensionCfg.
var ExampleExtensionFactory = extensionhelper.NewFactory(extType, createExtensionDefaultConfig, createExtension)

// CreateDefaultConfig creates the default configuration for the Extension.
func createExtensionDefaultConfig() config.Extension {
	return &ExampleExtensionCfg{
		ExtensionSettings: config.NewExtensionSettings(config.NewID(extType)),
		ExtraSetting:      "extra string setting",
		ExtraMapSetting:   nil,
		ExtraListSetting:  nil,
	}
}

// CreateExtension creates an Extension based on this config.
func createExtension(context.Context, component.ExtensionCreateSettings, config.Extension) (component.Extension, error) {
	return componenthelper.New(), nil
}
