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
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter"
)

// exportersKeyName is the configuration key name for exporters section.
const exportersKeyName = "exporters"

type Exporters struct {
	exps map[component.ID]component.Config

	factories map[component.Type]exporter.Factory
}

func NewExporters(factories map[component.Type]exporter.Factory) *Exporters {
	return &Exporters{factories: factories}
}

func (e *Exporters) Unmarshal(conf *confmap.Conf) error {
	rawExps := make(map[component.ID]map[string]interface{})
	if err := conf.Unmarshal(&rawExps, confmap.WithErrorUnused()); err != nil {
		return err
	}

	// Prepare resulting map.
	e.exps = make(map[component.ID]component.Config)

	// Iterate over Exporters and create a config for each.
	for id, value := range rawExps {
		// Find exporter factory based on "type" that we read from config source.
		factory := e.factories[id.Type()]
		if factory == nil {
			return errorUnknownType(exportersKeyName, id, reflect.ValueOf(e.factories).MapKeys())
		}

		// Create the default config for this exporter.
		exporterCfg := factory.CreateDefaultConfig()
		exporterCfg.SetIDName(id.Name()) //nolint:staticcheck

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := component.UnmarshalConfig(confmap.NewFromStringMap(value), exporterCfg); err != nil {
			return errorUnmarshalError(exportersKeyName, id, err)
		}

		e.exps[id] = exporterCfg
	}

	return nil
}

func (e *Exporters) GetExporters() map[component.ID]component.Config {
	return e.exps
}
