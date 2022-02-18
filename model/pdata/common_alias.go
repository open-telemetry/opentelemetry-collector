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

import "go.opentelemetry.io/collector/model/pcommon"

// AttributeValueType is an alias for pcommon.AttributeValueType type.
type AttributeValueType = pcommon.AttributeValueType

const (
	AttributeValueTypeEmpty  = pcommon.AttributeValueTypeEmpty
	AttributeValueTypeString = pcommon.AttributeValueTypeString
	AttributeValueTypeInt    = pcommon.AttributeValueTypeInt
	AttributeValueTypeDouble = pcommon.AttributeValueTypeDouble
	AttributeValueTypeBool   = pcommon.AttributeValueTypeBool
	AttributeValueTypeMap    = pcommon.AttributeValueTypeMap
	AttributeValueTypeArray  = pcommon.AttributeValueTypeArray
	AttributeValueTypeBytes  = pcommon.AttributeValueTypeBytes
)

// AttributeValue is an alias for pcommon.AttributeValue struct.
type AttributeValue = pcommon.AttributeValue

// Aliases for functions to create pcommon.AttributeValue.
var (
	NewAttributeValueEmpty  = pcommon.NewAttributeValueEmpty
	NewAttributeValueString = pcommon.NewAttributeValueString
	NewAttributeValueInt    = pcommon.NewAttributeValueInt
	NewAttributeValueDouble = pcommon.NewAttributeValueDouble
	NewAttributeValueBool   = pcommon.NewAttributeValueBool
	NewAttributeValueMap    = pcommon.NewAttributeValueMap
	NewAttributeValueArray  = pcommon.NewAttributeValueArray
	NewAttributeValueBytes  = pcommon.NewAttributeValueBytes
)

// AttributeMap is an alias for pcommon.AttributeMap struct.
type AttributeMap = pcommon.AttributeMap

// Aliases for functions to create pcommon.AttributeMap.
var (
	NewAttributeMap        = pcommon.NewAttributeMap
	NewAttributeMapFromMap = pcommon.NewAttributeMapFromMap
)
