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

package configsource

import (
	"go.opentelemetry.io/collector/config"
)

// Settings defines common settings of a ConfigSource configuration.
// Specific config sources can embed this struct and extend it with more fields if needed.
// When embedded it must be with `mapstructure:",squash"` tag.
type Settings struct {
	config.ComponentID `mapstructure:"-"`
}

var _ Config = (*Settings)(nil)

// ID returns the ID of the component that this configuration belongs to.
func (s *Settings) ID() config.ComponentID {
	return s.ComponentID
}

// SetIDName updates the name part of the ID for the component that this configuration belongs to.
func (s *Settings) SetIDName(idName string) {
	s.ComponentID = config.NewIDWithName(s.ComponentID.Type(), idName)
}

// Validate validates the configuration and returns an error if invalid.
func (s *Settings) Validate() error {
	return nil
}

// NewSettings return a new configsource.Settings struct with the given ComponentID.
func NewSettings(id config.ComponentID) Settings {
	return Settings{id}
}

// Config is the configuration of a ConfigSource. Specific config sources must implement this
// interface and will typically embed Settings struct or a struct that extends it.
type Config interface {
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
