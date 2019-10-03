// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spanprocessor

import (
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

// Config is the configuration for the span processor.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// Rename specifies the components required to re-name a span.
	// The `from_attributes` field needs to be set for this processor to be properly
	// configured.
	// Note: The field name is `Rename` to avoid collision with the Name() method
	// from configmodels.ProcessorSettings.NamedEntity
	Rename Name `mapstructure:"name"`
}

// Name specifies the attributes to use to re-name a span.
type Name struct {
	// Separator is the string used to separate attributes values in the new
	// span name. If no value is set, no separator is used between attribute
	// values.
	Separator string `mapstructure:"separator"`
	// FromAttributes represents the attribute keys to pull the values from to
	// generate the new span name. All attribute keys are required in the span
	// to re-name a span. If any attribute is missing from the span, no re-name
	// will occur.
	// Note: The new span name is constructed in order of the `from_attributes`
	// specified in the configuration. This field is required and cannot be empty.
	FromAttributes []string `mapstructure:"from_attributes"`
}
