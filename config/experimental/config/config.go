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

package config

import (
	"go.opentelemetry.io/collector/config"
)

// SourceSettings defines common settings of a Source configuration.
// Specific config sources can embed this struct and extend it with more fields if needed.
// When embedded it must be with `mapstructure:",squash"` tag.
type SourceSettings struct {
	config.ComponentID `mapstructure:"-"`
}

// ID returns the ID of the component that this configuration belongs to.
func (s *SourceSettings) ID() config.ComponentID {
	return s.ComponentID
}

// SetIDName updates the name part of the ID for the component that this configuration belongs to.
func (s *SourceSettings) SetIDName(idName string) {
	s.ComponentID = config.NewIDWithName(s.ComponentID.Type(), idName)
}

// NewSourceSettings return a new config.SourceSettings struct with the given ComponentID.
func NewSourceSettings(id config.ComponentID) SourceSettings {
	return SourceSettings{id}
}

// Source is the configuration of a config source. Specific config sources must implement this
// interface and will typically embed SourceSettings struct or a struct that extends it.
type Source interface {
	// TODO: While config sources are experimental and not in the config package they can't
	//	 reference the private interfaces config.identifiable and config.validatable.
	//   Defining the required methods of the interfaces temporarily here.

	// From config.identifiable:

	// ID returns the ID of the component that this configuration belongs to.
	ID() config.ComponentID
	// SetIDName updates the name part of the ID for the component that this configuration belongs to.
	SetIDName(idName string)

	// From config.validatable:

	// Validate validates the configuration and returns an error if invalid.
	Validate() error
}
