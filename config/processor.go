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

// ProcessorSettings defines common settings for a component.Processor configuration.
// Specific processors can embed this struct and extend it with more fields if needed.
//
// It is highly recommended to "override" the Validate() function.
//
// When embedded in the processor config it must be with `mapstructure:",squash"` tag.
type ProcessorSettings struct {
	id component.ID `mapstructure:"-"`
	component.ProcessorConfig
}

// NewProcessorSettings return a new ProcessorSettings with the given ComponentID.
func NewProcessorSettings(id component.ID) ProcessorSettings {
	return ProcessorSettings{id: id}
}

var _ component.ProcessorConfig = (*ProcessorSettings)(nil)

// ID returns the receiver ComponentID.
func (ps *ProcessorSettings) ID() component.ID {
	return ps.id
}

// SetIDName sets the receiver name.
func (ps *ProcessorSettings) SetIDName(idName string) {
	ps.id = component.NewIDWithName(ps.id.Type(), idName)
}

// Deprecated: [v0.65.0] use component.ValidateConfig.
func (ps *ProcessorSettings) Validate() error {
	return nil
}
