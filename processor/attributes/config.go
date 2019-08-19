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

package attributes

import "github.com/open-telemetry/opentelemetry-service/config/configmodels"

// Config specifies the set of attributes to be inserted, updated, upserted and deleted.
// This processor handles all forms of modifications to attributes within a span.
// The list of actions is applied in order specified in the configuration.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// Actions specifies the list of attributes to act on.
	// The set of actions are {INSERT, UPDATE, UPSERT, DELETE}.
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
