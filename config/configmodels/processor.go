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

// Processor is the configuration of a processor. Specific processors must implement this
// interface and will typically embed ProcessorSettings struct or a struct that extends it.
type Processor interface {
	NamedEntity
}

// Processors is a map of names to Processors.
type Processors map[string]Processor

// ProcessorSettings defines common settings for a processor configuration.
// Specific processors can embed this struct and extend it with more fields if needed.
type ProcessorSettings struct {
	TypeVal Type   `mapstructure:"-"`
	NameVal string `mapstructure:"-"`
}

var _ Processor = (*ProcessorSettings)(nil)

// Name gets the processor name.
func (proc *ProcessorSettings) Name() string {
	return proc.NameVal
}

// SetName sets the processor name.
func (proc *ProcessorSettings) SetName(name string) {
	proc.NameVal = name
}

// Type sets the processor type.
func (proc *ProcessorSettings) Type() Type {
	return proc.TypeVal
}
