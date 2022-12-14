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

package configunmarshaler // import "go.opentelemetry.io/collector/otelcol/internal/configunmarshaler"

import (
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

type Configs struct {
	cfgs map[component.ID]component.Config

	factories map[component.Type]component.Factory
}

func NewConfigs(factories map[component.Type]component.Factory) *Configs {
	return &Configs{factories: factories}
}

func (p *Configs) Unmarshal(conf *confmap.Conf) error {
	rawProcs := make(map[component.ID]map[string]interface{})
	if err := conf.Unmarshal(&rawProcs, confmap.WithErrorUnused()); err != nil {
		return err
	}

	// Prepare resulting map.
	p.cfgs = make(map[component.ID]component.Config)
	// Iterate over processors and create a config for each.
	for id, value := range rawProcs {
		// Find processor factory based on "type" that we read from config source.
		factory := p.factories[id.Type()]
		if factory == nil {
			return errorUnknownType(id, reflect.ValueOf(p.factories).MapKeys())
		}

		// Create the default config for this processor.
		processorCfg := factory.CreateDefaultConfig()

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := component.UnmarshalConfig(confmap.NewFromStringMap(value), processorCfg); err != nil {
			return errorUnmarshalError(id, err)
		}

		p.cfgs[id] = processorCfg
	}

	return nil
}

func (p *Configs) Configs() map[component.ID]component.Config {
	return p.cfgs
}

func errorUnknownType(id component.ID, factories []reflect.Value) error {
	return fmt.Errorf("unknown type: %q for id: %q (valid values: %v)", id.Type(), id, factories)
}

func errorUnmarshalError(id component.ID, err error) error {
	return fmt.Errorf("error reading configuration for %q: %w", id, err)
}
