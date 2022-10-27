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

// ProcessorConfig is the configuration of a component.Processor. Specific extensions must implement
// this interface and must embed ProcessorConfigSettings struct or a struct that extends it.
type ProcessorConfig interface {
	identifiable
	validatable

	privateConfigProcessor()
}

// UnmarshalProcessorConfig helper function to unmarshal a ProcessorConfig.
// It checks if the config implements confmap.Unmarshaler and uses that if available,
// otherwise uses Map.UnmarshalExact, erroring if a field is nonexistent.
func UnmarshalProcessorConfig(conf *confmap.Conf, cfg ProcessorConfig) error {
	return unmarshal(conf, cfg)
}

// ProcessorConfigSettings defines common settings for a component.Processor configuration.
// Specific processors can embed this struct and extend it with more fields if needed.
//
// It is highly recommended to "override" the Validate() function.
//
// When embedded in the processor config it must be with `mapstructure:",squash"` tag.
type ProcessorConfigSettings struct {
	id ID `mapstructure:"-"`
}

// NewProcessorConfigSettings return a new ProcessorConfigSettings with the given ComponentID.
func NewProcessorConfigSettings(id ID) ProcessorConfigSettings {
	return ProcessorConfigSettings{id: ID{typeVal: id.Type(), nameVal: id.Name()}}
}

var _ ProcessorConfig = (*ProcessorConfigSettings)(nil)

// ID returns the receiver ComponentID.
func (ps *ProcessorConfigSettings) ID() ID {
	return ps.id
}

// SetIDName sets the receiver name.
func (ps *ProcessorConfigSettings) SetIDName(idName string) {
	ps.id.nameVal = idName
}

// Validate validates the configuration and returns an error if invalid.
func (ps *ProcessorConfigSettings) Validate() error {
	return nil
}

func (ps *ProcessorConfigSettings) privateConfigProcessor() {}
