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

	// CopyKeyValues specifies the list of keys whose corresponding values
	// are to be copied to a new key.
	CopyKeyValues []CopyKeyValue `mapstructure:"copy-key-values"`

	// ModifyValues specifies the list of attributes based on keys to modify
	// the value.
	ModifyValues []ModifyKeyValue `mapstructure:"modify-key-values"`

	// SpanRename specifies the set of keys
	SpanRename SpanRename `mapstructure:"span-rename"`
}

// CopyKeyValue defines the properties of the key/value pair to be copied to
// a new key.
// The value is not modified.
type CopyKeyValue struct {
	// SourceKey specifies the attribute key to get the value.
	// This field is required.
	SourceKey string `mapstructure:"source-key"`

	// NewKey is the value that will be used as the new key for the attribute.
	// This field is required.
	NewKey string `mapstructure:"new-key"`

	// Overwrite is set to true to indicate that the replacement should be
	// performed even if the new key already exists on the attributes.
	// In this case the original value associated with the new key is lost.
	Overwrite bool `mapstructure:"overwrite"`

	// KeepOriginal is set to true to indicate that the original key
	// should not be removed from the attributes.
	KeepOriginal bool `mapstructure:"keep-original"`
}

// ModifyKeyValue defines the key/value attribute pair to set/update.
type ModifyKeyValue struct {
	// SourceKey specifies the attribute to modify the value of.
	// This field is required.
	SourceKey string `mapstructure:"source-key"`

	// NewValue specifies what to replace the old value with.
	// If no value is set, the attribute key/value is removed from the span.
	NewValue interface{} `mapstructure:"new-value"`

	// Overwrite specifies if this operation should overwrite a value, if the attribute
	// key value already exists.
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
