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

package remotesamplingextension

import (
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

// Config has the configuration settings for the remote sampling extension,
// used to fetch sampling configuration from the upstream Jaeger collector instance.
//
// It is an extension of configmodels.ExtensionSettings
type Config struct {
	configmodels.ExtensionSettings `mapstructure:",squash"`

	// Port is the port used to publish the health check status.
	// Default is `5778` (https://www.jaegertracing.io/docs/1.15/deployment/#agent).
	Port uint16 `mapstructure:"port"`

	// Addr is the upstream Jaeger collector address that can be used to fetch
	// sampling configurations. The default value is `:14250`.
	Addr string `mapstructure:"addr"`
}
