// Copyright Splunk, Inc.
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

package configprovider

import (
	"go.opentelemetry.io/collector/config"
)

// ConfigSettings is the interface required to be implemented by the settings of
// all ConfigSource objects.
type ConfigSettings interface {
	// Name gets the config source name.
	Name() string
	// SetName sets the config source name.
	SetName(name string)
	// Type sets the config source type.
	Type() config.Type
}

// Settings defines common settings of a ConfigSource configuration.
// Specific config sources can embed this struct and extend it with more fields if needed.
// When embedded it must be with `mapstructure:"-"` tag.
type Settings struct {
	TypeVal config.Type `mapstructure:"-"`
	NameVal string      `mapstructure:"-"`
}

// NewSettings return a new ConfigSourceSettings with the given type.
func NewSettings(typeVal config.Type) *Settings {
	return &Settings{TypeVal: typeVal, NameVal: string(typeVal)}
}

// Ensure that Settings satisfy the ConfigSettings interface.
var _ ConfigSettings = (*Settings)(nil)

// Name gets the config source name.
func (css *Settings) Name() string {
	return css.NameVal
}

// SetName sets the config source name.
func (css *Settings) SetName(name string) {
	css.NameVal = name
}

// Type sets the config source type.
func (css *Settings) Type() config.Type {
	return css.TypeVal
}
