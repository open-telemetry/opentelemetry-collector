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

package attributesprocessor

import (
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

// Config specifies the set of attributes to be inserted, updated, upserted and
// deleted and the properties to include/exclude a span from being processed.
// This processor handles all forms of modifications to attributes within a span.
// Prior to any actions being applied, each span is compared against
// the match properties if they are specified. This determines if a span
// is to be processed or not.
// The list of actions is applied in order specified in the configuration.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// Match specifies the set of properties of a span to match against and
	// if the span should be included or excluded from the processor.
	// This is an optional field. If not set, all spans are processed.
	Match MatchProperties `mapstructure:"match_properties"`

	// Actions specifies the list of attributes to act on.
	// The set of actions are {INSERT, UPDATE, UPSERT, DELETE}.
	// This is a required field.
	Actions []ActionKeyValue `mapstructure:"actions"`
}

// ActionKeyValue specifies the attribute key to act upon.
type ActionKeyValue struct {
	// Key specifies the attribute to act upon.
	// This is a required field.
	Key string `mapstructure:"key"`

	// Value specifies the value to populate for the key.
	// The type of the value is inferred from the configuration.
	Value interface{} `mapstructure:"value"`

	// FromAttribute specifies the attribute from the span to use to populate
	// the value. If the attribute doesn't exist, no action is performed.
	FromAttribute string `mapstructure:"from_attribute"`

	// Action specifies the type of action to perform.
	// The set of values are {INSERT, UPDATE, UPSERT, DELETE}.
	// Both lower case and upper case are supported.
	// INSERT - Inserts the key/value to spans when the key does not exist.
	//          No action is applied to spans where the key already exists.
	//          Either Value or FromAttribute must be set.
	// UPDATE - Updates an existing key with a value. No action is applied
	//          to spans where the key does not exist.
	//          Either Value or FromAttribute must be set.
	// UPSERT - Performs insert or update action depending on the span
	//          containing the key. The key/value is insert to spans
	//          that did not originally have the key. The key/value is updated
	//          for spans where the key already existed.
	//          Either Value or FromAttribute must be set.
	// DELETE - Deletes the attribute from the span. If the key doesn't exist,
	//          no action is performed.
	// This is a required field.
	Action Action `mapstructure:"action"`
}

// Action is the enum to capture the four types of actions to perform on an
// attribute.
type Action string

const (
	// INSERT adds the key/value to spans when the key does not exist.
	// No action is applied to spans where the key already exists.
	INSERT Action = "insert"

	// UPDATE updates an existing key with a value. No action is applied
	// to spans where the key does not exist.
	UPDATE Action = "update"

	// UPSERT performs the INSERT or UPDATE action. The key/value is
	// insert to spans that did not originally have the key. The key/value is
	// updated for spans where the key already existed.
	UPSERT Action = "upsert"

	// DELETE deletes the attribute from the span. If the key doesn't exist,
	//no action is performed.
	DELETE Action = "delete"
)

// MatchProperties specifies the set of properties in a span to match against
// and if the span should be included or excluded from the processor.
// At least one of services or attributes must be specified. It is supported
// to have both specified, but this requires all of the properties to match
// for the inclusion/exclusion to occur.
type MatchProperties struct {

	// Services specify the list of service names to match against.
	// A match occurs if the span service name is in this list.
	Services []string `mapstructure:"services"`

	// Attributes specifies the list of attributes to match against.
	// All of these attributes must be in the span and match exactly
	// for a match to count.
	Attributes []Attribute `mapstructure:"attributes"`

	// Include specifies if the spans that match the properties should be
	// included or excluded from the processor.
	// A `true` value specifies that spans only matching these properties should
	// be included in the processing by this processor.
	// A `false` value specifies all spans that do not match these properties
	// should be processed.
	// This field is required.
	Include bool `mapstructure:"include"`
}

// Attribute specifies the attribute key and optional value to match against.
type Attribute struct {
	// Key specifies the attribute key.
	Key string `mapstructure:"key"`

	// Values specifies the value to match against.
	// If it is not set, any value will match.
	Value interface{} `mapstructure:"value"`
}
