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

package component // import "go.opentelemetry.io/collector/component"

import (
	"go.opentelemetry.io/collector/component/id"
)

// identifiable is an interface that all components configurations MUST embed.
type identifiable interface {
	// ID returns the ID of the component that this configuration belongs to.
	ID() id.ID
	// SetIDName updates the name part of the ID for the component that this configuration belongs to.
	SetIDName(idName string)
}

// Deprecated: [0.65.0] Use id.ID instead.
type ID = id.ID

// Deprecated: [0.65.0] Use id.Type instead.
type Type = id.Type

// Deprecated: [0.65.0] Use id.NewID instead.
func NewID(typeVal Type) ID {
	return id.NewID(typeVal)
}

// Deprecated: [0.65.0] Use id.NewIDWithName instead.
func NewIDWithName(typeVal Type, nameVal string) ID {
	return id.NewIDWithName(typeVal, nameVal)
}
