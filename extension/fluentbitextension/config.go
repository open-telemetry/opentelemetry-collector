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

package fluentbitextension

import (
	"go.opentelemetry.io/collector/config/configmodels"
)

// Config has the configuration for the fluentbit extension.
type Config struct {
	configmodels.ExtensionSettings `mapstructure:",squash"`

	// The TCP `host:port` to which the subprocess should send log entries.
	// This is required unless you are overridding `args` and providing the
	// output configuration yourself either in `args` or `config`.
	TCPEndpoint string `mapstructure:"tcp_endpoint"`

	// The path to the executable for FluentBit. Ideally should be an absolute
	// path since the CWD of the collector is not guaranteed to be stable.
	ExecutablePath string `mapstructure:"executable_path"`

	// Exec arguments to the FluentBit process.  If you provide this, none of
	// the standard args will be set, and only these provided args will be
	// passed to FluentBit.  The standard args will set the flush interval to 1
	// second, configure the forward output with the given `tcp_endpoint`
	// option, enable the HTTP monitoring server in FluentBit, and set the
	// config file to stdin. The only required arg is `--config=/dev/stdin`,
	// since this extension passes the provided config to FluentBit via stdin.
	// If you set args manually, you will be responsible for setting the
	// forward output to the right port for the fluentforward receiver. See
	// `process.go#constructArgs` of this extension source to see the current
	// default args.
	Args []string `mapstructure:"args"`

	// A configuration for FluentBit.  This is the text content of the config
	// itself, not a path to a config file.
	Config string `mapstructure:"config"`
}
