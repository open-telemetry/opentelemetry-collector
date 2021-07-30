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

// Extension is the configuration of a component.Extension. Specific extensions must implement
// this interface and must embed ExtensionSettings struct or a struct that extends it.
type Extension interface {
	identifiable
	validatable

	privateConfigExtension()
}

// Extensions is a map of names to extensions.
type Extensions map[ComponentID]Extension

// ExtensionSettings defines common settings for a component.Extension configuration.
// Specific processors can embed this struct and extend it with more fields if needed.
//
// It is highly recommended to "override" the Validate() function.
//
// When embedded in the extension config, it must be with `mapstructure:",squash"` tag.
type ExtensionSettings struct {
	id ComponentID `mapstructure:"-"`
}

// NewExtensionSettings return a new ExtensionSettings with the given ComponentID.
func NewExtensionSettings(id ComponentID) ExtensionSettings {
	return ExtensionSettings{id: ComponentID{typeVal: id.Type(), nameVal: id.Name()}}
}

var _ Extension = (*ExtensionSettings)(nil)

// ID returns the receiver ComponentID.
func (rs *ExtensionSettings) ID() ComponentID {
	return rs.id
}

// SetIDName sets the receiver name.
func (rs *ExtensionSettings) SetIDName(idName string) {
	rs.id.nameVal = idName
}

// Validate validates the configuration and returns an error if invalid.
func (rs *ExtensionSettings) Validate() error {
	return nil
}

func (rs *ExtensionSettings) privateConfigExtension() {}
