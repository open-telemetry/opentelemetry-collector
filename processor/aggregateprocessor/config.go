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

package aggregateprocessor

import (
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

// Config holds configuration settings for this processor to discover collector peers
// using any of the available service discovery mechanisms
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// DNS name used to discover peers
	PeerDiscoveryDNSName string `mapstructure:"peer_discovery_dns_name"`

	// peer port to forward spans (oc exporter right now)
	PeerPort int `mapstructure:"peer_port"`
}
