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

package configmodels

import (
	"go.opentelemetry.io/collector/config/configmodels"
)

// ConfigSource is the configuration of a config source. Specific configuration
// sources must implement this interface and will typically embed ConfigSourceSettings
// or a struct that extends it.
type ConfigSource interface {
	configmodels.NamedEntity
}

// ConfigSources is a map of names to ConfigSource instances.
type ConfigSources map[string]ConfigSource

// ConfigSourceSettings defines common settings for a ConfigSource configuration.
// Specific ConfigSource types can embed this struct and extend it with more fields if needed.
type ConfigSourceSettings struct {
	TypeVal configmodels.Type `mapstructure:"-"`
	NameVal string            `mapstructure:"-"`
}

var _ ConfigSource = (*ConfigSourceSettings)(nil)

// Name gets the ConfigSource name.
func (css *ConfigSourceSettings) Name() string {
	return css.NameVal
}

// SetName sets the ConfigSource name.
func (css *ConfigSourceSettings) SetName(name string) {
	css.NameVal = name
}

// Type sets the ConfigSource type.
func (css *ConfigSourceSettings) Type() configmodels.Type {
	return css.TypeVal
}
