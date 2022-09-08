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

package configunmarshaler // import "go.opentelemetry.io/collector/service/internal/configunmarshaler"

import (
	"reflect"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// connectorsKeyName is the configuration key name for connectors section.
const connectorsKeyName = "connectors"

type Connectors struct {
	conns map[component.ID]component.Config

	factories map[component.Type]component.ConnectorFactory
}

func NewConnectors(factories map[component.Type]component.ConnectorFactory) *Connectors {
	return &Connectors{factories: factories}
}

func (c *Connectors) Unmarshal(conf *confmap.Conf) error {
	rawConns := make(map[component.ID]map[string]interface{})
	if err := conf.Unmarshal(&rawConns, confmap.WithErrorUnused()); err != nil {
		return err
	}

	// Prepare resulting map.
	c.conns = make(map[component.ID]component.Config)

	// Iterate over Connectors and create a config for each.
	for id, value := range rawConns {
		// Find connector factory based on "type" that we read from config source.
		factory := c.factories[id.Type()]
		if factory == nil {
			return errorUnknownType(connectorsKeyName, id, reflect.ValueOf(c.factories).MapKeys())
		}

		// Create the default config for this exporter.
		connectorCfg := factory.CreateDefaultConfig()
		connectorCfg.SetIDName(id.Name()) //nolint:staticcheck

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := component.UnmarshalConfig(confmap.NewFromStringMap(value), connectorCfg); err != nil {
			return errorUnmarshalError(connectorsKeyName, id, err)
		}

		c.conns[id] = connectorCfg
	}

	return nil
}

func (c *Connectors) GetConnectors() map[component.ID]component.Config {
	return c.conns
}
