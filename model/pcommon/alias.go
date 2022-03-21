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

package pcommon // import "go.opentelemetry.io/collector/model/pcommon"

// This file contains aliases to data structures that are common for all
// signal types, such as timestamps, attributes, etc.

import "go.opentelemetry.io/collector/model/internal/pdata"

// ValueType is an alias for pdata.ValueType type.
type ValueType = pdata.ValueType

const (
	ValueTypeEmpty  = pdata.ValueTypeEmpty
	ValueTypeString = pdata.ValueTypeString
	ValueTypeInt    = pdata.ValueTypeInt
	ValueTypeDouble = pdata.ValueTypeDouble
	ValueTypeBool   = pdata.ValueTypeBool
	ValueTypeMap    = pdata.ValueTypeMap
	ValueTypeSlice  = pdata.ValueTypeSlice
	ValueTypeBytes  = pdata.ValueTypeBytes
)

// Value is an alias for pdata.Value struct.
type Value = pdata.Value

// Aliases for functions to create pdata.Value.
var (
	NewValueEmpty  = pdata.NewValueEmpty
	NewValueString = pdata.NewValueString
	NewValueInt    = pdata.NewValueInt
	NewValueDouble = pdata.NewValueDouble
	NewValueBool   = pdata.NewValueBool
	NewValueMap    = pdata.NewValueMap
	NewValueSlice  = pdata.NewValueSlice
	NewValueBytes  = pdata.NewValueBytes
)

// Map is an alias for pdata.Map struct.
type Map = pdata.Map

// Deprecated: [v0.48.0] Use Map instead.
type AttributeMap = pdata.Map

// Aliases for functions to create pdata.Map.
var (
	NewMap        = pdata.NewMap
	NewMapFromRaw = pdata.NewMapFromRaw
)
