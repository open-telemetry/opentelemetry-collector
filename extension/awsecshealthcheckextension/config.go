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

package awsecshealthcheckextension

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confignet"
)

type Config struct {
	config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	// TCPAddr represents a tcp endpoint address that is to publish the health check status.
	// The default endpoint is "127.0.0.1:13134".
	TCPAddr confignet.TCPAddr `mapstructure:",squash"`

	// Interval is the time error kept, and error limit is the threshold of number of error happened during the interval
	Interval string `mapstructure:"interval"`

	// current scope will be focus on exporter failures
	// TODO receiver/processor failures
	ExporterErrorLimit int `mapstructure:"exporter_error_limit"`
}

var _ config.Extension = (*Config)(nil)

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	_, err := time.ParseDuration(cfg.Interval)
	if err != nil {
		return err
	}
	if cfg.ExporterErrorLimit < 0 {
		return errors.New("math: error limit is a negative number")
	}
	return nil
}
