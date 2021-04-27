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

package configschema

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

// GetAllConfigs accepts a Factories struct, creates and return the default
// configs for all of its components.
func GetAllConfigs(components component.Factories) []interface{} {
	var cfgs []interface{}
	for _, f := range components.Receivers {
		cfgs = append(cfgs, f.CreateDefaultConfig())
	}
	for _, f := range components.Extensions {
		cfgs = append(cfgs, f.CreateDefaultConfig())
	}
	for _, f := range components.Processors {
		cfgs = append(cfgs, f.CreateDefaultConfig())
	}
	for _, f := range components.Exporters {
		cfgs = append(cfgs, f.CreateDefaultConfig())
	}
	return cfgs
}

// GetConfig accepts a Factories struct, creates and return the default config
// for the component specified by the passed-in componentType and componentName.
func GetConfig(components component.Factories, componentType, componentName string) (interface{}, error) {
	t := config.Type(componentName)
	switch componentType {
	case "receiver":
		c := components.Receivers[t]
		if c == nil {
			return nil, fmt.Errorf("unknown receiver name %q", componentName)
		}
		return c.CreateDefaultConfig(), nil
	case "processor":
		c := components.Processors[t]
		if c == nil {
			return nil, fmt.Errorf("unknown processor name %q", componentName)
		}
		return c.CreateDefaultConfig(), nil
	case "exporter":
		c := components.Exporters[t]
		if c == nil {
			return nil, fmt.Errorf("unknown exporter name %q", componentName)
		}
		return c.CreateDefaultConfig(), nil
	case "extension":
		c := components.Extensions[t]
		if c == nil {
			return nil, fmt.Errorf("unknown extension name %q", componentName)
		}
		return c.CreateDefaultConfig(), nil
	}
	return nil, fmt.Errorf("unknown component type %q", componentType)
}
