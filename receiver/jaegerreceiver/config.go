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
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
)

// The config field name to load the protocol map from
const protocolsFieldName = "protocols"

// RemoteSamplingConfig defines config key for remote sampling fetch endpoint
type RemoteSamplingConfig struct {
	HostEndpoint  string `mapstructure:"host_endpoint"`
	FetchEndpoint string `mapstructure:"fetch_endpoint"`
	StrategyFile  string `mapstructure:"strategy_file"`
}

// Config defines configuration for Jaeger receiver.
type Config struct {
	TypeVal        configmodels.Type                           `mapstructure:"-"`
	NameVal        string                                      `mapstructure:"-"`
	Protocols      map[string]*receiver.SecureReceiverSettings `mapstructure:"protocols"`
	RemoteSampling *RemoteSamplingConfig                       `mapstructure:"remote_sampling"`
}

// Name gets the receiver name.
func (rs *Config) Name() string {
	return rs.NameVal
}

// SetName sets the receiver name.
func (rs *Config) SetName(name string) {
	rs.NameVal = name
}

// Type sets the receiver type.
func (rs *Config) Type() configmodels.Type {
	return rs.TypeVal
}

// SetType sets the receiver type.
func (rs *Config) SetType(typeStr configmodels.Type) {
	rs.TypeVal = typeStr
}

// IsEnabled returns true if the entity is enabled.
func (rs *Config) IsEnabled() bool {
	for _, p := range rs.Protocols {
		if p.IsEnabled() {
			// If any protocol is enabled then the receiver as a whole should be enabled.
			return true
		}
	}
	// All protocols are disabled so the entire receiver can be disabled.
	return false
}
