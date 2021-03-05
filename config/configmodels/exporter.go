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

// Exporter is the configuration of an exporter.
type Exporter interface {
	NamedEntity
}

// Exporters is a map of names to Exporters.
type Exporters map[string]Exporter

// ExporterSettings defines common settings for an exporter configuration.
// Specific exporters can embed this struct and extend it with more fields if needed.
type ExporterSettings struct {
	TypeVal Type   `mapstructure:"-"`
	NameVal string `mapstructure:"-"`
}

var _ Exporter = (*ExporterSettings)(nil)

// Name gets the exporter name.
func (es *ExporterSettings) Name() string {
	return es.NameVal
}

// SetName sets the exporter name.
func (es *ExporterSettings) SetName(name string) {
	es.NameVal = name
}

// Type sets the exporter type.
func (es *ExporterSettings) Type() Type {
	return es.TypeVal
}
