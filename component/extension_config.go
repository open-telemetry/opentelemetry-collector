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
	"go.opentelemetry.io/collector/confmap"
)

// ExtensionConfig is the configuration of a component.Extension. Specific extensions must implement
// this interface and must embed ExtensionConfigSettings struct or a struct that extends it.
type ExtensionConfig interface {
	identifiable
	validatable

	privateConfigExtension()
}

// UnmarshalExtensionConfig helper function to unmarshal an ExtensionConfig.
// It checks if the config implements confmap.Unmarshaler and uses that if available,
// otherwise uses Map.UnmarshalExact, erroring if a field is nonexistent.
func UnmarshalExtensionConfig(conf *confmap.Conf, cfg ExtensionConfig) error {
	return unmarshal(conf, cfg)
}

// ExtensionConfigSettings defines common settings for a component.Extension configuration.
// Specific processors can embed this struct and extend it with more fields if needed.
//
// It is highly recommended to "override" the Validate() function.
//
// When embedded in the extension config, it must be with `mapstructure:",squash"` tag.
type ExtensionConfigSettings struct {
	id ID `mapstructure:"-"`
}

// NewExtensionConfigSettings return a new ExtensionConfigSettings with the given ID.
func NewExtensionConfigSettings(id ID) ExtensionConfigSettings {
	return ExtensionConfigSettings{id: ID{typeVal: id.Type(), nameVal: id.Name()}}
}

var _ ExtensionConfig = (*ExtensionConfigSettings)(nil)

// ID returns the receiver ID.
func (es *ExtensionConfigSettings) ID() ID {
	return es.id
}

// SetIDName sets the receiver name.
func (es *ExtensionConfigSettings) SetIDName(idName string) {
	es.id.nameVal = idName
}

// Validate validates the configuration and returns an error if invalid.
func (es *ExtensionConfigSettings) Validate() error {
	return nil
}

func (es *ExtensionConfigSettings) privateConfigExtension() {}
