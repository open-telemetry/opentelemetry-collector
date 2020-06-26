// Copyright The OpenTelemetry Authors
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

package configprotocol

import (
	"go.opentelemetry.io/collector/config/configtls"
)

// ProtocolServerSettings defines common settings for a single-protocol server configuration.
// Specific receivers or exporters can embed this struct and extend it with more fields if needed.
type ProtocolServerSettings struct {
	// Endpoint configures the endpoint in the format 'address:port' for the receiver.
	// The default value is set by the receiver populating the struct.
	Endpoint string `mapstructure:"endpoint"`

	// Configures the protocol to use TLS.
	// The default value is nil, which will cause the protocol to not use TLS.
	TLSCredentials *configtls.TLSServerSetting `mapstructure:"tls_credentials, omitempty"`
}
