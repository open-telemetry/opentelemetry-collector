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
)

// processorsKeyName is the configuration key name for processors section.
const processorsKeyName = "processors"

type Processors struct {
	procs map[component.ID]component.ProcessorConfig

	factories map[component.Type]component.ProcessorFactory
}

func NewProcessors(factories map[component.Type]component.ProcessorFactory) *Processors {
	return &Processors{factories: factories}
}

func (p *Processors) Unmarshal(conf *confmap.Conf) error {
	rawProcs := make(map[component.ID]map[string]interface{})
	if err := conf.Unmarshal(&rawProcs, confmap.WithErrorUnused()); err != nil {
		return err
	}

	// Prepare resulting map.
	p.procs = make(map[component.ID]component.ProcessorConfig)
	// Iterate over processors and create a config for each.
	for id, value := range rawProcs {
		// Find processor factory based on "type" that we read from config source.
		factory := p.factories[id.Type()]
		if factory == nil {
			return errorUnknownType(processorsKeyName, id, reflect.ValueOf(p.factories).MapKeys())
		}

		// Create the default config for this processor.
		processorCfg := factory.CreateDefaultConfig()
		processorCfg.SetIDName(id.Name())

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := component.UnmarshalProcessorConfig(confmap.NewFromStringMap(value), processorCfg); err != nil {
			return errorUnmarshalError(processorsKeyName, id, err)
		}

		p.procs[id] = processorCfg
	}

	return nil
}

func (p *Processors) GetProcessors() map[component.ID]component.ProcessorConfig {
	return p.procs
}
