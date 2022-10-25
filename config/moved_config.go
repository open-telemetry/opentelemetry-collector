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

package config // import "go.opentelemetry.io/collector/config"

import (
	"go.opentelemetry.io/collector/service/telemetry"
)

// Service defines the configurable components of the service.
// Deprecated: [v0.52.0] Use service.ConfigService
type Service struct {
	// Telemetry is the configuration for collector's own telemetry.
	Telemetry telemetry.Config `mapstructure:"telemetry"`

	// Extensions are the ordered list of extensions configured for the service.
	Extensions []ComponentID `mapstructure:"extensions"`

	// Pipelines are the set of data pipelines configured for the service.
	Pipelines map[ComponentID]*Pipeline `mapstructure:"pipelines"`
}

// Pipeline defines a single pipeline.
// Deprecated: [v0.52.0] Use service.ConfigServicePipeline
type Pipeline struct {
	Receivers  []ComponentID `mapstructure:"receivers"`
	Processors []ComponentID `mapstructure:"processors"`
	Exporters  []ComponentID `mapstructure:"exporters"`
}

// Deprecated: [v0.52.0] will be removed soon.
type Pipelines = map[ComponentID]*Pipeline
