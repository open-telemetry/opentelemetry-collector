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

package config

// Exporter is the configuration of a component.Exporter. Specific extensions must implement
// this interface and must embed ExporterSettings struct or a struct that extends it.
type Exporter interface {
	identifiable
	validatable

	privateConfigExporter()
}

// Exporters is a map of names to Exporters.
type Exporters map[ComponentID]Exporter

// ExporterSettings defines common settings for a component.Exporter configuration.
// Specific exporters can embed this struct and extend it with more fields if needed.
//
// It is highly recommended to "override" the Validate() function.
//
// When embedded in the exporter config, it must be with `mapstructure:",squash"` tag.
type ExporterSettings struct {
	id ComponentID `mapstructure:"-"`
}

// NewExporterSettings return a new ExporterSettings with the given ComponentID.
func NewExporterSettings(id ComponentID) ExporterSettings {
	return ExporterSettings{id: ComponentID{typeVal: id.Type(), nameVal: id.Name()}}
}

var _ Exporter = (*ExporterSettings)(nil)

// ID returns the receiver ComponentID.
func (rs *ExporterSettings) ID() ComponentID {
	return rs.id
}

// SetIDName sets the receiver name.
func (rs *ExporterSettings) SetIDName(idName string) {
	rs.id.nameVal = idName
}

func (rs *ExporterSettings) Validate() error {
	return nil
}

func (rs *ExporterSettings) privateConfigExporter() {}
