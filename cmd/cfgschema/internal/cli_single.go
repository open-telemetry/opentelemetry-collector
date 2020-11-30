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

package internal

import (
	"fmt"
	"os"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/service/defaultcomponents"
)

// CreateSingleCfgSchemaFile creates a config schema yaml file for a single component
func CreateSingleCfgSchemaFile(componentType, componentName string, env Env) {
	cfg, err := getConfig(componentType, componentName)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	createSchemaFile(cfg, env)
}

func getConfig(componentType, componentName string) (configmodels.NamedEntity, error) {
	components, err := defaultcomponents.Components()
	if err != nil {
		return nil, err
	}
	t := configmodels.Type(componentName)
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
