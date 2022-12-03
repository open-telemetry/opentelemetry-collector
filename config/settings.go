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

type settings struct {
	id component.ID `mapstructure:"-"`
	component.Config
}

func newSettings(id component.ID) settings {
	return settings{id: id}
}

var _ component.Config = (*settings)(nil)

// Deprecated: [v0.67.0] use Settings.ID.
func (es *settings) ID() component.ID {
	return es.id
}

// Deprecated: [v0.67.0] use Settings.ID.
func (es *settings) SetIDName(idName string) {
	es.id = component.NewIDWithName(es.id.Type(), idName)
}
