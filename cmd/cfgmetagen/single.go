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

package main

import (
	"errors"
	"os"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/service/defaultcomponents"
)

// singleCfg creates a config metadata yaml file for a single component
func singleCfg() {
	componentType := os.Args[1]
	componentName := os.Args[2]

	cfg, err := createConfig(componentType, componentName)
	if err != nil {
		panic(err)
	}
	genMeta(cfg)
}

func createConfig(componentType string, componentName string) (configmodels.NamedEntity, error) {
	components, err := defaultcomponents.Components()
	if err != nil {
		return nil, err
	}
	t := configmodels.Type(componentName)
	switch componentType {
	case "receiver":
		c := components.Receivers[t]
		if c == nil {
			return nil, errors.New("unknown receiver name " + componentName)
		}
		return c.CreateDefaultConfig(), nil
	case "processor":
		c := components.Processors[t]
		if c == nil {
			return nil, errors.New("unknown processor name " + componentName)
		}
		return c.CreateDefaultConfig(), nil
	case "exporter":
		c := components.Exporters[t]
		if c == nil {
			return nil, errors.New("unknown exporter name " + componentName)
		}
		return c.CreateDefaultConfig(), nil
	case "extension":
		c := components.Extensions[t]
		if c == nil {
			return nil, errors.New("unknown extension name " + componentName)
		}
		return c.CreateDefaultConfig(), nil
	}
	return nil, errors.New("unknown component type " + componentType)
}
