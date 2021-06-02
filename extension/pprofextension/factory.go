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

package pprofextension

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/extension/extensionhelper"
)

const (
	// The value of extension "type" in configuration.
	typeStr = "pprof"

	defaultEndpoint = "localhost:1777"
)

// NewFactory creates a factory for pprof extension.
func NewFactory() component.ExtensionFactory {
	return extensionhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		createExtension)
}

func createDefaultConfig() config.Extension {
	return &Config{
		ExtensionSettings: config.NewExtensionSettings(config.NewID(typeStr)),
		TCPAddr: confignet.TCPAddr{
			Endpoint: defaultEndpoint,
		},
	}
}

func createExtension(_ context.Context, set component.ExtensionCreateSettings, cfg config.Extension) (component.Extension, error) {
	config := cfg.(*Config)
	if config.TCPAddr.Endpoint == "" {
		return nil, errors.New("\"endpoint\" is required when using the \"pprof\" extension")
	}

	return newServer(*config, set.Logger), nil
}
