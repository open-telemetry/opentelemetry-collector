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

// ReceiverSettings defines common settings for a component.Receiver configuration.
// Specific receivers can embed this struct and extend it with more fields if needed.
//
// It is highly recommended to "override" the Validate() function.
//
// When embedded in the receiver config it must be with `mapstructure:",squash"` tag.
type ReceiverSettings struct {
	id component.ID `mapstructure:"-"`
	component.ReceiverConfig
}

// NewReceiverSettings return a new ReceiverSettings with the given ID.
func NewReceiverSettings(id component.ID) ReceiverSettings {
	return ReceiverSettings{id: id}
}

var _ component.ReceiverConfig = (*ReceiverSettings)(nil)

// ID returns the receiver ID.
func (rs *ReceiverSettings) ID() component.ID {
	return rs.id
}

// SetIDName sets the receiver name.
func (rs *ReceiverSettings) SetIDName(idName string) {
	rs.id = component.NewIDWithName(rs.id.Type(), idName)
}

// Deprecated: [v0.65.0] Not needed anymore since the Validate() will be moved from Config interface to the optional ConfigValidator interface.
func (rs *ReceiverSettings) Validate() error {
	return nil
}
