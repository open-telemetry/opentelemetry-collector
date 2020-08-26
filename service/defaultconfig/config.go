// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package defaultconfig

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
)

// CreateDefaultConfig creates default configuration for a given configuration.
// The default configuration is not created if the pipeline does not exist.
func CreateDefaultConfig(factories component.Factories, forConfig *configmodels.Config) *configmodels.Config {
	// the default config cannot be installed to the missing pipeline
	// because the service will fail to start as there is no default exporter.
	if forConfig == nil || forConfig.Service.Pipelines["traces"] == nil {
		return nil
	}
	otlpRec := factories.Receivers["otlp"].CreateDefaultConfig()
	batch := factories.Processors["batch"].CreateDefaultConfig()
	return &configmodels.Config{
		Receivers:  configmodels.Receivers{otlpRec.Name(): otlpRec},
		Processors: configmodels.Processors{batch.Name(): batch},
		Service: configmodels.Service{
			Pipelines: configmodels.Pipelines{
				string(configmodels.TracesDataType): {
					InputType:  configmodels.TracesDataType,
					Receivers:  []string{otlpRec.Name()},
					Processors: []string{batch.Name()},
				},
			},
		},
	}
}
