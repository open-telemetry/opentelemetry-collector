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

package span

import (
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

// Config is the configuration for the span processor.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// Rename specifies the components required to rename a span.
	// The `keys` field needs to be set for this processor to be properly
	// configured. Once there are multiple operations within span, this
	// requirement can be removed.
	Rename Rename `mapstructure:"rename"`
}

// Rename specifies the components of a span to use to rename a span.
type Rename struct {
	// Separator is the string used to concatenate various parts of the span name.
	// If no value is set, no separator is used between attribute values.
	Separator string `mapstructure:"separator"`
	// Keys represents the attribute keys to pull the values from to generate the new span name.
	// If not all attribute keys are present in the span, no rename will occur.
	// Note: The order in which these are specified is the order in which the new span name will
	// be built with the attribute values.
	// This field is required and cannot be empty.
	Keys []string `mapstructure:"keys"`
}
