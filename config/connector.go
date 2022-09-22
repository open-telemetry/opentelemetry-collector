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
	"go.opentelemetry.io/collector/confmap"
)

// Connector is the configuration of a component.Connector. Specific extensions must implement
// this interface and must embed ConnectorSettings struct or a struct that extends it.
type Connector interface {
	identifiable
	validatable

	// Implement both to ensure:
	// 1. Only connectors are defined in 'connectors' section
	// 2. Connectors may be placed in receiver and exporter positions in pipelines
	privateConfigExporter()
	privateConfigReceiver()
}

// UnmarshalConnector helper function to unmarshal an Connector config.
// It checks if the config implements confmap.Unmarshaler and uses that if available,
// otherwise uses Map.UnmarshalExact, erroring if a field is nonexistent.
func UnmarshalConnector(conf *confmap.Conf, cfg Connector) error {
	return unmarshal(conf, cfg)
}

// ConnectorSettings defines common settings for a component.Connector configuration.
// Specific exporters can embed this struct and extend it with more fields if needed.
//
// It is highly recommended to "override" the Validate() function.
//
// When embedded in the exporter config, it must be with `mapstructure:",squash"` tag.
type ConnectorSettings struct {
	id ComponentID `mapstructure:"-"`
}

// NewConnectorSettings return a new ConnectorSettings with the given ComponentID.
func NewConnectorSettings(id ComponentID) ConnectorSettings {
	return ConnectorSettings{id: ComponentID{typeVal: id.Type(), nameVal: id.Name()}}
}

var _ Connector = (*ConnectorSettings)(nil)

// ID returns the receiver ComponentID.
func (es *ConnectorSettings) ID() ComponentID {
	return es.id
}

// SetIDName sets the receiver name.
func (es *ConnectorSettings) SetIDName(idName string) {
	es.id.nameVal = idName
}

// Validate validates the configuration and returns an error if invalid.
func (es *ConnectorSettings) Validate() error {
	return nil
}

func (es *ConnectorSettings) privateConfigExporter() {}

func (es *ConnectorSettings) privateConfigReceiver() {}
