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

package modifyattributes

import (
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

// Config defines configuration for Modify Attributes processor.
// This processor will handle all forms of modifications to
// attributes. It will logically replaces two previously committed
// processors
// 1. Add Attributes Processor
// That processor functionality is now covered by the ModifyValues
// field. It allows to set/update key/values in the attribute.
// 2. Attribute Key Processor
// The expected functionality is to duplicate a key/value pair under a new key.
// Being added to this Modify Attributes Processor is the ability to rename
// spans using values from attribute keys in the span.
//
// TODO(ccaraman) Document heavily that intermediary changes from sub-functions
//  of this processors can't be inputs to other sub-functions.
//  The current expectation is any changes to the attributes
// 	will use the span that 'enters' the processor. The set of changes to the span
// 	attributes are committed at the end of the processor.
// 	It is up to the configurer to ensure to not invoke copying/updating/setting
// 	the same attribute key/value by various sub-functions.
// 	What this means is one cannot expect to say modify a key value pair and then
// 	use that same key value pair as input for span rename.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// ModifyValues specifies the list of attributes to modify.
	ModifyKeyValues []ModifyKeyValue `mapstructure:"modify-key-values"`

	// SpanRename specifies the set of keys to for renaming the span.
	SpanRename SpanRename `mapstructure:"span-rename"`
}

// SourceValue specifies where the value should be retrieved from, either:
// 1. The configuration by setting a `RawValue`
// or
// 2. another attribute from the span.
// Only one of RawValue or SourceKey/KeepOriginal can be set.
type SourceValue struct {
	// Only one of RawValue or SourceKey/KeepOriginal can be set. If nothing is set,
	// the value is nil and it will be equivalent to

	// RawValue specifies the value to set.
	// To redact a key, set this to the empty string.
	RawValue interface{} `mapstructure:"raw-value"`

	// SourceKey specifies the attribute value to set.
	SourceKey string `mapstructure:"source-key"`

	// KeepOriginal specifies if the `SourceKey` attribute should be removed
	// from the span.
	// The default value is false.
	KeepOriginal bool `mapstructure:"keep-original"`
}

// ModifyKeyValue defines the key/value attribute pair to set/update.
type ModifyKeyValue struct {
	// Key specifies the attribute key to set/update the value to.
	// This key doesn't need to exist
	Key string `mapstructure:"key"`

	// Value specifies the value to set for the key. The value can either be set
	// from a value in the configuration or from another attribute in the span.
	Value SourceValue `mapstructure:"value"`

	// Overwrite specifies if this operation should overwrite a value, if the attribute
	// key value already exists.
	// The def
	Overwrite bool `mapstructure:"overwrite"`
}

// SpanRename defines the set of keys and separator to use for renaming a span.
type SpanRename struct {
	// Keys represents the attribute keys to pull the values from to generate the new span name.
	// If not all attribute keys are present in the span, no rename will occur.
	// Note: The order in which these are specified is the order in which the new span name will
	// be built with the attribute values.
	// This field is required and cannot be empty.
	Keys []string `mapstructure:"keys"`

	// Separator is the string used to concatenate various parts of the span name.
	// If no value is set, no separator is used between attribute values.
	Separator string `mapstructure:"separator"`
}
