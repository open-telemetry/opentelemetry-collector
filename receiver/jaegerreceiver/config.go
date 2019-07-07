// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jaegerreceiver

import (
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

// ConfigV2 defines configuration for Jaeger receiver.
type ConfigV2 struct {
	TypeVal   string                                    `mapstructure:"-"`
	NameVal   string                                    `mapstructure:"-"`
	Protocols map[string]*configmodels.ReceiverSettings `mapstructure:"protocols"`
}

// Name gets the receiver name.
func (rs *ConfigV2) Name() string {
	return rs.NameVal
}

// SetName sets the receiver name.
func (rs *ConfigV2) SetName(name string) {
	rs.NameVal = name
}

// Type sets the receiver type.
func (rs *ConfigV2) Type() string {
	return rs.TypeVal
}

// SetType sets the receiver type.
func (rs *ConfigV2) SetType(typeStr string) {
	rs.TypeVal = typeStr
}
