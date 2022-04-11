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

import "go.opentelemetry.io/collector/pdata/pcommon"

// ValueType is an alias for pcommon.ValueType type.
// Deprecated: [v0.49.0] Use pcommon.ValueType instead.
type ValueType = pcommon.ValueType

// AttributeValueType is an alias for pcommon.ValueType type.
// Deprecated: [v0.48.0] Use ValueType instead.
type AttributeValueType = pcommon.ValueType

const (
	// Deprecated: [v0.49.0] Use pcommon.ValueTypeEmpty instead.
	ValueTypeEmpty = pcommon.ValueTypeEmpty

	// Deprecated: [v0.49.0] Use pcommon.ValueTypeString instead.
	ValueTypeString = pcommon.ValueTypeString

	// Deprecated: [v0.49.0] Use pcommon.ValueTypeInt instead.
	ValueTypeInt = pcommon.ValueTypeInt

	// Deprecated: [v0.49.0] Use pcommon.ValueTypeDouble instead.
	ValueTypeDouble = pcommon.ValueTypeDouble

	// Deprecated: [v0.49.0] Use pcommon.ValueTypeBool instead.
	ValueTypeBool = pcommon.ValueTypeBool

	// Deprecated: [v0.49.0] Use pcommon.ValueTypeMap instead.
	ValueTypeMap = pcommon.ValueTypeMap

	// Deprecated: [v0.49.0] Use pcommon.ValueTypeSlice instead.
	ValueTypeSlice = pcommon.ValueTypeSlice

	// Deprecated: [v0.49.0] Use pcommon.ValueTypeBytes instead.
	ValueTypeBytes = pcommon.ValueTypeBytes

	// Deprecated: [v0.48.0] Use ValueTypeEmpty instead.
	AttributeValueTypeEmpty = pcommon.ValueTypeEmpty

	// Deprecated: [v0.48.0] Use ValueTypeString instead.
	AttributeValueTypeString = pcommon.ValueTypeString

	// Deprecated: [v0.48.0] Use ValueTypeInt instead.
	AttributeValueTypeInt = pcommon.ValueTypeInt

	// Deprecated: [v0.48.0] Use ValueTypeDouble instead.
	AttributeValueTypeDouble = pcommon.ValueTypeDouble

	// Deprecated: [v0.48.0] Use ValueTypeBool instead.
	AttributeValueTypeBool = pcommon.ValueTypeBool

	// Deprecated: [v0.48.0] Use ValueTypeMap instead.
	AttributeValueTypeMap = pcommon.ValueTypeMap

	// Deprecated: [v0.48.0] Use ValueTypeSlice instead.
	AttributeValueTypeArray = pcommon.ValueTypeSlice

	// Deprecated: [v0.48.0] Use ValueTypeBytes instead.
	AttributeValueTypeBytes = pcommon.ValueTypeBytes
)

// Value is an alias for pcommon.Value struct.
// Deprecated: [v0.49.0] Use pcommon.Value instead.
type Value = pcommon.Value

// Deprecated: [v0.48.0] Use Value instead.
type AttributeValue = pcommon.Value

// Aliases for functions to create pcommon.Value.
var (

	// Deprecated: [v0.49.0] Use pcommon.NewValueEmpty instead.
	NewValueEmpty = pcommon.NewValueEmpty

	// Deprecated: [v0.49.0] Use pcommon.NewValueString instead.
	NewValueString = pcommon.NewValueString

	// Deprecated: [v0.49.0] Use pcommon.NewValueInt instead.
	NewValueInt = pcommon.NewValueInt

	// Deprecated: [v0.49.0] Use pcommon.NewValueDouble instead.
	NewValueDouble = pcommon.NewValueDouble

	// Deprecated: [v0.49.0] Use pcommon.NewValueBool instead.
	NewValueBool = pcommon.NewValueBool

	// Deprecated: [v0.49.0] Use pcommon.NewValueMap instead.
	NewValueMap = pcommon.NewValueMap

	// Deprecated: [v0.49.0] Use pcommon.NewValueSlice instead.
	NewValueSlice = pcommon.NewValueSlice

	// Deprecated: [v0.49.0] Use pcommon.NewValueBytes instead.
	NewValueBytes = pcommon.NewValueBytes

	// Deprecated: [v0.48.0] Use NewValueEmpty instead.
	NewAttributeValueEmpty = pcommon.NewValueEmpty

	// Deprecated: [v0.48.0] Use NewValueString instead.
	NewAttributeValueString = pcommon.NewValueString

	// Deprecated: [v0.48.0] Use NewValueInt instead.
	NewAttributeValueInt = pcommon.NewValueInt

	// Deprecated: [v0.48.0] Use NewValueDouble instead.
	NewAttributeValueDouble = pcommon.NewValueDouble

	// Deprecated: [v0.48.0] Use NewValueBool instead.
	NewAttributeValueBool = pcommon.NewValueBool

	// Deprecated: [v0.48.0] Use NewValueMap instead.
	NewAttributeValueMap = pcommon.NewValueMap

	// Deprecated: [v0.48.0] Use NewValueSlice instead.
	NewAttributeValueArray = pcommon.NewValueSlice

	// Deprecated: [v0.48.0] Use NewValueBytes instead.
	NewAttributeValueBytes = pcommon.NewValueBytes
)

// Map is an alias for pcommon.Map struct.
// Deprecated: [v0.49.0] Use pcommon.Map instead.
type Map = pcommon.Map

// Deprecated: [v0.48.0] Use Map instead.
type AttributeMap = pcommon.Map

// Aliases for functions to create pcommon.Map.
var (
	// Deprecated: [v0.49.0] Use pcommon.NewMap instead.
	NewMap = pcommon.NewMap

	// Deprecated: [v0.49.0] Use pcommon.NewMapFromRaw instead.
	NewMapFromRaw = pcommon.NewMapFromRaw
)

// Deprecated: [v0.48.0] Use NewMap instead.
var NewAttributeMap = pcommon.NewMap

// Deprecated: [v0.48.0] Use Slice instead.
type AttributeValueSlice = pcommon.Slice

// Deprecated: [v0.48.0] Use NewSlice instead.
var NewAttributeValueSlice = pcommon.NewSlice

// Deprecated: [v0.48.0] Use InstrumentationScope instead.
type InstrumentationLibrary = pcommon.InstrumentationScope

// Deprecated: [v0.48.0] Use NewInstrumentationScope instead.
var NewInstrumentationLibrary = pcommon.NewInstrumentationScope
