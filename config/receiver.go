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

// Receiver is the configuration of a receiver. Specific receivers must implement this
// interface and will typically embed ReceiverSettings struct or a struct that extends it.
// Embedded CustomConfigOptions will force each receiver to implement Validate() function
type Receiver interface {
	NamedEntity
	CustomConfigOptions
}

// CustomConfigOptions defines the interfaces for configuration customization options
// Different custom interfaces can be added like CustomConfigValidator and any other interface.
type CustomConfigOptions interface {
	CustomConfigValidator
}

// CustomConfigValidator defines the interface for the custom configuration validation on each component
type CustomConfigValidator interface {
	Validate() error
}

// CustomValidator is a function that runs the customized validation
// on component level configuration.
type CustomValidator func() error

// Receivers is a map of names to Receivers.
type Receivers map[string]Receiver

// ReceiverSettings defines common settings for a receiver configuration.
// Specific receivers can embed this struct and extend it with more fields if needed.
type ReceiverSettings struct {
	TypeVal Type   `mapstructure:"-"`
	NameVal string `mapstructure:"-"`
}

var _ NamedEntity = (*ReceiverSettings)(nil)

// Name gets the receiver name.
func (rs *ReceiverSettings) Name() string {
	return rs.NameVal
}

// SetName sets the receiver name.
func (rs *ReceiverSettings) SetName(name string) {
	rs.NameVal = name
}

// Type sets the receiver type.
func (rs *ReceiverSettings) Type() Type {
	return rs.TypeVal
}
