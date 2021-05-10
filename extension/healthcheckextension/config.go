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

package healthcheckextension

import (
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confignet"
)

// Config has the configuration for the extension enabling the health check
// extension, used to report the health status of the service.
type Config struct {
	config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	// Port is the port used to publish the health check status.
	// The default value is 13133.
	// Deprecated: use Endpoint instead.
	Port uint16 `mapstructure:"port"`

	// TCPAddr represents a tcp endpoint address that is to publish the health
	// check status.
	// The default endpoint is "0.0.0.0:13133".
	TCPAddr confignet.TCPAddr `mapstructure:",squash"`
}

var _ config.Extension = (*Config)(nil)

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	return nil
}
