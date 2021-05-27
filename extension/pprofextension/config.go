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

package pprofextension

import (
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confignet"
)

// Config has the configuration for the extension enabling the golang
// net/http/pprof (Performance Profiler) extension.
type Config struct {
	config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	// TCPAddr is the address and port in which the pprof will be listening to.
	// Use localhost:<port> to make it available only locally, or ":<port>" to
	// make it available on all network interfaces.
	TCPAddr confignet.TCPAddr `mapstructure:",squash"`

	// Fraction of blocking events that are profiled. A value <= 0 disables
	// profiling. See https://golang.org/pkg/runtime/#SetBlockProfileRate for details.
	BlockProfileFraction int `mapstructure:"block_profile_fraction"`

	// Fraction of mutex contention events that are profiled. A value <= 0
	// disables profiling. See https://golang.org/pkg/runtime/#SetMutexProfileFraction
	// for details.
	MutexProfileFraction int `mapstructure:"mutex_profile_fraction"`

	// Optional file name to save the CPU profile to. The profiling starts when the
	// Collector starts and is saved to the file when the Collector is terminated.
	SaveToFile string `mapstructure:"save_to_file"`
}

var _ config.Extension = (*Config)(nil)

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	return nil
}
