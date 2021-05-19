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

package bearertokenauthextension

import (
	"errors"

	"go.opentelemetry.io/collector/config"
)

// Config specifies how the Per-RPC bearer token based authentication data should be obtained.
type Config struct {
	config.ExtensionSettings `mapstructure:",squash"`

	// BearerToken specifies the bearer token to use for every RPC.
	BearerToken string `mapstructure:"token,omitempty"`
}

var _ config.Extension = (*Config)(nil)
var errNoTokenProvided = errors.New("no bearer token provided")

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	if cfg.BearerToken == "" {
		return errNoTokenProvided
	}
	return nil
}
