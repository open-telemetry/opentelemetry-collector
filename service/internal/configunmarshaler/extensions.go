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

package configunmarshaler // import "go.opentelemetry.io/collector/service/internal/configunmarshaler"

import (
	"reflect"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/confmap"
)

// extensionsKeyName is the configuration key name for extensions section.
const extensionsKeyName = "extensions"

type Extensions struct {
	exts map[config.ComponentID]config.Extension

	factories map[config.Type]component.ExtensionFactory
}

func NewExtensions(factories map[config.Type]component.ExtensionFactory) *Extensions {
	return &Extensions{factories: factories}
}

func (e *Extensions) Unmarshal(conf *confmap.Conf) error {
	rawExts := make(map[config.ComponentID]map[string]interface{})
	if err := conf.Unmarshal(&rawExts, confmap.WithErrorUnused()); err != nil {
		return err
	}

	// Prepare resulting map.
	e.exts = make(map[config.ComponentID]config.Extension)

	// Iterate over extensions and create a config for each.
	for id, value := range rawExts {
		// Find extension factory based on "type" that we read from config source.
		factory, ok := e.factories[id.Type()]
		if !ok {
			return errorUnknownType(extensionsKeyName, id, reflect.ValueOf(e.factories).MapKeys())
		}

		// Create the default config for this extension.
		extensionCfg := factory.CreateDefaultConfig()
		extensionCfg.SetIDName(id.Name())

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := config.UnmarshalExtension(confmap.NewFromStringMap(value), extensionCfg); err != nil {
			return errorUnmarshalError(extensionsKeyName, id, err)
		}

		e.exts[id] = extensionCfg
	}

	return nil
}

func (e *Extensions) GetExtensions() map[config.ComponentID]config.Extension {
	return e.exts
}
