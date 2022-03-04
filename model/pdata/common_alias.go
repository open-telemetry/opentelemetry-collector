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

import "go.opentelemetry.io/collector/model/internal/pdata"

// AttributeValueType is an alias for pdata.AttributeValueType type.
type AttributeValueType = pdata.AttributeValueType

const (
	AttributeValueTypeEmpty  = pdata.AttributeValueTypeEmpty
	AttributeValueTypeString = pdata.AttributeValueTypeString
	AttributeValueTypeInt    = pdata.AttributeValueTypeInt
	AttributeValueTypeDouble = pdata.AttributeValueTypeDouble
	AttributeValueTypeBool   = pdata.AttributeValueTypeBool
	AttributeValueTypeMap    = pdata.AttributeValueTypeMap
	AttributeValueTypeArray  = pdata.AttributeValueTypeArray
	AttributeValueTypeBytes  = pdata.AttributeValueTypeBytes
)

// AttributeValue is an alias for pdata.AttributeValue struct.
type AttributeValue = pdata.AttributeValue

// Aliases for functions to create pdata.AttributeValue.
var (
	NewAttributeValueEmpty  = pdata.NewAttributeValueEmpty
	NewAttributeValueString = pdata.NewAttributeValueString
	NewAttributeValueInt    = pdata.NewAttributeValueInt
	NewAttributeValueDouble = pdata.NewAttributeValueDouble
	NewAttributeValueBool   = pdata.NewAttributeValueBool
	NewAttributeValueMap    = pdata.NewAttributeValueMap
	NewAttributeValueArray  = pdata.NewAttributeValueArray
	NewAttributeValueBytes  = pdata.NewAttributeValueBytes
)

// AttributeMap is an alias for pdata.AttributeMap struct.
type AttributeMap = pdata.AttributeMap

// Aliases for functions to create pdata.AttributeMap.
var (
	NewAttributeMap        = pdata.NewAttributeMap
	NewAttributeMapFromMap = pdata.NewAttributeMapFromMap
)
