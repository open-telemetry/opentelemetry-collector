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

package config // import "go.opentelemetry.io/collector/config"

// ConfigSource is the configuration of a component.ConfigSource.
// Specific ConfigSources must implement this interface and must embed
// ConfigSourceSettings struct or a struct that extends it.
type ConfigSource interface { //nolint
	identifiable
	validatable

	privateConfigConfigSource()
}

// ConfigSourceSettings defines common settings for a component.ConfigSource configuration.
// Specific processors can embed this struct and extend it with more fields if needed.
//
// It is highly recommended to "override" the Validate() function.
//
// When embedded in the ConfigSource config, it must be with `mapstructure:",squash"` tag.
type ConfigSourceSettings struct { //nolint
	id ComponentID `mapstructure:"-"`
}

// NewConfigSourceSettings return a new ConfigSourceSettings with the given ComponentID.
func NewConfigSourceSettings(id ComponentID) ConfigSourceSettings {
	return ConfigSourceSettings{id: ComponentID{typeVal: id.Type(), nameVal: id.Name()}}
}

var _ ConfigSource = (*ConfigSourceSettings)(nil)

// ID returns the receiver ComponentID.
func (es *ConfigSourceSettings) ID() ComponentID {
	return es.id
}

// SetIDName sets the receiver name.
func (es *ConfigSourceSettings) SetIDName(idName string) {
	es.id.nameVal = idName
}

// Validate validates the configuration and returns an error if invalid.
func (es *ConfigSourceSettings) Validate() error {
	return nil
}

func (es *ConfigSourceSettings) privateConfigConfigSource() {}
