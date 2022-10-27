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

// ReceiverConfig is the configuration of a component.Receiver. Specific extensions must implement
// this interface and must embed ReceiverConfigSettings struct or a struct that extends it.
type ReceiverConfig interface {
	identifiable
	validatable

	privateConfigReceiver()
}

// UnmarshalReceiverConfig helper function to unmarshal a ReceiverConfig.
// It checks if the config implements confmap.Unmarshaler and uses that if available,
// otherwise uses Map.UnmarshalExact, erroring if a field is nonexistent.
func UnmarshalReceiverConfig(conf *confmap.Conf, cfg ReceiverConfig) error {
	return unmarshal(conf, cfg)
}

// ReceiverConfigSettings defines common settings for a component.Receiver configuration.
// Specific receivers can embed this struct and extend it with more fields if needed.
//
// It is highly recommended to "override" the Validate() function.
//
// When embedded in the receiver config it must be with `mapstructure:",squash"` tag.
type ReceiverConfigSettings struct {
	id ID `mapstructure:"-"`
}

// NewReceiverConfigSettings return a new ReceiverConfigSettings with the given ID.
func NewReceiverConfigSettings(id ID) ReceiverConfigSettings {
	return ReceiverConfigSettings{id: ID{typeVal: id.Type(), nameVal: id.Name()}}
}

var _ ReceiverConfig = (*ReceiverConfigSettings)(nil)

// ID returns the receiver ID.
func (rs *ReceiverConfigSettings) ID() ID {
	return rs.id
}

// SetIDName sets the receiver name.
func (rs *ReceiverConfigSettings) SetIDName(idName string) {
	rs.id.nameVal = idName
}

// Validate validates the configuration and returns an error if invalid.
func (rs *ReceiverConfigSettings) Validate() error {
	return nil
}

func (rs *ReceiverConfigSettings) privateConfigReceiver() {}
