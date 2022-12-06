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

// ConnectorSettings defines common settings for a component.Connector configuration.
// Specific connectors can embed this struct and extend it with more fields if needed.
//
// When embedded in the exporter config, it must be with `mapstructure:",squash"` tag.
type ConnectorSettings struct {
	settings
}

// NewConnectorSettings return a new ConnectorSettings with the given ComponentID.
func NewConnectorSettings(id component.ID) ConnectorSettings {
	return ConnectorSettings{settings: newSettings(id)}
}

var _ component.Config = (*ConnectorSettings)(nil)
