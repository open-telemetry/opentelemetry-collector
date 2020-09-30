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

package spanprocessor

import (
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
)

// Config is the configuration for the span processor.
// Prior to any actions being applied, each span is compared against
// the include properties and then the exclude properties if they are specified.
// This determines if a span is to be processed or not.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	filterconfig.MatchConfig `mapstructure:",squash"`

	// Rename specifies the components required to re-name a span.
	// The `from_attributes` field needs to be set for this processor to be properly
	// configured.
	// Note: The field name is `Rename` to avoid collision with the Name() method
	// from configmodels.ProcessorSettings.NamedEntity
	Rename Name `mapstructure:"name"`
}

// Name specifies the attributes to use to re-name a span.
type Name struct {
	// Specifies transformations of span name to and from attributes.
	// First FromAttributes rules are applied, then ToAttributes are applied.
	// At least one of these 2 fields must be set.

	// FromAttributes represents the attribute keys to pull the values from to
	// generate the new span name. All attribute keys are required in the span
	// to re-name a span. If any attribute is missing from the span, no re-name
	// will occur.
	// Note: The new span name is constructed in order of the `from_attributes`
	// specified in the configuration. This field is required and cannot be empty.
	FromAttributes []string `mapstructure:"from_attributes"`

	// Separator is the string used to separate attributes values in the new
	// span name. If no value is set, no separator is used between attribute
	// values. Used with FromAttributes only.
	Separator string `mapstructure:"separator"`

	// ToAttributes specifies a configuration to extract attributes from span name.
	ToAttributes *ToAttributes `mapstructure:"to_attributes"`
}

type ToAttributes struct {
	// Rules is a list of rules to extract attribute values from span name. The values
	// in the span name are replaced by extracted attribute names. Each rule in the list
	// is a regex pattern string. Span name is checked against the regex. If it matches
	// then all named subexpressions of the regex are extracted as attributes
	// and are added to the span. Each subexpression name becomes an attribute name and
	// subexpression matched portion becomes the attribute value. The matched portion
	// in the span name is replaced by extracted attribute name. If the attributes
	// already exist in the span then they will be overwritten. The process is repeated
	// for all rules in the order they are specified. Each subsequent rule works on the
	// span name that is the output after processing the previous rule.
	Rules []string `mapstructure:"rules"`

	// BreakAfterMatch specifies if processing of rules should stop after the first
	// match. If it is false rule processing will continue to be performed over the
	// modified span name.
	BreakAfterMatch bool `mapstructure:"break_after_match"`
}
