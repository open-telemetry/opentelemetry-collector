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

package component // import "go.opentelemetry.io/collector/component"

import (
	"go.opentelemetry.io/collector/confmap"
)

type Config interface {
	identifiable
	// Deprecated: [v0.65.0] use component.ValidateConfig.
	Validate() error

	privateConfig()
}

// ConfigValidator defines an optional interface for configurations to implement to do validation.
type ConfigValidator interface {
	// Validate the configuration and returns an error if invalid.
	Validate() error
}

// ValidateConfig validates a config, by doing this:
//   - Call Validate on the config itself if the config implements ConfigValidator.
func ValidateConfig(cfg Config) error {
	validator, ok := cfg.(ConfigValidator)
	if !ok {
		return nil
	}

	return validator.Validate()
}

// Type is the component type as it is used in the config.
type Type string

// DataType is a special Type that represents the data types supported by the collector. We currently support
// collecting metrics, traces and logs, this can expand in the future.
type DataType = Type

// Currently supported data types. Add new data types here when new types are supported in the future.
const (
	// DataTypeTraces is the data type tag for traces.
	DataTypeTraces DataType = "traces"

	// DataTypeMetrics is the data type tag for metrics.
	DataTypeMetrics DataType = "metrics"

	// DataTypeLogs is the data type tag for logs.
	DataTypeLogs DataType = "logs"
)

func unmarshal(componentSection *confmap.Conf, intoCfg interface{}) error {
	if cu, ok := intoCfg.(confmap.Unmarshaler); ok {
		return cu.Unmarshal(componentSection)
	}

	return componentSection.Unmarshal(intoCfg, confmap.WithErrorUnused())
}
