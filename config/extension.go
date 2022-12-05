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
import (
	"go.opentelemetry.io/collector/component"
)

// ExtensionSettings defines common settings for a component.Extension configuration.
// Specific processors can embed this struct and extend it with more fields if needed.
//
// When embedded in the extension config, it must be with `mapstructure:",squash"` tag.
type ExtensionSettings struct {
	settings
}

// NewExtensionSettings return a new ExtensionSettings with the given ID.
func NewExtensionSettings(id component.ID) ExtensionSettings {
	return ExtensionSettings{settings: newSettings(id)}
}

var _ component.Config = (*ExtensionSettings)(nil)
