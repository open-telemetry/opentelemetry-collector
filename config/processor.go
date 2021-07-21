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

// Processor is the configuration of a component.Processor. Specific extensions must implement
// this interface and must embed ProcessorSettings struct or a struct that extends it.
type Processor interface {
	identifiable
	validatable

	privateConfigProcessor()
}

// Processors is a map of names to Processors.
type Processors map[ComponentID]Processor

// ProcessorSettings defines common settings for a component.Processor configuration.
// Specific processors can embed this struct and extend it with more fields if needed.
//
// It is highly recommended to "override" the Validate() function.
//
// When embedded in the processor config it must be with `mapstructure:",squash"` tag.
type ProcessorSettings struct {
	id ComponentID `mapstructure:"-"`
}

// NewProcessorSettings return a new ProcessorSettings with the given ComponentID.
func NewProcessorSettings(id ComponentID) ProcessorSettings {
	return ProcessorSettings{id: ComponentID{typeVal: id.Type(), nameVal: id.Name()}}
}

var _ Processor = (*ProcessorSettings)(nil)

// ID returns the receiver ComponentID.
func (rs *ProcessorSettings) ID() ComponentID {
	return rs.id
}

// SetIDName sets the receiver name.
func (rs *ProcessorSettings) SetIDName(idName string) {
	rs.id.nameVal = idName
}

// Validate validates the configuration and returns an error if invalid.
func (rs *ProcessorSettings) Validate() error {
	return nil
}

func (rs *ProcessorSettings) privateConfigProcessor() {}
