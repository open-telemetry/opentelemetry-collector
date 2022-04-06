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

package pdata // import "go.opentelemetry.io/collector/model/pdata"

// This file contains aliases to data structures that are common for all
// signal types, such as timestamps, attributes, etc.

import "go.opentelemetry.io/collector/model/internal"

// ValueType is an alias for internal.ValueType type.
type ValueType = internal.ValueType

// AttributeValueType is an alias for internal.ValueType type.
// Deprecated: [v0.48.0] Use ValueType instead.
type AttributeValueType = internal.ValueType

const (
	ValueTypeEmpty  = internal.ValueTypeEmpty
	ValueTypeString = internal.ValueTypeString
	ValueTypeInt    = internal.ValueTypeInt
	ValueTypeDouble = internal.ValueTypeDouble
	ValueTypeBool   = internal.ValueTypeBool
	ValueTypeMap    = internal.ValueTypeMap
	ValueTypeSlice  = internal.ValueTypeSlice
	ValueTypeBytes  = internal.ValueTypeBytes

	// Deprecated: [v0.48.0] Use ValueTypeEmpty instead.
	AttributeValueTypeEmpty = internal.ValueTypeEmpty

	// Deprecated: [v0.48.0] Use ValueTypeString instead.
	AttributeValueTypeString = internal.ValueTypeString

	// Deprecated: [v0.48.0] Use ValueTypeInt instead.
	AttributeValueTypeInt = internal.ValueTypeInt

	// Deprecated: [v0.48.0] Use ValueTypeDouble instead.
	AttributeValueTypeDouble = internal.ValueTypeDouble

	// Deprecated: [v0.48.0] Use ValueTypeBool instead.
	AttributeValueTypeBool = internal.ValueTypeBool

	// Deprecated: [v0.48.0] Use ValueTypeMap instead.
	AttributeValueTypeMap = internal.ValueTypeMap

	// Deprecated: [v0.48.0] Use ValueTypeSlice instead.
	AttributeValueTypeArray = internal.ValueTypeSlice

	// Deprecated: [v0.48.0] Use ValueTypeBytes instead.
	AttributeValueTypeBytes = internal.ValueTypeBytes
)

// Value is an alias for internal.Value struct.
type Value = internal.Value

// Deprecated: [v0.48.0] Use Value instead.
type AttributeValue = internal.Value

// Aliases for functions to create internal.Value.
var (
	NewValueEmpty  = internal.NewValueEmpty
	NewValueString = internal.NewValueString
	NewValueInt    = internal.NewValueInt
	NewValueDouble = internal.NewValueDouble
	NewValueBool   = internal.NewValueBool
	NewValueMap    = internal.NewValueMap
	NewValueSlice  = internal.NewValueSlice
	NewValueBytes  = internal.NewValueBytes

	// Deprecated: [v0.48.0] Use NewValueEmpty instead.
	NewAttributeValueEmpty = internal.NewValueEmpty

	// Deprecated: [v0.48.0] Use NewValueString instead.
	NewAttributeValueString = internal.NewValueString

	// Deprecated: [v0.48.0] Use NewValueInt instead.
	NewAttributeValueInt = internal.NewValueInt

	// Deprecated: [v0.48.0] Use NewValueDouble instead.
	NewAttributeValueDouble = internal.NewValueDouble

	// Deprecated: [v0.48.0] Use NewValueBool instead.
	NewAttributeValueBool = internal.NewValueBool

	// Deprecated: [v0.48.0] Use NewValueMap instead.
	NewAttributeValueMap = internal.NewValueMap

	// Deprecated: [v0.48.0] Use NewValueSlice instead.
	NewAttributeValueArray = internal.NewValueSlice

	// Deprecated: [v0.48.0] Use NewValueBytes instead.
	NewAttributeValueBytes = internal.NewValueBytes
)

// Map is an alias for internal.Map struct.
type Map = internal.Map

// Deprecated: [v0.48.0] Use Map instead.
type AttributeMap = internal.Map

// Aliases for functions to create internal.Map.
var (
	NewMap        = internal.NewMap
	NewMapFromRaw = internal.NewMapFromRaw
)

// Deprecated: [v0.48.0] Use NewMap instead.
var NewAttributeMap = internal.NewMap

// Deprecated: [v0.48.0] Use NewMapFromRaw instead.
var NewAttributeMapFromMap = internal.NewAttributeMapFromMap

// Deprecated: [v0.48.0] Use Slice instead.
type AttributeValueSlice = internal.Slice

// Deprecated: [v0.48.0] Use NewSlice instead.
var NewAttributeValueSlice = internal.NewSlice

// Deprecated: [v0.48.0] Use InstrumentationScope instead.
type InstrumentationLibrary = internal.InstrumentationScope

// Deprecated: [v0.48.0] Use NewInstrumentationScope instead.
var NewInstrumentationLibrary = internal.NewInstrumentationScope
