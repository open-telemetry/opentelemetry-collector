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

// ExporterConfig is the configuration of a component.Exporter. Specific extensions must implement
// this interface and must embed ExporterSettings struct or a struct that extends it.
type ExporterConfig interface {
	identifiable
	validatable

	privateConfigExporter()
}

// UnmarshalExporterConfig helper function to unmarshal an ExporterConfig.
// It checks if the config implements confmap.Unmarshaler and uses that if available,
// otherwise uses Map.UnmarshalExact, erroring if a field is nonexistent.
func UnmarshalExporterConfig(conf *confmap.Conf, cfg ExporterConfig) error {
	return unmarshal(conf, cfg)
}

// ExporterConfigSettings defines common settings for a component.Exporter configuration.
// Specific exporters can embed this struct and extend it with more fields if needed.
//
// It is highly recommended to "override" the Validate() function.
//
// When embedded in the exporter config, it must be with `mapstructure:",squash"` tag.
type ExporterConfigSettings struct {
	id ID `mapstructure:"-"`
}

// NewExporterConfigSettings return a new ExporterSettings with the given ComponentID.
func NewExporterConfigSettings(id ID) ExporterConfigSettings {
	return ExporterConfigSettings{id: ID{typeVal: id.Type(), nameVal: id.Name()}}
}

var _ ExporterConfig = (*ExporterConfigSettings)(nil)

// ID returns the receiver ComponentID.
func (es *ExporterConfigSettings) ID() ID {
	return es.id
}

// SetIDName sets the receiver name.
func (es *ExporterConfigSettings) SetIDName(idName string) {
	es.id.nameVal = idName
}

// Validate validates the configuration and returns an error if invalid.
func (es *ExporterConfigSettings) Validate() error {
	return nil
}

func (es *ExporterConfigSettings) privateConfigExporter() {}
