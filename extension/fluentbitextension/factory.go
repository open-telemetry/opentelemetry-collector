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

package fluentbitextension

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/extension/extensionhelper"
)

const (
	// The value of extension "type" in configuration.
	typeStr = "fluentbit"
)

// NewFactory creates a factory for FluentBit extension.
func NewFactory() component.ExtensionFactory {
	return extensionhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		createExtension)
}

func createDefaultConfig() configmodels.Extension {
	return &Config{
		ExtensionSettings: configmodels.ExtensionSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

func createExtension(_ context.Context, params component.ExtensionCreateParams, cfg configmodels.Extension) (component.ServiceExtension, error) {
	config := cfg.(*Config)
	return newProcessManager(config, params.Logger), nil
}
