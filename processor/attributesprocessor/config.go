// Copyright The OpenTelemetry Authors
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
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/internal/processor/filterspan"
)

// Config specifies the set of attributes to be inserted, updated, upserted and
// deleted and the properties to include/exclude a span from being processed.
// This processor handles all forms of modifications to attributes within a span.
// Prior to any actions being applied, each span is compared against
// the include properties and then the exclude properties if they are specified.
// This determines if a span is to be processed or not.
// The list of actions is applied in order specified in the configuration.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	filterspan.MatchConfig `mapstructure:",squash"`

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

	// A regex pattern  must be specified for the action EXTRACT.
	// It uses the attribute specified by `key' to extract values from
	// The target keys are inferred based on the names of the matcher groups
	// provided and the names will be inferred based on the values of the
	// matcher group.
	// Note: All subexpressions must have a name.
	// Note: The value type of the source key must be a string. If it isn't,
	// no extraction will occur.
	RegexPattern string `mapstructure:"pattern"`

	// FromAttribute specifies the attribute from the span to use to populate
	// the value. If the attribute doesn't exist, no action is performed.
	FromAttribute string `mapstructure:"from_attribute"`

	// Action specifies the type of action to perform.
	// The set of values are {INSERT, UPDATE, UPSERT, DELETE, HASH}.
	// Both lower case and upper case are supported.
	// INSERT -  Inserts the key/value to spans when the key does not exist.
	//           No action is applied to spans where the key already exists.
	//           Either Value or FromAttribute must be set.
	// UPDATE -  Updates an existing key with a value. No action is applied
	//           to spans where the key does not exist.
	//           Either Value or FromAttribute must be set.
	// UPSERT -  Performs insert or update action depending on the span
	//           containing the key. The key/value is insert to spans
	//           that did not originally have the key. The key/value is updated
	//           for spans where the key already existed.
	//           Either Value or FromAttribute must be set.
	// DELETE  - Deletes the attribute from the span. If the key doesn't exist,
	//           no action is performed.
	// HASH    - Calculates the SHA-1 hash of an existing value and overwrites the
	//           value with it's SHA-1 hash result.
	// EXTRACT - Extracts values using a regular expression rule from the input
	//           'key' to target keys specified in the 'rule'. If a target key
	//           already exists, it will be overridden.
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
	// no action is performed.
	DELETE Action = "delete"

	// HASH calculates the SHA-1 hash of an existing value and overwrites the
	// value with it's SHA-1 hash result.
	HASH Action = "hash"

	// EXTRACT extracts values using a regular expression rule from the input
	// 'key' to target keys specified in the 'rule'. If a target key already
	// exists, it will be overridden.
	EXTRACT Action = "extract"
)
