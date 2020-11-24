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
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/service/defaultcomponents"
)

// createAllCfgSchemas creates config yaml schema files for all registered components
func createAllCfgSchemas(env env) {
	cfgs := getAllConfigs()
	for _, cfg := range cfgs {
		createSchema(cfg, env)
	}
}

func getAllConfigs() []configmodels.NamedEntity {
	components, err := defaultcomponents.Components()
	if err != nil {
		panic(err)
	}
	var cfgs []configmodels.NamedEntity
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
