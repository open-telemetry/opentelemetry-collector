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

package pcommon // import "go.opentelemetry.io/collector/pdata/pcommon"

// This file contains aliases to data structures that are common for all
// signal types, such as timestamps, attributes, etc.

import "go.opentelemetry.io/collector/pdata/internal"

// ValueType is an alias for internal.ValueType type.
type ValueType = internal.ValueType

const (
	ValueTypeEmpty  = internal.ValueTypeEmpty
	ValueTypeString = internal.ValueTypeString
	ValueTypeInt    = internal.ValueTypeInt
	ValueTypeDouble = internal.ValueTypeDouble
	ValueTypeBool   = internal.ValueTypeBool
	ValueTypeMap    = internal.ValueTypeMap
	ValueTypeSlice  = internal.ValueTypeSlice
	ValueTypeBytes  = internal.ValueTypeBytes
)

// Value is an alias for internal.Value struct.
type Value = internal.Value

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
)

// Map is an alias for internal.Map struct.
type Map = internal.Map

// Aliases for functions to create internal.Map.
var (
	NewMap        = internal.NewMap
	NewMapFromRaw = internal.NewMapFromRaw
)
