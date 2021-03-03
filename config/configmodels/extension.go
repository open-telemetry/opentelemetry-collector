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

// Extension is the configuration of a service extension. Specific extensions
// must implement this interface and will typically embed ExtensionSettings
// struct or a struct that extends it.
type Extension interface {
	NamedEntity
}

// Extensions is a map of names to extensions.
type Extensions map[string]Extension

// ExtensionSettings defines common settings for a service extension configuration.
// Specific extensions can embed this struct and extend it with more fields if needed.
type ExtensionSettings struct {
	TypeVal Type   `mapstructure:"-"`
	NameVal string `mapstructure:"-"`
}

var _ Extension = (*ExtensionSettings)(nil)

// Name gets the extension name.
func (ext *ExtensionSettings) Name() string {
	return ext.NameVal
}

// SetName sets the extension name.
func (ext *ExtensionSettings) SetName(name string) {
	ext.NameVal = name
}

// Type sets the extension type.
func (ext *ExtensionSettings) Type() Type {
	return ext.TypeVal
}
